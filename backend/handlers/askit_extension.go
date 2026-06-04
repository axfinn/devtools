package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	askitCache     []byte
	askitCacheTime time.Time
	askitCacheCommit string // short SHA + subject of the cloned HEAD
	askitCacheMu   sync.Mutex
	askitCacheTTL  = 10 * time.Minute
)

const (
	askitRepoSSH = "git@github.com:axfinn/Askit.git"
	askitBranch  = "main"
)

// AskitExtensionDownload serves the AskIt Chrome extension zip from the cached
// build (refreshed by AskitExtensionRefresh or the 10-min TTL). This is the only
// download entry point — the "更新最新" button no longer downloads, it refreshes.
// GET /api/askit/extension
func (h *AIGatewayHandler) AskitExtensionDownload(c *gin.Context) {
	forceRefresh := c.Query("refresh") == "1"
	data, _, err := getAskitZipCached(forceRefresh)
	if err == nil {
		c.Header("Content-Disposition", "attachment; filename=askit-extension.zip")
		c.Header("Content-Type", "application/zip")
		c.Data(http.StatusOK, "application/zip", data)
		return
	}

	// Fallback: local static zip from Docker build
	zipPath := filepath.Join(".", "data", "askit", "askit-extension.zip")
	if _, statErr := os.Stat(zipPath); statErr == nil {
		c.Header("Content-Disposition", "attachment; filename=askit-extension.zip")
		c.File(zipPath)
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取扩展包失败: %v", err)})
}

// AskitExtensionRefresh forces a fresh clone (bypassing the cache) and reports
// which commit the extension is now built from, WITHOUT returning the zip. The
// frontend "更新最新" button calls this; users still download via AskitExtensionDownload.
// POST /api/askit/extension/refresh
func (h *AIGatewayHandler) AskitExtensionRefresh(c *gin.Context) {
	_, commit, err := getAskitZipCached(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("刷新失败: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":        true,
		"commit":    commit,
		"branch":    askitBranch,
		"refreshed": time.Now().Format("2006-01-02 15:04:05"),
	})
}

func getAskitZipCached(forceRefresh bool) ([]byte, string, error) {
	askitCacheMu.Lock()
	defer askitCacheMu.Unlock()

	if !forceRefresh && askitCache != nil && time.Since(askitCacheTime) < askitCacheTTL {
		return askitCache, askitCacheCommit, nil
	}

	data, commit, err := buildZipFromGitHub()
	if err != nil {
		return nil, "", err
	}

	askitCache = data
	askitCacheTime = time.Now()
	askitCacheCommit = commit
	return data, commit, nil
}

// askitSSHCommand builds a GIT_SSH_COMMAND that reuses the persistent autodev
// SSH key (already registered on GitHub) so the Askit clone authenticates without
// a separate credential. GitHub rejects anonymous SSH even for public repos, so
// reusing autodev's key is what makes this work. Returns "" if no key exists yet.
func askitSSHCommand() string {
	dataDir := os.Getenv("AUTODEV_DATA_DIR")
	if dataDir == "" {
		dataDir = filepath.Join(".", "data", "autodev")
	}
	sshDir := filepath.Join(dataDir, "ssh")
	keyPath := filepath.Join(sshDir, "id_ed25519")
	if _, err := os.Stat(keyPath); err != nil {
		return ""
	}
	base := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=accept-new -o BatchMode=yes", keyPath)
	if knownHosts := filepath.Join(sshDir, "known_hosts"); fileExists(knownHosts) {
		base += fmt.Sprintf(" -o UserKnownHostsFile=%s", knownHosts)
	}
	return base
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// buildZipFromGitHub shallow-clones Askit over SSH and zips its dist/ directory
// in memory. SSH bypasses the REST API entirely, so there is no 60/hour rate
// limit. Returns the zip bytes and a short "<sha> <subject>" version string.
func buildZipFromGitHub() ([]byte, string, error) {
	tmpDir, err := os.MkdirTemp("", "askit-clone-")
	if err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", askitBranch, askitRepoSSH, tmpDir)
	if sshCmd := askitSSHCommand(); sshCmd != "" {
		cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCmd)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, "", fmt.Errorf("git clone failed: %v: %s", err, string(out))
	}

	commit := askitClonedCommit(tmpDir)

	distDir := filepath.Join(tmpDir, "dist")
	if _, err := os.Stat(distDir); err != nil {
		return nil, "", fmt.Errorf("dist/ not found in cloned repo")
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	walkErr := filepath.Walk(distDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(distDir, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		_, err = w.Write(content)
		return err
	})
	if walkErr != nil {
		return nil, "", walkErr
	}
	if err := zw.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), commit, nil
}

// askitClonedCommit returns a short "<sha> <subject>" description of the cloned
// HEAD, used to tell the user which version the extension was refreshed to.
func askitClonedCommit(repoDir string) string {
	cmd := exec.Command("git", "-C", repoDir, "log", "-1", "--format=%h %s")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return string(bytes.TrimSpace(out))
}

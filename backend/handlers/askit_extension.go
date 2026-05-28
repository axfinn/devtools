package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	askitCache     []byte
	askitCacheTime time.Time
	askitCacheMu   sync.Mutex
	askitCacheTTL  = 10 * time.Minute
)

const (
	askitRepoOwner = "axfinn"
	askitRepoName  = "Askit"
	askitBranch    = "main"
)

// AskitExtensionDownload serves the latest AskIt Chrome extension zip.
// Priority: GitHub dist/ (always latest) -> local static file (Docker build time).
// GET /api/askit/extension
func (h *AIGatewayHandler) AskitExtensionDownload(c *gin.Context) {
	data, err := getAskitZipCached()
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

func getAskitZipCached() ([]byte, error) {
	askitCacheMu.Lock()
	defer askitCacheMu.Unlock()

	if askitCache != nil && time.Since(askitCacheTime) < askitCacheTTL {
		return askitCache, nil
	}

	data, err := buildZipFromGitHub()
	if err != nil {
		return nil, err
	}

	askitCache = data
	askitCacheTime = time.Now()
	return data, nil
}

// buildZipFromGitHub fetches the dist/ tree from GitHub API and builds a zip in memory
func buildZipFromGitHub() ([]byte, error) {
	treeURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", askitRepoOwner, askitRepoName, askitBranch)
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(treeURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub tree API: %d", resp.StatusCode)
	}

	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
			URL  string `json:"url"`
			Size int    `json:"size"`
		} `json:"tree"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		return nil, err
	}

	// Filter dist/ files
	type fileEntry struct {
		path string
		url  string
	}
	var files []fileEntry
	for _, item := range tree.Tree {
		if item.Type == "blob" && len(item.Path) > 5 && item.Path[:5] == "dist/" {
			files = append(files, fileEntry{path: item.Path[5:], url: item.URL})
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no dist/ files found in repo")
	}

	// Build zip
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, f := range files {
		content, err := fetchGitHubBlob(client, f.url)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", f.path, err)
		}
		w, err := zw.Create(f.path)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(content); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// fetchGitHubBlob downloads a blob's content via GitHub Git Blob API
func fetchGitHubBlob(client *http.Client, blobURL string) ([]byte, error) {
	req, _ := http.NewRequest("GET", blobURL, nil)
	req.Header.Set("Accept", "application/vnd.github.raw+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("blob API: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

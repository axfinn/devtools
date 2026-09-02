package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSafeResolveWorkDir 表驱动覆盖 SafeResolveWorkDir 的白名单 / 符号链接 / 边界场景。
// 防御 prompt injection + admin pwd 撞库造成的内网/根目录代码执行:
//   - filepath.Abs 防 ../ 跳出
//   - filepath.EvalSymlinks 防软链绕过白名单
//   - 前缀匹配末尾加 Separator 防 /data/foo 误包含 /data/foobar
func TestSafeResolveWorkDir(t *testing.T) {
	base := t.TempDir()
	allowed := filepath.Join(base, "workdir")
	outside := filepath.Join(base, "outside")
	// foobared 必须真实存在才能跑到前缀校验那一行;否则 EvalSymlinks 先报 "not exist"
	foobared := filepath.Join(base, "foobared")

	for _, d := range []string{allowed, outside, foobared} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// 在 allowed 目录里放一个指向 /tmp 的软链(符号链接绕过攻击向量)
	linkInAllowed := filepath.Join(allowed, "escape")
	if err := os.Symlink(os.TempDir(), linkInAllowed); err != nil {
		// macOS 上 /var/folders/... 是 symlink 链,这里若失败就跳过
		t.Skipf("skip symlink test: %v", err)
	}

	tests := []struct {
		name         string
		reqPath      string
		allowed      []string
		wantErr      bool
		wantResolved string // "" 表示只校验 err
	}{
		{
			name:         "合法路径在白名单内",
			reqPath:      allowed,
			allowed:      []string{allowed},
			wantErr:      false,
			wantResolved: allowed,
		},
		{
			name:    "路径在白名单外",
			reqPath: outside,
			allowed: []string{allowed},
			wantErr: true,
		},
		{
			name:    "符号链接绕过:allowed 内软链指向 /tmp,解链后落到白名单外",
			reqPath: linkInAllowed,
			allowed: []string{allowed},
			wantErr: true,
		},
		{
			name:    "reqPath 为空",
			reqPath: "",
			allowed: []string{allowed},
			wantErr: true,
		},
		{
			name:         "allowed 为空切片:无限制模式,允许任意路径",
			reqPath:      outside,
			allowed:      nil,
			wantErr:      false,
			wantResolved: outside,
		},
		{
			name:         "allowed 为空切片但传入白名单外的相对路径,仍会被 Abs+EvalSymlinks 规范化",
			reqPath:      outside,
			allowed:      []string{},
			wantErr:      false,
			wantResolved: outside,
		},
		{
			name:    "路径不存在,触发 EvalSymlinks 报错",
			reqPath: filepath.Join(base, "no-such-dir"),
			allowed: []string{base},
			wantErr: true,
		},
		{
			// 关键:foobar 不应误包含在 foo 白名单内,因为 prefix 是 foo + "/" 才会匹配
			name:    "前缀误包含:foobared 不应误包含在 foo 白名单内(末尾 Separator 防御)",
			reqPath: foobared,
			allowed: []string{filepath.Join(base, "foo")},
			wantErr: true,
		},
		{
			// 正向对照:foo/sub 才应匹配 foo 白名单
			name:         "前缀匹配:foo/sub 应通过 foo 白名单",
			reqPath:      filepath.Join(base, "foo", "sub"),
			allowed:      []string{filepath.Join(base, "foo")},
			wantErr:      false,
			wantResolved: filepath.Join(base, "foo", "sub"),
		},
	}
	// foo 目录不存在的话 EvalSymlinks 会先报"no such file"导致无法走到前缀匹配分支,
	// 单独 MkdirAll 让 prefix 也能解链。
	if err := os.MkdirAll(filepath.Join(base, "foo", "sub"), 0o755); err != nil {
		t.Fatalf("mkdir foo/sub: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeResolveWorkDir(tt.reqPath, tt.allowed)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("SafeResolveWorkDir(%q, %v) 期望报错,却返回 %q", tt.reqPath, tt.allowed, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("SafeResolveWorkDir(%q, %v) 报错: %v", tt.reqPath, tt.allowed, err)
			}
			// macOS 上 t.TempDir() 返回的 /var/folders/... 是 /private/var/folders/... 的 symlink,
			// EvalSymlinks 会规范化;wantResolved 也走同样的规范化以公平比对。
			want := tt.wantResolved
			if want != "" {
				if rw, err := filepath.EvalSymlinks(want); err == nil {
					want = rw
				}
			}
			if want != "" && got != want {
				t.Errorf("SafeResolveWorkDir(%q) = %q, want %q", tt.reqPath, got, want)
			}
			// 任何成功返回必须是绝对路径
			if !filepath.IsAbs(got) {
				t.Errorf("SafeResolveWorkDir 返回非绝对路径: %q", got)
			}
		})
	}
}

// TestSafeResolveWorkDir_AllowedRelativePrefix 确认 allowed 给的是相对路径时,
// 内部统一 Abs 后再前缀匹配,不会因为 /abs 与 /abs/foobar 不匹配而误判。
func TestSafeResolveWorkDir_AllowedRelativePrefix(t *testing.T) {
	base := t.TempDir()
	workdir := filepath.Join(base, "project")
	subdir := filepath.Join(workdir, "src")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// chdir 到 base 再用纯相对路径,绕开 macOS /var -> /private/var symlink
	// 让 filepath.Rel(".", x) 在临时目录内一致(否则 base 是 symlink 而 "." 是真实路径,Rel 拒)。
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(base); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origCwd) })

	relAllowed := "project"

	got, err := SafeResolveWorkDir("project/src", []string{relAllowed})
	if err != nil {
		t.Fatalf("SafeResolveWorkDir 期望通过, 实际: %v", err)
	}
	// macOS 上 base 是 /var/folders/... symlink,EvalSymlinks 后是 /private/var/folders/...;
	// 比对 want 时也走同样的规范化。
	wantPrefix := workdir
	if rw, err := filepath.EvalSymlinks(wantPrefix); err == nil {
		wantPrefix = rw
	}
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("resolved %q 应是 %q 的前缀", got, wantPrefix)
	}
}

// TestSafeResolveWorkDir_MultipleAllowedPrefixes 确认多个白名单项时,
// 只要 reqPath 命中任一即可。
func TestSafeResolveWorkDir_MultipleAllowedPrefixes(t *testing.T) {
	base := t.TempDir()
	allowed1 := filepath.Join(base, "proj1")
	allowed2 := filepath.Join(base, "proj2")
	target := filepath.Join(allowed1, "deep", "nested")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(allowed2, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := SafeResolveWorkDir(target, []string{allowed2, allowed1})
	if err != nil {
		t.Fatalf("SafeResolveWorkDir 期望通过(命中 proj1), 实际: %v", err)
	}
	want := target
	if rw, err := filepath.EvalSymlinks(want); err == nil {
		want = rw
	}
	if got != want {
		t.Errorf("resolved = %q, want %q", got, want)
	}
}

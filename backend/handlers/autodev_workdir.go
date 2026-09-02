package handlers

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SafeResolveWorkDir 把用户提供的 workDir 路径规范化(绝对路径 + 解符号链接),
// 并要求最终路径落在 allowed 白名单前缀之内。返回可安全用作 cmd.Dir 的绝对路径。
//
// 防御 prompt injection + admin pwd 撞库造成的内网/根目录代码执行:
//   - filepath.Abs: 相对路径→绝对路径,防止通过 ../ 跳出预期目录
//   - filepath.EvalSymlinks: 解符号链接,防止创建软链绕过白名单
//   - 前缀匹配(末尾强制加 filepath.Separator): 防止 /data/foo 误包含 /data/foobar
//
// allowed 为空时,允许任意路径(默认放行,仅用于本地单租户部署)。
// 生产必须配置 AutoDevConfig.AllowedWorkDirs。
//
// 注意:macOS 上 /var 是 /private/var 的 symlink,所以 allowed 列表项也会走同样的 Abs+EvalSymlinks 规范化,
// 否则 `/var/foo` 白名单匹配不上 `EvalSymlinks` 返回的 `/private/var/foo`。
func SafeResolveWorkDir(reqPath string, allowed []string) (string, error) {
	if reqPath == "" {
		return "", fmt.Errorf("workdir 不能为空")
	}
	abs, err := filepath.Abs(reqPath)
	if err != nil {
		return "", fmt.Errorf("workdir 路径无效: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("workdir 不可访问: %w", err)
	}
	if len(allowed) == 0 {
		return resolved, nil
	}
	for _, prefix := range allowed {
		// 规范化与 reqPath 一致:Abs + EvalSymlinks。
		// 若 prefix 指向不存在的路径,fallback 到只 Abs(允许"声明未来才会创建的目录")。
		absPref, err := filepath.Abs(prefix)
		if err != nil {
			continue
		}
		resolvedPref, err := filepath.EvalSymlinks(absPref)
		if err != nil {
			resolvedPref = absPref
		}
		if resolved == resolvedPref || strings.HasPrefix(resolved, resolvedPref+string(filepath.Separator)) {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("workdir %q 不在白名单内", reqPath)
}

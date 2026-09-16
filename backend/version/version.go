// Package version 提供构建期注入的版本标识与三级 fallback。
//
// 解析优先级：ldflags 注入 > DEVTOOLS_VERSION 环境变量 > 字面量 "dev"。
// 独立小包（无内部依赖），handlers 与 main 都可读取，不产生循环依赖。
package version

import (
	"os"
	"strings"
)

// 三个变量由构建期 -ldflags "-X devtools/version.Version=..." 注入。
// 未注入时保持下面的字面量，Resolve 负责兜底保证非空。
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// Resolve 返回三级 fallback 后的版本串，保证返回值非空。
// 额外提供 DEVTOOLS_VERSION 一级，是为了让当前没有 ldflags 的本地构建
// （deploy.sh:388 `go build -o server main.go`）也能拿到真版本：
//
//	DEVTOOLS_VERSION=$(git rev-parse --short HEAD) ./deploy.sh
func Resolve() string {
	if Version != "" && Version != "dev" {
		return Version
	}
	if v := strings.TrimSpace(os.Getenv("DEVTOOLS_VERSION")); v != "" {
		return v
	}
	return "dev"
}

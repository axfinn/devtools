package main

// =====================================================================
//  FINN-94 Stage 5 · ssrules 文档口径 / 接线纪律补测
//
//  补 v3 方案 §6.3 里两条**静态断言**的覆盖缺口:
//
//    A17  AGENTS.md 口径:模块地图表格行数 == 标题里的数字
//         == 正文散文里引用的数字(裁决 4 写死)
//    A20  取件纪律:backend/routes.go 必须 gofmt -s 干净;
//         且 ssrules 的管理端**真的接上了线**(struct 字段 + 赋值 +
//         RegisterSSRuleRoutes 调用),不能只在自己的测试里手搓 RouteHandlers
//
//  两条都用 AST / 文本解析实现,不用 grep:断言锚在具体语法节点上,
//  改动实现(删字段、删赋值、删注册、改文档数字)必须立刻变红。
// =====================================================================

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"testing"
)

// ssRuleQAAgentsDocPath 是仓库根的 AGENTS.md(相对 backend/ 目录)。
const ssRuleQAAgentsDocPath = "../AGENTS.md"

// ssRuleQARoutesFilePath / ssRuleQARoutesIndexPath 是「接线」所在的两个文件。
const (
	ssRuleQARoutesFilePath      = "routes.go"
	ssRuleQARoutesIndexPath     = "routes/index.go"
	ssRuleQARouteHandlersType   = "routes.RouteHandlers"
	ssRuleQARouteHandlersField  = "SSRuleHandler"
	ssRuleQARouteHandlerVar     = "ssRuleHandler"
	ssRuleQARegisterFunc        = "RegisterSSRuleRoutes"
	ssRuleQARouteHandlersStruct = "routeHandlers"
)

// ---------------------------------------------------------------------
// A17 · AGENTS.md 模块地图口径
// ---------------------------------------------------------------------

// TestSSRule_AGENTSModuleMapCountIsConsistent A17(裁决 4):
//
//	grep -cE '^\| [0-9]+ \|' AGENTS.md 的结果
//	  == grep -m1 -E '^## [0-9]+ 模块地图' 里的数字
//	  == 正文里引用的「N 模块地图」的数字
//
// 三处必须一致 —— 少写一行表格、或只改标题不改表格,模块地图就与实际模块数
// 对不上(侧边栏/文档都按这张表走)。
//
// 反事实:只加一行模块表格、标题不改 → 本用例必须红(43 ≠ 44)。
func TestSSRule_AGENTSModuleMapCountIsConsistent(t *testing.T) {
	raw, err := os.ReadFile(ssRuleQAAgentsDocPath)
	if err != nil {
		t.Skipf("读不到 %s(非仓库布局下跳过,不算通过): %v", ssRuleQAAgentsDocPath, err)
	}
	doc := string(raw)

	// ① 表格行:形如 "| 12 | **ShortURL 短链** | ..."
	rowRe := regexp.MustCompile(`(?m)^\|\s*([0-9]+)\s*\|`)
	rowNums := map[int]bool{}
	rowCount := 0
	for _, m := range rowRe.FindAllStringSubmatch(doc, -1) {
		n, convErr := strconv.Atoi(m[1])
		if convErr != nil {
			t.Fatalf("模块表格行号不是数字: %q", m[1])
		}
		if rowNums[n] {
			t.Errorf("模块表格里出现了重复的行号 %d", n)
		}
		rowNums[n] = true
		rowCount++
	}
	if rowCount == 0 {
		t.Fatalf("%s 里没有匹配 '^| N |' 的模块表格行", ssRuleQAAgentsDocPath)
	}

	// 表格行号必须是 1..N 连续(漏号/重号都要红)。
	for i := 1; i <= rowCount; i++ {
		if !rowNums[i] {
			t.Errorf("模块表格缺少第 %d 行(行号不连续,共 %d 行)", i, rowCount)
		}
	}

	// ② 标题:## N 模块地图
	titleRe := regexp.MustCompile(`(?m)^##\s*([0-9]+)\s*模块地图`)
	titleMatch := titleRe.FindStringSubmatch(doc)
	if titleMatch == nil {
		t.Fatalf("%s 里找不到 '## N 模块地图' 标题", ssRuleQAAgentsDocPath)
	}
	titleNum, _ := strconv.Atoi(titleMatch[1])

	// ③ 正文里引用的「N 模块地图」(例如「更新本文件 `43 模块地图` 表格」)
	refRe := regexp.MustCompile(`([0-9]+)\s*模块地图`)
	refs := refRe.FindAllStringSubmatch(doc, -1)
	if len(refs) == 0 {
		t.Fatalf("%s 里找不到任何「N 模块地图」引用", ssRuleQAAgentsDocPath)
	}
	for _, r := range refs {
		n, _ := strconv.Atoi(r[1])
		if n != titleNum {
			t.Errorf("正文里的「%s 模块地图」与标题「## %d 模块地图」不一致", r[1], titleNum)
		}
	}

	if rowCount != titleNum {
		t.Errorf("模块表格实际 %d 行,标题写的是 %d —— 口径不一致(只加行不改标题就会这样)", rowCount, titleNum)
	}
}

// ---------------------------------------------------------------------
// A20 · 取件纪律 + 真实接线
// ---------------------------------------------------------------------

// TestSSRule_RoutesFileIsGofmtSimplifyClean A20:
// backend/routes.go 必须 `gofmt -s -l` 输出为空(方案 §5.2.1 特别点名这个文件
// 之前 gofmt 不干净,取件后要求单独 gofmt -w)。
//
// 反事实:往 routes.go 里塞一段可被 -s 简化的代码 → 本用例必须红。
func TestSSRule_RoutesFileIsGofmtSimplifyClean(t *testing.T) {
	gofmtPath := ssRuleQAFindGofmt(t)

	for _, f := range []string{ssRuleQARoutesFilePath, ssRuleQARoutesIndexPath} {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("文件不存在: %s (%v)", f, err)
		}
		out, err := exec.Command(gofmtPath, "-s", "-l", f).CombinedOutput()
		if err != nil {
			t.Fatalf("gofmt -s -l %s 执行失败: %v\n%s", f, err, out)
		}
		if len(out) != 0 {
			t.Errorf("%s 没有 gofmt -s 干净(gofmt -s -l 有输出):\n%s", f, out)
		}
	}
}

// ssRuleQAFindGofmt 找 gofmt 可执行文件:先看 PATH,再看 GOROOT/bin。
// 两个都找不到时跳过(而不是判红)—— 环境问题不该算实现缺陷。
func ssRuleQAFindGofmt(t *testing.T) string {
	t.Helper()
	if p, err := exec.LookPath("gofmt"); err == nil {
		return p
	}
	candidate := filepath.Join(runtime.GOROOT(), "bin", "gofmt")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	t.Skipf("找不到 gofmt(PATH 与 GOROOT/bin 都没有),跳过 gofmt -s 检查")
	return ""
}

// TestSSRule_AdminRoutesAreWiredInRoutesFile A20 的延伸(取件纪律):
// ssrules 的管理端必须**真的接上了线**,而不是只在单测里手搓 RouteHandlers:
//
//	① backend/routes.go 的 routeHandlers 结构体里有 ssRuleHandler 字段
//	② setupRoutes 里给 routes.RouteHandlers 赋了 SSRuleHandler: h.ssRuleHandler
//	③ backend/routes/index.go 的 RegisterAllRoutes 里调了 RegisterSSRuleRoutes
//
// 少任何一环,线上 /api/ssrules/* 就不存在(而单测仍然全绿)。
//
// 反事实:删掉 ② 的赋值行 → 本用例必须红。
func TestSSRule_AdminRoutesAreWiredInRoutesFile(t *testing.T) {
	fset := token.NewFileSet()

	// ① + ②:backend/routes.go
	file, err := parser.ParseFile(fset, ssRuleQARoutesFilePath, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("解析 %s 失败: %v", ssRuleQARoutesFilePath, err)
	}

	fieldOK := false
	for _, decl := range file.Decls {
		gen, isGen := decl.(*ast.GenDecl)
		if !isGen {
			continue
		}
		for _, spec := range gen.Specs {
			ts, isTS := spec.(*ast.TypeSpec)
			if !isTS || ts.Name.Name != ssRuleQARouteHandlersStruct {
				continue
			}
			st, isStruct := ts.Type.(*ast.StructType)
			if !isStruct {
				continue
			}
			for _, f := range st.Fields.List {
				for _, name := range f.Names {
					if name.Name == ssRuleQARouteHandlerVar {
						fieldOK = true
						if got := types.ExprString(f.Type); got != "*handlers.SSRuleHandler" {
							t.Errorf("%s 里 %s 的类型想要 *handlers.SSRuleHandler,得到 %s",
								ssRuleQARoutesFilePath, ssRuleQARouteHandlerVar, got)
						}
					}
				}
			}
		}
	}
	if !fieldOK {
		t.Errorf("%s 的 %s 结构体里没有 %s 字段 —— 管理端根本没接上线",
			ssRuleQARoutesFilePath, ssRuleQARouteHandlersStruct, ssRuleQARouteHandlerVar)
	}

	// ② setupRoutes 里的 routes.RouteHandlers{...} 字面量必须带
	//    SSRuleHandler: h.ssRuleHandler
	assignOK := false
	for _, decl := range file.Decls {
		fn, isFn := decl.(*ast.FuncDecl)
		if !isFn || fn.Name.Name != "setupRoutes" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			cl, isCL := n.(*ast.CompositeLit)
			if !isCL {
				return true
			}
			if types.ExprString(cl.Type) != ssRuleQARouteHandlersType {
				return true
			}
			for _, elt := range cl.Elts {
				kv, isKV := elt.(*ast.KeyValueExpr)
				if !isKV {
					continue
				}
				key, isIdent := kv.Key.(*ast.Ident)
				if !isIdent || key.Name != ssRuleQARouteHandlersField {
					continue
				}
				assignOK = true
				if got := types.ExprString(kv.Value); got != "h."+ssRuleQARouteHandlerVar {
					t.Errorf("setupRoutes 里 %s 的值想要 h.%s,得到 %s",
						ssRuleQARouteHandlersField, ssRuleQARouteHandlerVar, got)
				}
			}
			return true
		})
	}
	if !assignOK {
		t.Errorf("setupRoutes 里没有给 %s 赋值 %s: h.%s —— 管理端路由不会被注册",
			ssRuleQARouteHandlersType, ssRuleQARouteHandlersField, ssRuleQARouteHandlerVar)
	}

	// ③ backend/routes/index.go 的 RegisterAllRoutes 里必须调 RegisterSSRuleRoutes
	indexFile, err := parser.ParseFile(fset, ssRuleQARoutesIndexPath, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("解析 %s 失败: %v", ssRuleQARoutesIndexPath, err)
	}
	registerOK := false
	for _, decl := range indexFile.Decls {
		fn, isFn := decl.(*ast.FuncDecl)
		if !isFn {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, isCall := n.(*ast.CallExpr)
			if !isCall {
				return true
			}
			if ident, isIdent := call.Fun.(*ast.Ident); isIdent && ident.Name == ssRuleQARegisterFunc {
				registerOK = true
			}
			return true
		})
	}
	if !registerOK {
		t.Errorf("%s 里没有调用 %s —— 管理端 4 个端点不会被注册",
			ssRuleQARoutesIndexPath, ssRuleQARegisterFunc)
	}
}

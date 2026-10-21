package handlers

// =====================================================================
//  MiniMax MCP 启动参数归一化单测 (FINN-56 Stage 5)
//
//  被测对象(Stage 3 实现,分支 agent/devtools-backend/finn-59):
//    backend/handlers/image_understanding.go
//      normalizeMCPArgs / indexOfMinimaxMCPPackage / isMinimaxMCPPackageSpec
//
//  背景:
//    官方调用形式是 `uvx minimax-coding-plan-mcp -y`,`-y` 必须紧跟包名。
//    存量 config.yaml 里写的是 `args: ["minimax-coding-plan-mcp"]`(缺 -y),
//    旧兜底 `if len(args)==0` 几乎永不触发 → 默认路径全部绕过 → MCP 起不来。
//    运行期归一化把缺失的 -y 补上,但必须:
//      1) 只对 minimax 包生效(不误伤自定义 MCP Server)
//      2) 按【位置】判定「已含 -y」(包名之前的是 uv 层开关,会被 uv 消费掉)
//      3) 幂等(已含则不重复注入)
//      4) 不与入参共享底层数组(config.Get() 全局共享 *Config,原地 append 会 data race)
//
//  判据来源:docs/plans/minimax-mcp-args-alignment.md §3.2 / §3.3 / §6 L2(T1–T17)。
// =====================================================================

import (
	"reflect"
	"testing"
)

// equalArgs 逐元素比较两个 args 切片(不只比 len)。
func equalArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// copyArgs 深拷贝一份,用于断言「入参未被修改」。
func copyArgs(a []string) []string {
	out := make([]string, len(a))
	copy(out, a)
	return out
}

// TestNormalizeMCPArgs 表驱动用例 T1–T17(每条都带断言,不许只跑不判)。
func TestNormalizeMCPArgs(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		// —— 规则 1:args 为空 → 官方默认调用 ——
		{"T1_nil_空切片_返回官方默认调用", nil, []string{"minimax-coding-plan-mcp", "-y"}},
		{"T2_空列表_返回官方默认调用", []string{}, []string{"minimax-coding-plan-mcp", "-y"}},

		// —— 规则 4:命中包名但包名之后缺 -y → 末尾补 ——
		{"T3_存量config主场景_缺-y_末尾补", []string{"minimax-coding-plan-mcp"}, []string{"minimax-coding-plan-mcp", "-y"}},

		// —— 规则 3:包名之后已含 -y → 原样不变(幂等) ——
		{"T4_已含-y_不重复注入", []string{"minimax-coding-plan-mcp", "-y"}, []string{"minimax-coding-plan-mcp", "-y"}},

		// —— B1 位置化:包名【之前】的 -y 是 uv 层开关,不算「已含」→ 末尾补 ——
		{"T5_uv位置短-y_末尾补不删原值", []string{"-y", "minimax-coding-plan-mcp"}, []string{"-y", "minimax-coding-plan-mcp", "-y"}},

		// —— U-3:版本限定写法命中包名 → 注入 ——
		{"T6_版本限定写法_命中并注入", []string{"minimax-coding-plan-mcp==0.0.4"}, []string{"minimax-coding-plan-mcp==0.0.4", "-y"}},

		// —— B4 门控:非 minimax 的自定义 MCP 调用 → 字节级不变 ——
		{"T7_非minimax自定义MCP_原样不变", []string{"@modelcontextprotocol/server-filesystem", "/data"}, []string{"@modelcontextprotocol/server-filesystem", "/data"}},

		// —— B3:空串元素不特殊处理,不加不减 ——
		{"T8_单个空串_原样不变", []string{""}, []string{""}},
		{"T9_包名后跟空串_末尾补", []string{"minimax-coding-plan-mcp", ""}, []string{"minimax-coding-plan-mcp", "", "-y"}},

		// —— B2:--yes 不算「已含」,任何位置都末尾补 -y ——
		{"T10_包名后跟--yes_末尾补", []string{"minimax-coding-plan-mcp", "--yes"}, []string{"minimax-coding-plan-mcp", "--yes", "-y"}},
		{"T11_uv位置长形式--yes_末尾补", []string{"--yes", "minimax-coding-plan-mcp"}, []string{"--yes", "minimax-coding-plan-mcp", "-y"}},

		// —— B1 位置化命中分支:包名【之后】的 -y 才算已含 → 原样不动(与 T4/T5 构成完整判据对) ——
		{"T12_包名之后已含-y_原样不变", []string{"minimax-coding-plan-mcp", "-y"}, []string{"minimax-coding-plan-mcp", "-y"}},

		// —— U-2:包名藏在 flag 值里 → 仍视为 minimax 调用 → 注入 ——
		{"T13_包名在--from值里_末尾补", []string{"--from", "minimax-coding-plan-mcp"}, []string{"--from", "minimax-coding-plan-mcp", "-y"}},
		{"T14_包名在--with值里_末尾补", []string{"--with", "minimax-coding-plan-mcp"}, []string{"--with", "minimax-coding-plan-mcp", "-y"}},

		// —— N-4:禁止无脑 HasPrefix,不得误命中同名第三方包 ——
		{"T15_同名第三方包后缀_不误命中原样不变", []string{"minimax-coding-plan-mcp-fork"}, []string{"minimax-coding-plan-mcp-fork"}},

		// —— 勘误轮新增:uv 层 -y 在 --from 之前 → 原样保留 + 末尾补 ——
		{"T17_uv层-y在--from之前_末尾补", []string{"-y", "--from", "minimax-coding-plan-mcp"}, []string{"-y", "--from", "minimax-coding-plan-mcp", "-y"}},

		// —— PM 拍板边界 2:extras 括号写法不在拍板分隔符集合内 → 不命中、不注入 ——
		{"B2_extras括号写法_不命中原样不变", []string{"minimax-coding-plan-mcp[extra]"}, []string{"minimax-coding-plan-mcp[extra]"}},

		// —— PM 拍板边界 4:args 非空 + 不含 minimax 包名 → 原样不变(B4 门控不命中) ——
		{"B4_uvx非minimax参数_原样不变", []string{"--python", "3.11", "some-other-mcp"}, []string{"--python", "3.11", "some-other-mcp"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			before := copyArgs(tc.in)

			got := normalizeMCPArgs(tc.in)

			if !equalArgs(got, tc.want) {
				t.Fatalf("normalizeMCPArgs(%#v)\n  期望 = %#v\n  实际 = %#v", tc.in, tc.want, got)
			}
			// 入参不得被就地修改(纯函数约束,§2.6 data race 前置条件)
			if !equalArgs(tc.in, before) {
				t.Fatalf("normalizeMCPArgs 就地修改了入参:\n  调用前 = %#v\n  调用后 = %#v", before, tc.in)
			}
		})
	}
}

// TestNormalizeMCPArgs_NoSharedBackingArray T16(必须):
// 返回值不得与入参共享底层数组 —— config.Get() 全局共享 *Config,
// 若在 cfg.Args 的底层数组上原地 append,其它 goroutine 读全局配置就会 data race。
func TestNormalizeMCPArgs_NoSharedBackingArray(t *testing.T) {
	// len == cap == 1 的切片:任何原地 append 都会直接写穿到共享数组
	in := []string{"minimax-coding-plan-mcp"}
	if len(in) != 1 || cap(in) != 1 {
		t.Fatalf("前置失败:入参切片应为 len==cap==1,实际 len=%d cap=%d", len(in), cap(in))
	}

	got := normalizeMCPArgs(in)

	// 1) 返回值内容正确
	if !equalArgs(got, []string{"minimax-coding-plan-mcp", "-y"}) {
		t.Fatalf("期望 = %#v,实际 = %#v", []string{"minimax-coding-plan-mcp", "-y"}, got)
	}

	// 2) 返回切片的首元素地址不得与入参首元素地址相同(不共享底层数组)
	if &got[0] == &in[0] {
		t.Fatalf("返回值与入参共享底层数组(&got[0] == &in[0]):会在全局共享配置上原地 append,导致 data race")
	}

	// 3) 在返回值上追加后,入参仍保持 1 个元素且内容不变
	extended := append(got, "EXTRA")
	if len(extended) != 3 {
		t.Fatalf("追加后长度 = %d,期望 3", len(extended))
	}
	if len(in) != 1 {
		t.Fatalf("追加返回值后入参长度被污染 = %d,期望仍为 1", len(in))
	}
	if in[0] != "minimax-coding-plan-mcp" {
		t.Fatalf("追加返回值后入参内容被污染 = %#v,期望 [\"minimax-coding-plan-mcp\"]", in)
	}
}

// TestNormalizeMCPArgs_NoInPlaceWriteToBackingArray T16 加强版:
//
// 方案 §6 L2 的 T16 用 len==cap==1 的切片 —— 但切片 cap==len 时 Go 的 append
// 本来就会另分配新数组,即使实现写错了(`append(cfg.Args, "-y")` 原地写)也测不出来。
// 真正会出事的是 cap > len 的切片(§2.6:yaml 反序列化出来的切片可能有富余容量,
// 此时 append 会原地写进全局共享的底层数组 → 与其它 goroutine 读 config.Get() 撞车)。
//
// 这里用 cap=4 的切片锁死「底层数组第 1 个槽位不得被写入」,并额外要求返回值不与
// 入参共享底层数组。
func TestNormalizeMCPArgs_NoInPlaceWriteToBackingArray(t *testing.T) {
	in := make([]string, 1, 4)
	in[0] = "minimax-coding-plan-mcp"
	if len(in) != 1 || cap(in) != 4 {
		t.Fatalf("前置失败:期望 len=1 cap=4,实际 len=%d cap=%d", len(in), cap(in))
	}

	got := normalizeMCPArgs(in)

	if !equalArgs(got, []string{"minimax-coding-plan-mcp", "-y"}) {
		t.Fatalf("期望 = %#v,实际 = %#v", []string{"minimax-coding-plan-mcp", "-y"}, got)
	}

	// 底层数组第 1 个槽位(入参 len 之外)必须仍是零值 —— 被写过说明是原地 append
	if slot := in[:cap(in)][1]; slot != "" {
		t.Fatalf("归一化在入参的底层数组上原地写入:底层数组[1] = %q,期望空(原地 append 会写穿到 config.Get() 全局共享数组)", slot)
	}

	// 返回值不得与入参共享底层数组
	if &got[0] == &in[0] {
		t.Fatalf("返回值与入参共享底层数组(&got[0] == &in[0]):会在全局共享配置上原地 append,导致 data race")
	}
}

// TestNormalizeMCPArgs_AlreadyContainsYesFlagIsIdempotent 幂等性:
// 同一个输入连续归一化两次,结果必须一致(不重复注入 -y)。
func TestNormalizeMCPArgs_AlreadyContainsYesFlagIsIdempotent(t *testing.T) {
	in := []string{"minimax-coding-plan-mcp"}
	once := normalizeMCPArgs(in)
	twice := normalizeMCPArgs(once)
	if !reflect.DeepEqual(once, twice) {
		t.Fatalf("归一化不幂等:\n  一次 = %#v\n  两次 = %#v", once, twice)
	}
	if len(twice) != 2 || twice[1] != "-y" {
		t.Fatalf("幂等结果异常 = %#v,期望 [\"minimax-coding-plan-mcp\",\"-y\"]", twice)
	}
}

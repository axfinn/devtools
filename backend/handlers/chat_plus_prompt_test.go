package handlers

import (
	"strings"
	"testing"
	"time"

	"devtools/utils"
)

func TestCleanOptimizedPrompt(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "去代码围栏",
			raw:  "```text\n一只橘猫趴在窗台上，午后侧光\n```",
			want: "一只橘猫趴在窗台上，午后侧光",
		},
		{
			name: "去提示词前缀",
			raw:  "提示词：一只橘猫趴在窗台上",
			want: "一只橘猫趴在窗台上",
		},
		{
			name: "去多层前缀与英文前缀",
			raw:  "Prompt: 优化后：a cat on the windowsill",
			want: "a cat on the windowsill",
		},
		{
			name: "去包裹引号",
			raw:  "“一只橘猫趴在窗台上”",
			want: "一只橘猫趴在窗台上",
		},
		{
			name: "多行压成一段",
			raw:  "一只橘猫\n\n趴在窗台上\n午后侧光",
			want: "一只橘猫 趴在窗台上 午后侧光",
		},
		{
			name: "空输出",
			raw:  "   \n\t ",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := cleanOptimizedPrompt(tc.raw); got != tc.want {
				t.Fatalf("cleanOptimizedPrompt(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestCleanOptimizedPromptTruncates(t *testing.T) {
	raw := strings.Repeat("猫", promptOptimizeMaxOutput+50)
	got := cleanOptimizedPrompt(raw)
	if n := len([]rune(got)); n != promptOptimizeMaxOutput {
		t.Fatalf("截断后长度 = %d, want %d", n, promptOptimizeMaxOutput)
	}
}

func TestNormalizePromptOptimizeMode(t *testing.T) {
	cases := []struct{ raw, prompt, image, want string }{
		{"", "画一只猫", "", promptModeT2I},
		{"", "改成夜景", "data:image/png;base64,x", promptModeEdit},
		{"", "", "data:image/png;base64,x", promptModeDescribe},
		{"EDIT", "", "", promptModeEdit},
		{"describe", "", "", promptModeDescribe},
		{"乱填", "", "data:image/png;base64,x", promptModeDescribe},
	}
	for _, tc := range cases {
		if got := normalizePromptOptimizeMode(tc.raw, tc.prompt, tc.image); got != tc.want {
			t.Fatalf("normalizePromptOptimizeMode(%q,%q,image=%v) = %q, want %q",
				tc.raw, tc.prompt, tc.image != "", got, tc.want)
		}
	}
}

// 改图模式的写法要求是这次改动的核心：「编辑指令 + 保持不变清单」，
// 少了它模型就会整图重绘，这正是原来「图生图效果差」的主因。
func TestBuildImagePromptOptimizeUserEditMode(t *testing.T) {
	got := buildImagePromptOptimizeUser("把背景换成海边", promptModeEdit, imagePromptOptimizeRequest{
		ImageModel: "Qwen-Image 2.1",
	})
	for _, must := range []string{"编辑指令", "保持不变", "把背景换成海边", "Qwen-Image 2.1", "只输出改写后的提示词正文"} {
		if !strings.Contains(got, must) {
			t.Fatalf("改图模式提示词缺少 %q:\n%s", must, got)
		}
	}
}

func TestBuildImagePromptOptimizeUserT2IMode(t *testing.T) {
	got := buildImagePromptOptimizeUser("一只猫", promptModeT2I, imagePromptOptimizeRequest{})
	for _, must := range []string{"文生图", "主体与动作", "一只猫"} {
		if !strings.Contains(got, must) {
			t.Fatalf("文生图模式提示词缺少 %q:\n%s", must, got)
		}
	}
}

// 只有参考图、没写要求时：要让模型先看懂图再写提示词，并且不许出现「如图所示」这类指代。
func TestBuildImagePromptOptimizeUserDescribeMode(t *testing.T) {
	got := buildImagePromptOptimizeUser("", promptModeDescribe, imagePromptOptimizeRequest{
		ImageFamily: comfyFamilyFlux2,
	})
	if !strings.Contains(got, "看懂这张图") {
		t.Fatalf("描述模式缺少看图要求:\n%s", got)
	}
	if !strings.Contains(got, "英文") {
		t.Fatalf("FLUX 家族无原话时应要求英文输出:\n%s", got)
	}
}

func TestImagePromptOptimizeSystemRules(t *testing.T) {
	sys := imagePromptOptimizeSystem()
	for _, must := range []string{"只输出提示词正文", "不要替换主体", "masterpiece"} {
		if !strings.Contains(sys, must) {
			t.Fatalf("系统提示词缺少 %q:\n%s", must, sys)
		}
	}
}

// ── 异步任务化的内存生命周期 ────────────────────────────────────────────────
// 这些用例只覆盖状态机本身（不碰上游模型）：任务表是包级 map，
// 读写在锁内、状态只能单向收敛，是「轮询永远卡在 running」这类问题的根源。

func putPromptOptimizeTaskForTest(t *testing.T, status string, createdAt time.Time) *promptOptimizeTask {
	t.Helper()
	task := &promptOptimizeTask{
		ID:        "popt_test_" + utils.GenerateHexKey(4),
		Status:    status,
		CreatedAt: createdAt,
	}
	promptOptimizeTasksMu.Lock()
	promptOptimizeTasks[task.ID] = task
	promptOptimizeTasksMu.Unlock()
	t.Cleanup(func() {
		promptOptimizeTasksMu.Lock()
		delete(promptOptimizeTasks, task.ID)
		promptOptimizeTasksMu.Unlock()
	})
	return task
}

func TestFinishPromptOptimizeTask(t *testing.T) {
	h := &AIGatewayHandler{}
	task := putPromptOptimizeTaskForTest(t, promptOptimizeStatusRunning, time.Now().Add(-3*time.Second))

	h.finishPromptOptimizeTask(task.ID, promptOptimizeStatusDone, "一只橘猫趴在窗台上", "MiniMax-M2.5", false, "")

	promptOptimizeTasksMu.Lock()
	got := *promptOptimizeTasks[task.ID]
	promptOptimizeTasksMu.Unlock()

	if got.Status != promptOptimizeStatusDone {
		t.Fatalf("status = %q, want done", got.Status)
	}
	if got.Prompt != "一只橘猫趴在窗台上" {
		t.Fatalf("prompt = %q", got.Prompt)
	}
	if got.Model != "MiniMax-M2.5" {
		t.Fatalf("model = %q", got.Model)
	}
	if got.ElapsedMS < 2000 {
		t.Fatalf("elapsed_ms = %d，应该覆盖任务已跑的时长", got.ElapsedMS)
	}
	if got.cancel != nil {
		t.Fatal("终态任务不该继续持有 cancel")
	}
}

// 取消接口只对 running 的任务生效：已终态的任务被打上 canceled 会让前端显示错状态
func TestPromptOptimizeCancelOnlyWhenRunning(t *testing.T) {
	h := &AIGatewayHandler{}

	done := putPromptOptimizeTaskForTest(t, promptOptimizeStatusDone, time.Now())
	_, _ = markPromptOptimizeCanceled(done.ID)
	if h.promptOptimizeTaskCanceled(done.ID) {
		t.Fatal("已完成的任务不该被标记为取消")
	}

	running := putPromptOptimizeTaskForTest(t, promptOptimizeStatusRunning, time.Now())
	_, _ = markPromptOptimizeCanceled(running.ID)
	if !h.promptOptimizeTaskCanceled(running.ID) {
		t.Fatal("运行中的任务应该被标记为取消")
	}
}

func TestPrunePromptOptimizeTasks(t *testing.T) {
	now := time.Now()
	expired := putPromptOptimizeTaskForTest(t, promptOptimizeStatusDone, now.Add(-2*time.Hour))
	expired.finishedAt = now.Add(-promptOptimizeTaskTTL - time.Minute)
	fresh := putPromptOptimizeTaskForTest(t, promptOptimizeStatusDone, now)
	fresh.finishedAt = now.Add(-time.Minute)
	running := putPromptOptimizeTaskForTest(t, promptOptimizeStatusRunning, now.Add(-2*time.Hour))

	promptOptimizeTasksMu.Lock()
	prunePromptOptimizeTasksLocked(now)
	_, expiredLeft := promptOptimizeTasks[expired.ID]
	_, freshLeft := promptOptimizeTasks[fresh.ID]
	_, runningLeft := promptOptimizeTasks[running.ID]
	promptOptimizeTasksMu.Unlock()

	if expiredLeft {
		t.Fatal("过期终态任务应被清理")
	}
	if !freshLeft {
		t.Fatal("未过期的终态任务不该被清理")
	}
	if !runningLeft {
		t.Fatal("运行中的任务无论多久都不能被清理（否则前端轮询会 404）")
	}
}

package middleware

// =====================================================================
//  FINN-94 Stage 5 · ssrules 监控面补测
//
//  A12 / Q1:规则文件的公开分发链接里带着**分发 token**,
//  写进 http_request_logs 等于把凭据抄进日志表
//  ⇒ /ss/ 前缀必须进 shouldSkipMonitoringRequest 的跳过名单。
//
//  反事实:把 "/ss/" 从跳过名单里删掉 → 本用例必须红。
// =====================================================================

import "testing"

// TestSSRule_MonitoringSkipsDistributionPaths A12/Q1:
// 分发请求不出现在监控日志里。
//
// 两个方向都要咬合:
//
//	① /ss/ 前缀 —— 必须跳过(路径里带分发 token)
//	② 管理端 /api/ssrules —— **必须不跳过**(跳过的话监控面就少了一块,
//	   而且这条断言能证明本用例不是「一律跳过就绿」的恒真闸门)
func TestSSRule_MonitoringSkipsDistributionPaths(t *testing.T) {
	skipped := []string{
		"/ss/3f2a9c1d/rules.acl",
		"/ss/3f2a9c1d/rules.txt",
		"/ss/3f2a9c1d/rules.list",
		"/ss/3f2a9c1d/rules.conf",
		"/ss/3f2a9c1d/",
		"/ss/3f2a9c1d",
	}
	for _, p := range skipped {
		if !shouldSkipMonitoringRequest("GET", p, "") {
			t.Errorf("%s 应当跳过监控(路径里带着分发 token)", p)
		}
	}

	notSkipped := []string{
		"/api/ssrules",
		"/api/ssrules/verify",
		"/api/ssrules/regenerate-token",
		"/paste",
		"/",
	}
	for _, p := range notSkipped {
		if shouldSkipMonitoringRequest("GET", p, "") {
			t.Errorf("%s 不该被跳过监控", p)
		}
	}
}

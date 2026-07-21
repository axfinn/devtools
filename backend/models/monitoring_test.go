package models

import (
	"path/filepath"
	"testing"
	"time"
)

func newMonitoringTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := NewDB(filepath.Join(t.TempDir(), "monitoring.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InitMonitoring(); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestArchiveHTTPRequestLogsPreservesAggregatesBeforeDelete(t *testing.T) {
	db := newMonitoringTestDB(t)
	now := time.Now().UTC()
	items := []*HTTPRequestLog{
		{Method: "GET", Path: "/api/old", Route: "/api/old", StatusCode: 200, ClientIP: "10.0.0.1", LatencyMS: 10, CreatedAt: now.AddDate(0, 0, -40)},
		{Method: "GET", Path: "/api/recent", Route: "/api/recent", StatusCode: 500, ClientIP: "10.0.0.2", LatencyMS: 30, CreatedAt: now.Add(-2 * time.Hour)},
	}
	for _, item := range items {
		if err := db.CreateHTTPRequestLog(item); err != nil {
			t.Fatal(err)
		}
	}

	result, err := db.ArchiveHTTPRequestLogs(now, 30, 730)
	if err != nil {
		t.Fatal(err)
	}
	if result.ArchivedDetails != 2 {
		t.Fatalf("archived=%d, want 2", result.ArchivedDetails)
	}
	if result.DeletedDetails != 1 {
		t.Fatalf("deleted=%d, want 1", result.DeletedDetails)
	}

	stats, err := db.GetMonitoringStorageStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.DetailRows != 1 || stats.HourlyRows != 2 || stats.DailyVisitorRows != 2 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	overview, err := db.GetMonitoringOverview(now.AddDate(0, 0, -60))
	if err != nil {
		t.Fatal(err)
	}
	if overview.RequestCount != 2 || overview.UniqueIPs != 2 || overview.Status5xx != 1 {
		t.Fatalf("unexpected overview: %+v", overview)
	}
}

func TestDeleteHTTPRequestDetailsOnlyDeletesArchivedRows(t *testing.T) {
	db := newMonitoringTestDB(t)
	now := time.Now().UTC()
	archivedError := &HTTPRequestLog{Method: "GET", Path: "/error", Route: "/error", StatusCode: 500, ClientIP: "1.1.1.1", CreatedAt: now.Add(-2 * time.Hour)}
	liveError := &HTTPRequestLog{Method: "GET", Path: "/live", Route: "/live", StatusCode: 500, ClientIP: "2.2.2.2", CreatedAt: now.Add(-time.Minute)}
	if err := db.CreateHTTPRequestLog(archivedError); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateHTTPRequestLog(liveError); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ArchiveHTTPRequestLogs(now.Add(-time.Hour), 0, 0); err != nil {
		t.Fatal(err)
	}

	deleted, err := db.DeleteHTTPRequestDetails(now, "errors")
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted=%d, want 1", deleted)
	}

	logs, total, err := db.ListHTTPRequestLogs(now.Add(-24*time.Hour), "all", "", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(logs) != 1 || logs[0].Path != "/live" {
		t.Fatalf("unexpected remaining logs: total=%d logs=%+v", total, logs)
	}
	overview, err := db.GetMonitoringOverview(now.Add(-24 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if overview.RequestCount != 2 {
		t.Fatalf("request_count=%d, want 2", overview.RequestCount)
	}
}

func TestMonitoringTopEndpointsCombinesArchivedAndLive(t *testing.T) {
	db := newMonitoringTestDB(t)
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		item := &HTTPRequestLog{Method: "GET", Path: "/api/items/1", Route: "/api/items/:id", StatusCode: 200, ClientIP: "3.3.3.3", LatencyMS: int64(10 + i), CreatedAt: now.Add(-2 * time.Hour)}
		if err := db.CreateHTTPRequestLog(item); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.CreateHTTPRequestLog(&HTTPRequestLog{Method: "POST", Path: "/api/items", Route: "/api/items", StatusCode: 400, ClientIP: "4.4.4.4", LatencyMS: 20, CreatedAt: now.Add(-time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ArchiveHTTPRequestLogs(now.Add(-time.Hour), 30, 730); err != nil {
		t.Fatal(err)
	}

	stats, err := db.GetMonitoringTopEndpoints(now.Add(-24*time.Hour), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 2 || stats[0].Route != "/api/items/:id" || stats[0].Requests != 3 {
		t.Fatalf("unexpected endpoints: %+v", stats)
	}
}

func TestListHTTPRequestLogsAcceptsStatusBuckets(t *testing.T) {
	db := newMonitoringTestDB(t)
	now := time.Now().UTC()
	samples := []struct {
		path   string
		status int
	}{
		{"/a", 201}, {"/b", 301}, {"/c", 404}, {"/d", 502}, {"/e", 200},
	}
	for _, s := range samples {
		item := &HTTPRequestLog{Method: "GET", Path: s.path, Route: s.path, StatusCode: s.status, ClientIP: "9.9.9.9", CreatedAt: now.Add(-time.Minute)}
		if err := db.CreateHTTPRequestLog(item); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ArchiveHTTPRequestLogs(now.Add(-time.Hour), 30, 730); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		group   string
		want    []int
		minSize int
	}{
		{group: "2xx", want: []int{201, 200}},
		{group: "3xx", want: []int{301}},
		{group: "4xx", want: []int{404}},
		{group: "5xx", want: []int{502}},
	}
	for _, tc := range cases {
		logs, total, err := db.ListHTTPRequestLogs(now.Add(-24*time.Hour), tc.group, "", 50, 0)
		if err != nil {
			t.Fatalf("group=%s err=%v", tc.group, err)
		}
		if total != int64(len(tc.want)) {
			t.Fatalf("group=%s total=%d want=%d", tc.group, total, len(tc.want))
		}
		for _, l := range logs {
			found := false
			for _, code := range tc.want {
				if l.StatusCode == code {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("group=%s unexpected status %d", tc.group, l.StatusCode)
			}
		}
	}

	if _, _, err := db.ListHTTPRequestLogs(now.Add(-24*time.Hour), "bogus", "", 50, 0); err == nil {
		t.Fatal("expected error for invalid status group")
	}
}

func TestProxySessionLoggingRoundTrip(t *testing.T) {
	db := newMonitoringTestDB(t)
	now := time.Now().UTC()

	ws := &ProxySessionLog{
		SessionType: "websocket",
		Method:      "WS",
		Route:       "/api/terminal/:id/ws",
		Target:      "session-123",
		ClientIP:    "1.1.1.1",
		StatusCode:  101,
		DurationMS:  12345,
		BytesIn:     4096,
		BytesOut:    2048,
		FramesIn:    12,
		FramesOut:   8,
		CloseReason: "client_eof",
	}
	tunnel := &ProxySessionLog{
		SessionType: "connect_tunnel",
		Method:      "CONNECT",
		Route:       "",
		Target:      "example.com:443",
		ClientIP:    "2.2.2.2",
		StatusCode:  200,
		DurationMS:  6000,
		BytesIn:     1024 * 1024,
		BytesOut:    512 * 1024,
		CloseReason: "client_eof",
	}
	if err := db.CreateProxySessionLog(ws); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateProxySessionLog(tunnel); err != nil {
		t.Fatal(err)
	}

	summary, err := db.GetProxySessionSummary(now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if summary.WSSessions != 1 || summary.WSFramesIn != 12 || summary.WSFramesOut != 8 {
		t.Fatalf("ws summary wrong: %+v", summary)
	}
	if summary.TunnelSessions != 1 || summary.TunnelBytesIn != 1024*1024 || summary.TunnelBytesOut != 512*1024 {
		t.Fatalf("tunnel summary wrong: %+v", summary)
	}

	logs, total, err := db.ListProxySessionLogs("websocket", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(logs) != 1 || logs[0].Route != "/api/terminal/:id/ws" {
		t.Fatalf("logs wrong: total=%d logs=%+v", total, logs)
	}
}

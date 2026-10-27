package models

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

// 用例里反复出现的固定日期集中成常量。
const (
	newsTestDate      = "2026-10-25" // 用例基准日期
	newsTestDatePrev  = "2026-10-24" // 基准日期的前一天
	newsTestDatePrev2 = "2026-10-23" // 基准日期的前两天
)

// newsTestDB 构造只初始化了 news 表的内存库。
// 注意(R5):sqlite 的 ":memory:" 每个连接都是一份独立内存库,必须 SetMaxOpenConns(1),
// 否则另一条连接会拿到另一份库,表现为「刚插入就查不到」。
func newsTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("创建内存库失败: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitNews(); err != nil {
		t.Fatalf("初始化 news 表失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// newsDigestFixture 造一条归档记录。
func newsDigestFixture(date, summary string, itemCount int) *NewsDigest {
	return &NewsDigest{
		Date:        date,
		DateLabel:   date,
		Summary:     summary,
		Items:       MarshalNewsItems([]NewsItem{{Title: "标题-" + date, Link: "https://example.com/" + date, Source: "IT之家"}}),
		SourceCount: 1,
		ItemCount:   itemCount,
		AIModel:     "WorkBuddy-Agent",
		Status:      "ok",
	}
}

// TestNewsInitNewsCreatesTable InitNews 可重复执行(表已存在时不报错)。
func TestNewsInitNewsCreatesTable(t *testing.T) {
	db := newsTestDB(t)
	if err := db.InitNews(); err != nil {
		t.Fatalf("重复执行 InitNews 应无副作用, got err=%v", err)
	}
	d := newsDigestFixture(newsTestDate, "摘要", 1)
	if err := db.UpsertNewsDigest(d); err != nil {
		t.Fatalf("写入归档失败: %v", err)
	}
	got, err := db.GetNewsDigest(newsTestDate)
	if err != nil {
		t.Fatalf("读取归档失败: %v", err)
	}
	if got.Date != newsTestDate {
		t.Errorf("date = %q, want %q", got.Date, newsTestDate)
	}
}

// TestNewsUpsertDigestOverwritesSameDate 同一天二次写入是覆盖而不是追加(幂等)。
func TestNewsUpsertDigestOverwritesSameDate(t *testing.T) {
	db := newsTestDB(t)
	first := newsDigestFixture(newsTestDate, "第一版摘要", 2)
	if err := db.UpsertNewsDigest(first); err != nil {
		t.Fatalf("首次写入失败: %v", err)
	}
	second := newsDigestFixture(newsTestDate, "第二版摘要", 5)
	second.Status = "partial"
	second.SourceCount = 3
	if err := db.UpsertNewsDigest(second); err != nil {
		t.Fatalf("二次写入失败: %v", err)
	}

	n, err := db.CountNewsDigests()
	if err != nil {
		t.Fatalf("统计归档数失败: %v", err)
	}
	if n != 1 {
		t.Fatalf("归档条数 = %d, want 1(同一天应覆盖不追加)", n)
	}
	got, err := db.GetNewsDigest(newsTestDate)
	if err != nil {
		t.Fatalf("读取归档失败: %v", err)
	}
	if got.Summary != "第二版摘要" {
		t.Errorf("summary = %q, want %q", got.Summary, "第二版摘要")
	}
	if got.ItemCount != 5 {
		t.Errorf("item_count = %d, want 5", got.ItemCount)
	}
	if got.SourceCount != 3 {
		t.Errorf("source_count = %d, want 3", got.SourceCount)
	}
	if got.Status != "partial" {
		t.Errorf("status = %q, want %q", got.Status, "partial")
	}
}

// TestNewsUpsertDigestKeepsCreatedAt 覆盖写不改动最初的 created_at。
func TestNewsUpsertDigestKeepsCreatedAt(t *testing.T) {
	db := newsTestDB(t)
	first := newsDigestFixture(newsTestDate, "第一版", 1)
	if err := db.UpsertNewsDigest(first); err != nil {
		t.Fatalf("首次写入失败: %v", err)
	}
	created := first.CreatedAt
	if created.IsZero() {
		t.Fatal("首次写入后 CreatedAt 不应为零值")
	}

	second := newsDigestFixture(newsTestDate, "第二版", 2)
	if err := db.UpsertNewsDigest(second); err != nil {
		t.Fatalf("二次写入失败: %v", err)
	}
	got, err := db.GetNewsDigest(newsTestDate)
	if err != nil {
		t.Fatalf("读取归档失败: %v", err)
	}
	if !got.CreatedAt.Equal(created) {
		t.Errorf("created_at = %v, want %v(覆盖写不应改动创建时间)", got.CreatedAt, created)
	}
	if !got.UpdatedAt.Equal(second.UpdatedAt) {
		t.Errorf("updated_at = %v, want %v", got.UpdatedAt, second.UpdatedAt)
	}
}

// TestNewsMarshalNewsItemsRoundTrip 条目序列化/反序列化往返一致(含中文、空格、引号、换行)。
func TestNewsMarshalNewsItemsRoundTrip(t *testing.T) {
	items := []NewsItem{
		{Title: "中文标题", Link: "https://example.com/1", Source: "IT之家", Summary: "含 \"引号\" 的摘要", Category: "科技", Published: "2026-10-25T01:00:00Z"},
		{Title: "with space", Link: "https://example.com/2", Source: "sspai", Summary: "line1\nline2", Category: "设计"},
		{Title: "emoji 🚀 与标点,。!?", Link: "", Source: "", Summary: ""},
	}
	raw := MarshalNewsItems(items)
	if raw == "" {
		t.Fatal("MarshalNewsItems 不应返回空串")
	}
	got, err := UnmarshalNewsItems(raw)
	if err != nil {
		t.Fatalf("UnmarshalNewsItems 失败: %v, raw=%s", err, raw)
	}
	if len(got) != len(items) {
		t.Fatalf("往返后条目数 = %d, want %d", len(got), len(items))
	}
	for i := range items {
		if got[i] != items[i] {
			t.Errorf("条目 %d 往返不一致:\n got=%+v\nwant=%+v", i, got[i], items[i])
		}
	}
}

// TestNewsMarshalNewsItemsEmpty 空列表序列化为 "[]"。
func TestNewsMarshalNewsItemsEmpty(t *testing.T) {
	if got := MarshalNewsItems(nil); got != "[]" {
		t.Errorf("MarshalNewsItems(nil) = %q, want %q", got, "[]")
	}
	if got := MarshalNewsItems([]NewsItem{}); got != "[]" {
		t.Errorf("MarshalNewsItems(空切片) = %q, want %q", got, "[]")
	}
}

// TestNewsUnmarshalNewsItemsErrors 反序列化的容错语义。
func TestNewsUnmarshalNewsItemsErrors(t *testing.T) {
	t.Run("空串返回空列表且无错误", func(t *testing.T) {
		got, err := UnmarshalNewsItems("")
		if err != nil {
			t.Fatalf("UnmarshalNewsItems(\"\") 不应报错, got %v", err)
		}
		if len(got) != 0 {
			t.Errorf("条目数 = %d, want 0", len(got))
		}
	})
	t.Run("非法 JSON 返回错误", func(t *testing.T) {
		if _, err := UnmarshalNewsItems("not json"); err == nil {
			t.Error("UnmarshalNewsItems(\"not json\") 应返回错误")
		}
	})
	t.Run("JSON null 返回空列表且无错误", func(t *testing.T) {
		got, err := UnmarshalNewsItems("null")
		if err != nil {
			t.Fatalf("UnmarshalNewsItems(\"null\") 不应报错, got %v", err)
		}
		if len(got) != 0 {
			t.Errorf("条目数 = %d, want 0", len(got))
		}
	})
}

// TestNewsGetLatestNewsDigest 取最新一期(按日期倒序第一条)。
func TestNewsGetLatestNewsDigest(t *testing.T) {
	db := newsTestDB(t)
	if _, err := db.GetLatestNewsDigest(); err == nil {
		t.Error("空库时 GetLatestNewsDigest 应返回错误")
	} else if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("空库错误 = %v, want sql.ErrNoRows", err)
	}

	for _, date := range []string{newsTestDatePrev2, newsTestDate, newsTestDatePrev} {
		if err := db.UpsertNewsDigest(newsDigestFixture(date, "摘要", 1)); err != nil {
			t.Fatalf("写入 %s 失败: %v", date, err)
		}
	}
	got, err := db.GetLatestNewsDigest()
	if err != nil {
		t.Fatalf("读取最新归档失败: %v", err)
	}
	if got.Date != newsTestDate {
		t.Errorf("最新归档 date = %q, want %q", got.Date, newsTestDate)
	}
}

// TestNewsGetNewsDigestNotFound 查询不存在的日期返回 sql.ErrNoRows。
func TestNewsGetNewsDigestNotFound(t *testing.T) {
	db := newsTestDB(t)
	_, err := db.GetNewsDigest(newsTestDate)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
}

// TestNewsListNewsDates 归档索引:默认 limit、按日期倒序、offset 生效、上限截断。
func TestNewsListNewsDates(t *testing.T) {
	db := newsTestDB(t)
	for _, date := range []string{newsTestDatePrev2, newsTestDate, newsTestDatePrev} {
		if err := db.UpsertNewsDigest(newsDigestFixture(date, "摘要", 1)); err != nil {
			t.Fatalf("写入 %s 失败: %v", date, err)
		}
	}

	t.Run("limit<=0 时用默认30", func(t *testing.T) {
		list, err := db.ListNewsDates(0, 0)
		if err != nil {
			t.Fatalf("ListNewsDates 失败: %v", err)
		}
		if len(list) != 3 {
			t.Fatalf("列表长度 = %d, want 3", len(list))
		}
	})
	t.Run("按日期倒序", func(t *testing.T) {
		list, err := db.ListNewsDates(30, 0)
		if err != nil {
			t.Fatalf("ListNewsDates 失败: %v", err)
		}
		want := []string{newsTestDate, newsTestDatePrev, newsTestDatePrev2}
		for i, w := range want {
			if list[i].Date != w {
				t.Errorf("list[%d].date = %q, want %q", i, list[i].Date, w)
			}
		}
	})
	t.Run("limit 与 offset 生效", func(t *testing.T) {
		list, err := db.ListNewsDates(1, 1)
		if err != nil {
			t.Fatalf("ListNewsDates 失败: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("列表长度 = %d, want 1", len(list))
		}
		if list[0].Date != newsTestDatePrev {
			t.Errorf("list[0].date = %q, want %q", list[0].Date, newsTestDatePrev)
		}
	})
	t.Run("limit 上限截断为100且 offset 为负时归零", func(t *testing.T) {
		list, err := db.ListNewsDates(200, -5)
		if err != nil {
			t.Fatalf("ListNewsDates 失败: %v", err)
		}
		if len(list) != 3 {
			t.Fatalf("列表长度 = %d, want 3", len(list))
		}
		if list[0].Date != newsTestDate {
			t.Errorf("list[0].date = %q, want %q(负 offset 应归零)", list[0].Date, newsTestDate)
		}
	})
	t.Run("列表不含 items 大字段", func(t *testing.T) {
		list, err := db.ListNewsDates(30, 0)
		if err != nil {
			t.Fatalf("ListNewsDates 失败: %v", err)
		}
		if list[0].Items != "" {
			t.Errorf("列表项不应带 items, got %q", list[0].Items)
		}
	})
}

// TestNewsCountNewsDigests 统计归档条数。
func TestNewsCountNewsDigests(t *testing.T) {
	db := newsTestDB(t)
	n, err := db.CountNewsDigests()
	if err != nil {
		t.Fatalf("统计归档数失败: %v", err)
	}
	if n != 0 {
		t.Fatalf("空库归档数 = %d, want 0", n)
	}
	for _, date := range []string{newsTestDatePrev2, newsTestDate} {
		if err := db.UpsertNewsDigest(newsDigestFixture(date, "摘要", 1)); err != nil {
			t.Fatalf("写入 %s 失败: %v", date, err)
		}
	}
	n, err = db.CountNewsDigests()
	if err != nil {
		t.Fatalf("统计归档数失败: %v", err)
	}
	if n != 2 {
		t.Errorf("归档数 = %d, want 2", n)
	}
}

// TestNewsDeleteNewsDigestsBefore 清理早于给定日期的归档(按 YYYY-MM-DD 字符串比较)。
func TestNewsDeleteNewsDigestsBefore(t *testing.T) {
	db := newsTestDB(t)
	for _, date := range []string{"2026-10-20", "2026-10-21", newsTestDate} {
		if err := db.UpsertNewsDigest(newsDigestFixture(date, "摘要", 1)); err != nil {
			t.Fatalf("写入 %s 失败: %v", date, err)
		}
	}
	deleted, err := db.DeleteNewsDigestsBefore(time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("清理过期归档失败: %v", err)
	}
	if deleted != 2 {
		t.Errorf("删除条数 = %d, want 2", deleted)
	}
	n, err := db.CountNewsDigests()
	if err != nil {
		t.Fatalf("统计归档数失败: %v", err)
	}
	if n != 1 {
		t.Errorf("剩余归档数 = %d, want 1", n)
	}
	if _, err := db.GetNewsDigest(newsTestDate); err != nil {
		t.Errorf("2026-10-25 的归档不应被删除: %v", err)
	}
}

package trip

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"devtools/notif"
)

// ProcessDueReminders 扫描 [now - 1h, now + 1h) 区间内触发时刻的提醒,
// 走共享 notif 包发邮件;每条幂等(标记 remind_sent_at 后不再发)。
//
// 设计:
//   - 主协程在 cleanup tick 里每小时调一次。
//   - 单条 panic 由外层 recover 兜底,本函数内部不再嵌套。
//   - SMTP 未配置时,所有提醒静默跳过(只在启动期日志提示)。
type ReminderProcessor struct {
	repo  Repository
	notif notif.Config
}

// NewReminderProcessor 构造一个提醒处理器。
func NewReminderProcessor(repo Repository, cfg notif.Config) *ReminderProcessor {
	return &ReminderProcessor{repo: repo, notif: cfg}
}

// ProcessNow 立刻处理一次,window 为 [now-30min, now+30min)。
// 给前端 UI 一个"立即试发"的按钮用,也是 cleanup tick 复用入口。
func (p *ReminderProcessor) ProcessNow(ctx context.Context) (sent, skipped int, err error) {
	now := time.Now()
	from := now.Add(-30 * time.Minute)
	to := now.Add(30 * time.Minute)
	items, err := p.repo.ListDueReminders(ctx, from, to, 100)
	if err != nil {
		return 0, 0, err
	}
	for _, it := range items {
		if p.sendOne(ctx, it) {
			sent++
		} else {
			skipped++
		}
	}
	return sent, skipped, nil
}

// ProcessWindow 处理一个显式时间窗口,供 cleanup tick 调用。
func (p *ReminderProcessor) ProcessWindow(ctx context.Context, from, to time.Time) (sent, skipped int, err error) {
	items, err := p.repo.ListDueReminders(ctx, from, to, 200)
	if err != nil {
		return 0, 0, err
	}
	for _, it := range items {
		if p.sendOne(ctx, it) {
			sent++
		} else {
			skipped++
		}
	}
	return sent, skipped, nil
}

func (p *ReminderProcessor) sendOne(ctx context.Context, it *ActivityReminderItem) bool {
	trip := it.Trip
	day := it.Day
	act := it.Activity
	trig, _ := ActivityReminderTime(day, act)

	recipients := mergeRecipients(act.RemindEmails, trip.NotifyEmail)
	if len(recipients) == 0 {
		// 没收件人 = 跳过,但仍标记已发,避免每次 tick 都重新扫到。
		// 否则用户配置 NotifyEmail 之前会反复触发扫描。
		_ = p.repo.MarkReminderSent(ctx, act.ID, time.Now())
		return false
	}

	subject := fmt.Sprintf("行程提醒: %s · %s", trip.Name, act.Title)
	bodyLines := []string{
		fmt.Sprintf("行程: %s", trip.Name),
		fmt.Sprintf("日期: %s(%s)", day.Date, weekdayCN(day.Date)),
		fmt.Sprintf("时间: %s", firstNonEmpty(act.StartTime, "全天")),
		fmt.Sprintf("活动: [%s] %s", kindCN(act.Kind), act.Title),
	}
	if act.Location != "" {
		bodyLines = append(bodyLines, fmt.Sprintf("地点: %s", act.Location))
	}
	if dest := destinationLabel(act.Destination); dest != "" {
		bodyLines = append(bodyLines, fmt.Sprintf("目的地: %s", dest))
	}
	if act.Note != "" {
		bodyLines = append(bodyLines, "", "备注:", act.Note)
	}
	bodyLines = append(bodyLines,
		"",
		fmt.Sprintf("触发时刻: %s", trig.In(time.Local).Format("2006-01-02 15:04")),
		fmt.Sprintf("提前时间: %d 分钟", act.RemindBeforeMinutes),
	)
	body := strings.Join(bodyLines, "\n")

	ok := notif.SendAndLog(p.notif, recipients, subject, body, nil)
	if ok {
		_ = p.repo.MarkReminderSent(ctx, act.ID, time.Now())
	}
	return ok
}

func mergeRecipients(activityEmails []string, tripDefault string) []string {
	merged := make([]string, 0, len(activityEmails)+1)
	for _, e := range activityEmails {
		e = strings.TrimSpace(e)
		if e != "" {
			merged = append(merged, e)
		}
	}
	for _, e := range notif.SplitRecipients(tripDefault) {
		found := false
		for _, m := range merged {
			if strings.EqualFold(m, e) {
				found = true
				break
			}
		}
		if !found {
			merged = append(merged, e)
		}
	}
	return merged
}

// QuickLog 启动期一次性日志,提示 SMTP 是否可用,避免用户配错一脸懵。
func (p *ReminderProcessor) QuickLog(scope string) {
	if p.notif.Configured() {
		log.Printf("trip reminder[%s]: SMTP 已配置 host=%s from=%s", scope, p.notif.Host, p.notif.FromOrDefault())
	} else {
		log.Printf("trip reminder[%s]: SMTP 未配置,所有行程提醒将静默跳过(可在 TripTool 配置页填写 notify_email)", scope)
	}
}

// ResetSentAt 把活动的 remind_sent_at 清零,允许"测试提醒"按钮重发。
// 仅在用户显式传 reset=true 时调用,避免误清导致反复扫描。
func (p *ReminderProcessor) ResetSentAt(ctx context.Context, activityID string) (int64, error) {
	return p.repo.ResetReminderSent(ctx, activityID)
}
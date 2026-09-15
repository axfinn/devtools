// Command trip 是「旅游行程助手」的 CLI 入口。
//
// 与 HTTP 端点共享 backend/trip.Service;CLI 默认连独立 SQLite 文件
// (./data/trip.db,与 HTTP 默认走的主库 ./data/paste.db 不共享连接),
// 避免 CLI 误操作影响 HTTP 服务。HTTP/CLI 共用同一份领域代码。
//
// 子命令:
//
//	init                          初始化数据库表(幂等)
//	seed [--force]                落地第一个行程(已存在则跳过;--force 删除同名重建)
//	list [--json]                 列出所有行程
//	show <trip-id>                渲染完整 Markdown 行程单到 stdout
//	create --name N --start S --end E [--description X] [--cities X] [--tags X] [--notify-email E]
//	delete <trip-id>              删除行程(级联 days+activities+expenses)
//	add-day <trip-id> --date S [--city X] [--region X] [--country X]
//	add-activity <day-id> --kind K --title N [--time X] [--location X] [--notes X]
//	                          [--city X --region X --country X]
//	                          [--remind-before-minutes N --remind-email E1,E2]
//	remind <activity-id> --before N [--email E1,E2] [--reset]
//	update-activity <activity-id> [--title X --time X --location X --notes X --kind K]
//	                          出行中 inline 调整活动
//	days <trip-id>                列出某 trip 的所有 DayPlan
//	activities <day-id>           列出某天的所有 Activity
//	budget <trip-id> --total 20000 [--currency CNY]
//	                          设置整程预算
//	expense-add <trip-id> --date D --category K --amount N [--currency C] [--note X] [--payment X]
//	                          出行中记一笔费用
//	expense-list <trip-id>        列出费用
//	expense-delete <expense-id>   删除一笔费用
//	summary <trip-id> [--md]      行程总结(回程后一键 Markdown)
//	help                          显示帮助
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"devtools/trip"

	_ "github.com/mattn/go-sqlite3"
)

const defaultDBPath = "./data/trip.db"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	var dbPath string
	for i := 0; i < len(args); {
		switch {
		case args[i] == "--db" && i+1 < len(args):
			dbPath = args[i+1]
			args = append(args[:i], args[i+2:]...)
		case strings.HasPrefix(args[i], "--db="):
			dbPath = strings.TrimPrefix(args[i], "--db=")
			args = append(args[:i], args[i+1:]...)
		default:
			i++
		}
	}
	if dbPath == "" {
		dbPath = defaultDBPath
	}
	if len(args) == 0 {
		printUsage()
		return nil
	}
	cmd := args[0]
	rest := args[1:]

	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	conn, err := sql.Open("sqlite3", dbPath+"?_parse_time=true")
	if err != nil {
		return fmt.Errorf("open db %s: %w", dbPath, err)
	}
	defer conn.Close()

	repo := trip.NewSQLiteRepository(conn)
	svc := trip.NewService(repo)
	ctx := context.Background()
	if err := svc.EnsureSchema(ctx); err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}

	switch cmd {
	case "help", "-h", "--help":
		printUsage()
		return nil

	case "init":
		fmt.Printf("trip: schema initialized at %s\n", dbPath)
		return nil

	case "seed":
		fs := flag.NewFlagSet("seed", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		force := fs.Bool("force", false, "delete existing trip with same name first")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		id, created, err := svc.SeedRealTrip(ctx, *force)
		if err != nil {
			return err
		}
		verb := "seeded"
		if !created {
			verb = "kept existing"
		}
		fmt.Printf("trip: %s trip_id=%s\n", verb, id)
		return nil

	case "list":
		asJSON := hasFlag(rest, "--json")
		trips, err := svc.ListTrips(ctx)
		if err != nil {
			return err
		}
		if len(trips) == 0 {
			fmt.Println("(no trips; try `trip seed` to load the default itinerary)")
			return nil
		}
		if asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(trips)
		}
		fmt.Printf("%-36s  %-12s  %-12s  %-36s\n", "NAME", "START", "END", "ID")
		fmt.Println(strings.Repeat("-", 110))
		for _, t := range trips {
			fmt.Printf("%-36s  %-12s  %-12s  %-36s\n",
				truncate(t.Name, 36), t.StartDate, t.EndDate, t.ID)
		}
		return nil

	case "show":
		if len(rest) == 0 {
			return errors.New("usage: trip show <trip-id>")
		}
		md, err := svc.RenderMarkdown(ctx, rest[0])
		if err != nil {
			return err
		}
		fmt.Print(md)
		return nil

	case "delete":
		if len(rest) == 0 {
			return errors.New("usage: trip delete <trip-id>")
		}
		if err := svc.DeleteTrip(ctx, rest[0]); err != nil {
			return err
		}
		fmt.Printf("trip: deleted %s\n", rest[0])
		return nil

	case "create":
		fs := flag.NewFlagSet("create", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		name := fs.String("name", "", "trip name (required)")
		start := fs.String("start", "", "start date YYYY-MM-DD (required)")
		end := fs.String("end", "", "end date YYYY-MM-DD (required)")
		desc := fs.String("description", "", "one-line description")
		cities := fs.String("cities", "", "comma-separated cities")
		tags := fs.String("tags", "", "comma-separated tags")
		notify := fs.String("notify-email", "", "default reminder recipient email")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		t := &trip.Trip{
			Name:        strings.TrimSpace(*name),
			Description: strings.TrimSpace(*desc),
			StartDate:   strings.TrimSpace(*start),
			EndDate:     strings.TrimSpace(*end),
			CoverCities: splitCSV(*cities),
			Tags:        splitCSV(*tags),
			NotifyEmail: strings.TrimSpace(*notify),
		}
		if err := svc.CreateTrip(ctx, t); err != nil {
			return err
		}
		fmt.Printf("trip: created id=%s name=%q\n", t.ID, t.Name)
		return nil

	case "add-day":
		fs := flag.NewFlagSet("add-day", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		positional := fs.String("trip-id", "", "trip id (positional also accepted)")
		date := fs.String("date", "", "YYYY-MM-DD (required)")
		city := fs.String("city", "", "primary city for the day")
		region := fs.String("region", "", "region within city")
		country := fs.String("country", "", "country")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		tripID := *positional
		if tripID == "" && fs.NArg() > 0 {
			tripID = fs.Arg(0)
		}
		if tripID == "" {
			return errors.New("usage: trip add-day <trip-id> --date YYYY-MM-DD")
		}
		if *date == "" {
			return errors.New("--date is required (YYYY-MM-DD)")
		}
		dp := &trip.DayPlan{
			TripID: tripID,
			Date:   strings.TrimSpace(*date),
			Destination: trip.Destination{
				City:    strings.TrimSpace(*city),
				Region:  strings.TrimSpace(*region),
				Country: strings.TrimSpace(*country),
			},
		}
		if err := svc.AddDay(ctx, dp); err != nil {
			return err
		}
		fmt.Printf("trip: added day id=%s trip=%s date=%s\n", dp.ID, dp.TripID, dp.Date)
		return nil

	case "add-activity":
		fs := flag.NewFlagSet("add-activity", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		positional := fs.String("day-id", "", "day id (positional also accepted)")
		kind := fs.String("kind", "sight", "transit|sight|food|lodging|shopping|leisure")
		title := fs.String("title", "", "activity title (required)")
		t := fs.String("time", "", "HH:MM or 上午/下午/晚上")
		loc := fs.String("location", "", "free-text location")
		notes := fs.String("notes", "", "free-text notes")
		city := fs.String("city", "", "destination city")
		region := fs.String("region", "", "destination region")
		country := fs.String("country", "", "destination country")
		duration := fs.Int("duration-min", 0, "duration in minutes")
		remBefore := fs.Int("remind-before-minutes", 0, "send reminder N minutes before start time (0=disable)")
		remEmails := fs.String("remind-email", "", "comma-separated reminder recipients")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		dayID := *positional
		if dayID == "" && fs.NArg() > 0 {
			dayID = fs.Arg(0)
		}
		if dayID == "" {
			return errors.New("usage: trip add-activity <day-id> --title X --kind K [...]")
		}
		if *title == "" {
			return errors.New("--title is required")
		}
		a := &trip.Activity{
			DayID:               dayID,
			Kind:                trip.ActivityKind(strings.TrimSpace(*kind)),
			Title:               strings.TrimSpace(*title),
			StartTime:           strings.TrimSpace(*t),
			Location:            strings.TrimSpace(*loc),
			Note:                strings.TrimSpace(*notes),
			DurationMin:         *duration,
			RemindBeforeMinutes: *remBefore,
			RemindEmails:        splitCSV(*remEmails),
			Destination: trip.Destination{
				City:    strings.TrimSpace(*city),
				Region:  strings.TrimSpace(*region),
				Country: strings.TrimSpace(*country),
			},
		}
		if err := svc.AddActivity(ctx, a); err != nil {
			return err
		}
		fmt.Printf("trip: added activity id=%s day=%s [%s] %s\n", a.ID, a.DayID, a.Kind, a.Title)
		return nil

	case "remind":
		fs := flag.NewFlagSet("remind", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		positional := fs.String("activity-id", "", "activity id (positional also accepted)")
		before := fs.Int("before", 0, "minutes before start (0=disable)")
		emails := fs.String("email", "", "comma-separated reminder recipients")
		reset := fs.Bool("reset", false, "clear sent_at so reminder can fire again")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		actID := *positional
		if actID == "" && fs.NArg() > 0 {
			actID = fs.Arg(0)
		}
		if actID == "" {
			return errors.New("usage: trip remind <activity-id> --before N [--email E] [--reset]")
		}
		if err := svc.SetReminder(ctx, actID, *before, splitCSV(*emails)); err != nil {
			return err
		}
		if *reset {
			if _, err := repo.ResetReminderSent(ctx, actID); err != nil {
				return err
			}
		}
		fmt.Printf("trip: reminder updated activity=%s before=%dmin\n", actID, *before)
		return nil

	case "days":
		if len(rest) == 0 {
			return errors.New("usage: trip days <trip-id>")
		}
		days, err := repo.ListDaysByTrip(ctx, rest[0])
		if err != nil {
			return err
		}
		fmt.Printf("%-4s  %-12s  %-12s  %s\n", "DAY", "DATE", "CITY", "ID")
		fmt.Println(strings.Repeat("-", 80))
		for _, d := range days {
			city := d.Destination.City
			fmt.Printf("%-4d  %-12s  %-12s  %s\n", d.Order, d.Date, city, d.ID)
		}
		return nil

	case "activities":
		if len(rest) == 0 {
			return errors.New("usage: trip activities <day-id>")
		}
		acts, err := repo.ListActivitiesByDay(ctx, rest[0])
		if err != nil {
			return err
		}
		fmt.Printf("%-4s  %-12s  %-8s  %s\n", "ORDER", "KIND", "TIME", "TITLE")
		fmt.Println(strings.Repeat("-", 80))
		for _, a := range acts {
			fmt.Printf("%-4d  %-12s  %-8s  %s\n", a.Order, a.Kind, a.StartTime, a.Title)
		}
		return nil

	case "update-activity":
		fs := flag.NewFlagSet("update-activity", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		positional := fs.String("activity-id", "", "activity id (positional also accepted)")
		title := fs.String("title", "", "new title")
		kind := fs.String("kind", "", "transit|sight|food|leisure|shopping|lodging")
		t := fs.String("time", "", "HH:MM or 上午/下午/晚上")
		loc := fs.String("location", "", "free-text location")
		notes := fs.String("notes", "", "free-text notes")
		duration := fs.Int("duration-min", -1, "duration in minutes (-1 = leave unchanged)")
		city := fs.String("city", "", "destination city")
		region := fs.String("region", "", "destination region")
		country := fs.String("country", "", "destination country")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		actID := *positional
		if actID == "" && fs.NArg() > 0 {
			actID = fs.Arg(0)
		}
		if actID == "" {
			return errors.New("usage: trip update-activity <activity-id> [--title X --time X ...]")
		}
		// 先读,再覆盖
		cur, err := repo.GetActivity(ctx, actID)
		if err != nil {
			return err
		}
		if strings.TrimSpace(*title) != "" {
			cur.Title = strings.TrimSpace(*title)
		}
		if strings.TrimSpace(*kind) != "" {
			cur.Kind = trip.ActivityKind(strings.TrimSpace(*kind))
		}
		if strings.TrimSpace(*t) != "" {
			cur.StartTime = strings.TrimSpace(*t)
		}
		if strings.TrimSpace(*loc) != "" {
			cur.Location = strings.TrimSpace(*loc)
		}
		if strings.TrimSpace(*notes) != "" {
			cur.Note = strings.TrimSpace(*notes)
		}
		if *duration >= 0 {
			cur.DurationMin = *duration
		}
		if strings.TrimSpace(*city) != "" {
			cur.Destination.City = strings.TrimSpace(*city)
		}
		if *region != "" {
			cur.Destination.Region = strings.TrimSpace(*region)
		}
		if *country != "" {
			cur.Destination.Country = strings.TrimSpace(*country)
		}
		if err := svc.UpdateActivity(ctx, cur); err != nil {
			return err
		}
		fmt.Printf("trip: updated activity id=%s [%s] %s\n", cur.ID, cur.Kind, cur.Title)
		return nil

	case "budget":
		fs := flag.NewFlagSet("budget", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		positional := fs.String("trip-id", "", "trip id (positional also accepted)")
		total := fs.Float64("total", 0, "budget in yuan (e.g. 20000 = 2w 元)")
		cents := fs.Int64("total-cents", 0, "budget in cents (preferred for precision)")
		currency := fs.String("currency", "CNY", "ISO 4217 currency code")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		tripID := *positional
		if tripID == "" && fs.NArg() > 0 {
			tripID = fs.Arg(0)
		}
		if tripID == "" {
			return errors.New("usage: trip budget <trip-id> --total 20000 [--currency CNY]")
		}
		finalCents := *cents
		if finalCents == 0 && *total != 0 {
			finalCents = int64(*total * 100)
		}
		if err := svc.UpdateBudget(ctx, tripID, finalCents, strings.ToUpper(*currency)); err != nil {
			return err
		}
		fmt.Printf("trip: budget set trip=%s total=%s currency=%s\n",
			tripID, formatYuanFromCents(finalCents), strings.ToUpper(*currency))
		return nil

	case "expense-add":
		fs := flag.NewFlagSet("expense-add", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		positional := fs.String("trip-id", "", "trip id (positional also accepted)")
		date := fs.String("date", "", "YYYY-MM-DD (required)")
		category := fs.String("category", "", "transport|lodging|food|sight|shopping|misc (required)")
		amount := fs.Float64("amount", 0, "amount in yuan (e.g. 88.50)")
		cents := fs.Int64("amount-cents", 0, "amount in cents (preferred)")
		currency := fs.String("currency", "", "ISO 4217 (default = trip currency)")
		note := fs.String("note", "", "free-text note")
		payment := fs.String("payment", "", "cash|card|alipay|wechat|other")
		activityID := fs.String("activity-id", "", "optional: link to a specific activity")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		tripID := *positional
		if tripID == "" && fs.NArg() > 0 {
			tripID = fs.Arg(0)
		}
		if tripID == "" {
			return errors.New("usage: trip expense-add <trip-id> --date D --category K --amount N")
		}
		if *date == "" {
			return errors.New("--date is required (YYYY-MM-DD)")
		}
		if *category == "" {
			return errors.New("--category is required (transport|lodging|food|sight|shopping|misc)")
		}
		finalCents := *cents
		if finalCents == 0 && *amount != 0 {
			finalCents = int64(*amount * 100)
		}
		if finalCents == 0 {
			return errors.New("--amount (or --amount-cents) is required")
		}
		e := &trip.Expense{
			TripID:        tripID,
			Date:          strings.TrimSpace(*date),
			Category:      trip.ExpenseKind(strings.TrimSpace(*category)),
			AmountCents:   finalCents,
			Currency:      strings.ToUpper(strings.TrimSpace(*currency)),
			Note:          strings.TrimSpace(*note),
			PaymentMethod: strings.TrimSpace(*payment),
			ActivityID:    strings.TrimSpace(*activityID),
		}
		if err := svc.AddExpense(ctx, e); err != nil {
			return err
		}
		fmt.Printf("trip: added expense id=%s date=%s [%s] %s\n",
			e.ID, e.Date, e.Category, formatYuanFromCents(finalCents))
		return nil

	case "expense-list":
		if len(rest) == 0 {
			return errors.New("usage: trip expense-list <trip-id>")
		}
		exps, err := svc.ListExpenses(ctx, rest[0])
		if err != nil {
			return err
		}
		fmt.Printf("%-12s  %-10s  %-12s  %s\n", "DATE", "CATEGORY", "AMOUNT", "NOTE")
		fmt.Println(strings.Repeat("-", 90))
		for _, e := range exps {
			fmt.Printf("%-12s  %-10s  %-12s  %s\n",
				e.Date, e.Category, formatYuanFromCents(e.AmountCents), truncate(e.Note, 40))
		}
		fmt.Printf("\ntotal: %d expenses\n", len(exps))
		return nil

	case "expense-delete":
		if len(rest) == 0 {
			return errors.New("usage: trip expense-delete <expense-id>")
		}
		if err := svc.DeleteExpense(ctx, rest[0]); err != nil {
			return err
		}
		fmt.Printf("trip: deleted expense id=%s\n", rest[0])
		return nil

	case "summary":
		fs := flag.NewFlagSet("summary", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		positional := fs.String("trip-id", "", "trip id (positional also accepted)")
		asMarkdown := fs.Bool("md", false, "render Markdown summary instead of JSON-friendly text")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		tripID := *positional
		if tripID == "" && fs.NArg() > 0 {
			tripID = fs.Arg(0)
		}
		if tripID == "" {
			return errors.New("usage: trip summary <trip-id> [--md]")
		}
		if *asMarkdown {
			md, err := svc.RenderSummaryMarkdown(ctx, tripID)
			if err != nil {
				return err
			}
			fmt.Print(md)
			return nil
		}
		sum, err := svc.TripSummary(ctx, tripID)
		if err != nil {
			return err
		}
		fmt.Printf("trip summary: %s\n", sum.Trip.Name)
		fmt.Printf("  budget     : %s\n", formatYuanFromCents(sum.Trip.BudgetTotal))
		fmt.Printf("  spent      : %s\n", formatYuanFromCents(sum.TotalSpentCents))
		fmt.Printf("  remaining  : %s\n", formatYuanFromCents(sum.RemainingCents))
		fmt.Printf("  count      : %d expenses\n", sum.ExpenseCount)
		fmt.Printf("  by category: ")
		for i, c := range sum.TopCategories {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%s=%s", c.Category, formatYuanFromCents(c.Cents))
		}
		fmt.Println()
		return nil

	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func printUsage() {
	fmt.Println(`trip — 旅游行程助手 CLI

Usage:
  trip [--db PATH] <command> [args...]

Commands:
  init                          初始化数据库表(幂等)
  seed [--force]                落地第一个行程(已存在则跳过)
  list [--json]                 列出所有行程
  show <trip-id>                渲染完整 Markdown 行程单
  delete <trip-id>              删除行程
  create --name N --start S --end E [--description X --cities X --tags X --notify-email E]
  add-day <trip-id> --date S [--city X --region X --country X]
  add-activity <day-id> --kind K --title N [--time X --location X --notes X]
                                [--city X --region X --country X]
                                [--remind-before-minutes N --remind-email E1,E2]
  remind <activity-id> --before N [--email E1,E2] [--reset]
  update-activity <activity-id> [--title X --time X --location X --notes X --kind K --duration-min N]
                                出行中 inline 调整(标题/时间/地点/备注)
  days <trip-id>                列出某 trip 的所有 DayPlan
  activities <day-id>           列出某天的所有 Activity
  budget <trip-id> --total 20000 [--currency CNY]
                                设置整程预算(出行前)
  expense-add <trip-id> --date D --category K --amount N [--currency C --note X --payment X]
                                出行中快速记一笔费用
  expense-list <trip-id>        列出某 trip 的全部费用
  expense-delete <expense-id>   删除一笔费用
  summary <trip-id> [--md]      行程总结(回程后一键导出 Markdown)
  help                          显示帮助

Default DB path: ./data/trip.db (override with --db PATH)`)
}

func formatYuanFromCents(cents int64) string {
	if cents == 0 {
		return "¥0.00"
	}
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	whole := cents / 100
	frac := cents % 100
	return fmt.Sprintf("%s¥%d.%02d", sign, whole, frac)
}

func hasFlag(rest []string, name string) bool {
	for _, a := range rest {
		if a == name {
			return true
		}
	}
	return false
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
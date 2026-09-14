// Command trip 是「旅游行程助手」的 CLI 入口。
//
// 用法:
//
//	trip [--db PATH] <command> [args...]
//
// 子命令:
//
//	init                          初始化数据库表(幂等)
//	seed                          落地第一个行程(已存在则跳过)
//	create --name N --start S --end E [--summary X] [--cities X] [--tags X]
//	list                          列出所有行程
//	show <trip-id>                渲染完整 Markdown 行程单到 stdout
//	delete <trip-id>              删除行程(级联 days+activities)
//	add-day <trip-id> --date S [--city X] [--title X] [--summary X]
//	add-activity <day-id> --kind K --title N [--time X] [--location X] [--notes X]
//	days <trip-id>                列出某 trip 的所有 DayPlan
//	activities <day-id>           列出某天的所有 Activity
//	help                          显示帮助
//
// 默认 SQLite 路径: ./data/trip.db(可通过 --db 覆盖)。
// 数据库 schema 由 trip 包内的 InitSchema 负责,CLI 启动时自动建表。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	// 把第一个非 flag 参数作为子命令,后续参数由子命令解析。
	// 这避免子命令各自 fs.Parse 时的「flag provided but not defined」误伤。
	args := os.Args[1:]
	// 全局 --db 提到前面。
	var dbPath string
	for i := 0; i < len(args); {
		if args[i] == "--db" && i+1 < len(args) {
			dbPath = args[i+1]
			args = append(args[:i], args[i+2:]...)
			continue
		}
		if strings.HasPrefix(args[i], "--db=") {
			dbPath = strings.TrimPrefix(args[i], "--db=")
			args = append(args[:i], args[i+1:]...)
			continue
		}
		i++
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

	// 确保 data/ 目录存在(若使用默认相对路径)。
	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	repo, err := trip.OpenSQLite(dbPath)
	if err != nil {
		return fmt.Errorf("open db %s: %w", dbPath, err)
	}
	defer repo.Close()

	ctx := context.Background()
	if err := repo.InitSchema(ctx); err != nil {
		return err
	}
	svc := trip.NewService(repo)

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
		force := fs.Bool("force", false, "re-seed even if a trip with the same name already exists")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if *force {
			// force: 删除已有同名行程后重建
			existing, err := svc.List(ctx)
			if err != nil {
				return err
			}
			for _, t := range existing {
				if t.Name == "国庆 2026 粤港澳潮汕 14 日深度游" {
					if err := svc.DeleteTrip(ctx, t.ID); err != nil {
						return err
					}
					fmt.Printf("trip: removed existing %s (%s)\n", t.Name, t.ID)
				}
			}
		}
		t, created, err := trip.SeedFirstTrip(ctx, svc)
		if err != nil {
			return err
		}
		verb := "seeded"
		if !created {
			verb = "kept existing"
		}
		fmt.Printf("trip: %s %s (id=%s, %d days)\n", verb, t.Name, t.ID, len(t.Days))
		return nil

	case "create":
		fs := flag.NewFlagSet("create", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		name := fs.String("name", "", "trip name (required)")
		start := fs.String("start", "", "start date YYYY-MM-DD (required)")
		end := fs.String("end", "", "end date YYYY-MM-DD (required)")
		summary := fs.String("summary", "", "one-line summary")
		cities := fs.String("cities", "", "comma-separated cities")
		tags := fs.String("tags", "", "comma-separated tags")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		t, err := svc.CreateTrip(ctx, trip.CreateTripInput{
			Name:      *name,
			StartDate: *start,
			EndDate:   *end,
			Summary:   *summary,
			Cities:    *cities,
			Tags:      *tags,
		})
		if err != nil {
			return err
		}
		fmt.Printf("trip: created id=%s name=%q\n", t.ID, t.Name)
		return nil

	case "list":
		asJSON := hasFlag(rest, "--json")
		trips, err := svc.List(ctx)
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
		fmt.Printf("%-32s  %-12s  %-12s  %-12s  %s\n", "NAME", "START", "END", "ID", "CITIES")
		fmt.Println(strings.Repeat("-", 96))
		for _, t := range trips {
			fmt.Printf("%-32s  %-12s  %-12s  %-12s  %s\n",
				truncate(t.Name, 32), t.StartDate, t.EndDate, t.ID, t.Cities)
		}
		return nil

	case "show":
		if len(rest) == 0 {
			return errors.New("usage: trip show <trip-id> [--md]")
		}
		tripID := rest[0]
		t, err := svc.Show(ctx, tripID)
		if err != nil {
			return err
		}
		fmt.Print(svc.RenderMarkdown(t))
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

	case "add-day":
		flags, positional, err := parseFlagsAndPositional(rest, []string{"date", "city", "title", "summary"})
		if err != nil {
			return err
		}
		if len(positional) == 0 {
			return errors.New("usage: trip add-day <trip-id> --date YYYY-MM-DD [...]")
		}
		if flags["date"] == "" {
			return errors.New("--date is required (YYYY-MM-DD)")
		}
		d, err := svc.AddDay(ctx, trip.AddDayInput{
			TripID:  positional[0],
			Date:    flags["date"],
			City:    flags["city"],
			Title:   flags["title"],
			Summary: flags["summary"],
		})
		if err != nil {
			return err
		}
		fmt.Printf("trip: added day id=%s trip=%s index=%d date=%s\n",
			d.ID, d.TripID, d.DayIndex, d.Date)
		return nil

	case "add-activity":
		flags, positional, err := parseFlagsAndPositional(rest, []string{"kind", "time", "title", "location", "notes", "dest-name", "dest-region"})
		if err != nil {
			return err
		}
		if len(positional) == 0 {
			return errors.New("usage: trip add-activity <day-id> --title X --kind K [...]")
		}
		kind := trip.ActivityKind(flags["kind"])
		if kind == "" {
			kind = trip.ActivitySight
		}
		a, err := svc.AddActivity(ctx, trip.AddActivityInput{
			DayPlanID:         positional[0],
			Kind:              kind,
			Time:              flags["time"],
			Title:             flags["title"],
			Location:          flags["location"],
			Notes:             flags["notes"],
			DestinationName:   flags["dest-name"],
			DestinationRegion: flags["dest-region"],
		})
		if err != nil {
			return err
		}
		fmt.Printf("trip: added activity id=%s day=%s seq=%d [%s] %s\n",
			a.ID, a.DayPlanID, a.Seq, a.Kind, a.Title)
		return nil

	case "days":
		if len(rest) == 0 {
			return errors.New("usage: trip days <trip-id>")
		}
		days, err := repo.ListDaysByTrip(ctx, rest[0])
		if err != nil {
			return err
		}
		fmt.Printf("%-4s  %-12s  %-12s  %s\n", "DAY", "DATE", "CITY", "TITLE")
		fmt.Println(strings.Repeat("-", 80))
		for _, d := range days {
			fmt.Printf("%-4d  %-12s  %-12s  %s\n", d.DayIndex, d.Date, d.City, d.Title)
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
		fmt.Printf("%-4s  %-8s  %-8s  %s\n", "SEQ", "KIND", "TIME", "TITLE")
		fmt.Println(strings.Repeat("-", 80))
		for _, a := range acts {
			fmt.Printf("%-4d  %-8s  %-8s  %s\n", a.Seq, a.Kind, a.Time, a.Title)
		}
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
  create --name N --start S --end E [--summary X] [--cities X] [--tags X]
  list [--json]                 列出所有行程
  show <trip-id>                渲染完整 Markdown 行程单
  delete <trip-id>              删除行程
  add-day <trip-id> --date S [--city X] [--title X] [--summary X]
  add-activity <day-id> --kind K --title N [--time X] [--location X] [--notes X]
                                [--dest-name X --dest-region X]
  days <trip-id>                列出某 trip 的所有 DayPlan
  activities <day-id>           列出某天的所有 Activity
  help                          显示本帮助

Default DB path: ./data/trip.db (override with --db PATH)`)
}

// hasFlag —— 在 rest 里找完整 token(简单粗暴,但足够)。
func hasFlag(rest []string, name string) bool {
	for _, a := range rest {
		if a == name {
			return true
		}
	}
	return false
}

// parseFlagsAndPositional 手动拆分 --flag value 与位置参数,允许位置参数出现在任意位置。
// 之所以不用 fs.Parse:Go 的 flag 包在遇到第一个非 flag 实参时会停止,导致
// `trip add-day TRIP --date X` 解析不到 --date。
func parseFlagsAndPositional(rest []string, knownFlags []string) (map[string]string, []string, error) {
	flags := map[string]string{}
	for _, n := range knownFlags {
		flags[n] = ""
	}
	var positional []string
	i := 0
	for i < len(rest) {
		a := rest[i]
		if strings.HasPrefix(a, "--") {
			name := strings.TrimPrefix(a, "--")
			var value string
			hasValue := false
			if eq := strings.Index(name, "="); eq >= 0 {
				value = name[eq+1:]
				name = name[:eq]
				hasValue = true
			}
			if _, ok := flags[name]; !ok {
				return nil, nil, fmt.Errorf("unknown flag --%s", name)
			}
			if !hasValue {
				if i+1 >= len(rest) {
					return nil, nil, fmt.Errorf("flag --%s requires a value", name)
				}
				value = rest[i+1]
				i++
			}
			flags[name] = value
			i++
			continue
		}
		positional = append(positional, a)
		i++
	}
	return flags, positional, nil
}

// truncate 截断字符串到 n 字符(中文也算 1)。
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

// 为了消除 unused import 警告(time 保留给未来扩展)。
var _ = time.Now

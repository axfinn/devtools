package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"devtools/config"

	smb2 "github.com/hirochachacha/go-smb2"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config yaml")
	mountName := flag.String("mount", "win-share", "configured mount name")
	sharePath := flag.String("path", "", "share-relative file path, e.g. win-share/a.csv or a.csv")
	outPath := flag.String("out", "", "output file path")
	previewLines := flag.Int("preview-lines", 0, "print first N lines instead of writing file")
	dbPath := flag.String("db", "", "sqlite db path for direct import")
	profileID := flag.String("profile-id", "", "expense profile id for direct import")
	accountName := flag.String("account-name", "未指定账户", "target account name for imported rows")
	flag.Parse()

	if strings.TrimSpace(*sharePath) == "" {
		fail("missing --path")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fail("load config: %v", err)
	}

	var mount *config.MountConfig
	for i := range cfg.NFSShare.Mounts {
		if cfg.NFSShare.Mounts[i].Name == *mountName {
			mount = &cfg.NFSShare.Mounts[i]
			break
		}
	}
	if mount == nil {
		fail("mount %q not found", *mountName)
	}
	if strings.ToLower(mount.Type) != "smb" {
		fail("mount %q is %q, only smb is supported by this helper", mount.Name, mount.Type)
	}

	relPath := normalizePath(*sharePath, mount.Name)
	if relPath == "" {
		fail("invalid share path %q", *sharePath)
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(mount.Host, "445"), 10*time.Second)
	if err != nil {
		fail("dial smb host: %v", err)
	}
	defer conn.Close()

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     mount.Username,
			Password: mount.Password,
			Domain:   mount.Domain,
		},
	}

	session, err := d.Dial(conn)
	if err != nil {
		fail("dial smb session: %v", err)
	}
	defer session.Logoff()

	share, err := session.Mount(mount.Share)
	if err != nil {
		fail("mount smb share: %v", err)
	}
	defer share.Umount()

	file, err := share.Open(relPath)
	if err != nil {
		fail("open smb file: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		fail("read smb file: %v", err)
	}

	if *previewLines > 0 {
		printPreview(bytes.NewReader(data), *previewLines)
		return
	}

	if strings.TrimSpace(*outPath) != "" {
		if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
			fail("mkdir output dir: %v", err)
		}

		out, err := os.Create(*outPath)
		if err != nil {
			fail("create output file: %v", err)
		}
		defer out.Close()

		n, err := io.Copy(out, bytes.NewReader(data))
		if err != nil {
			fail("copy smb file: %v", err)
		}
		fmt.Printf("saved %d bytes to %s\n", n, *outPath)
	}

	if strings.TrimSpace(*dbPath) != "" || strings.TrimSpace(*profileID) != "" {
		if strings.TrimSpace(*dbPath) == "" || strings.TrimSpace(*profileID) == "" {
			fail("--db and --profile-id must be used together")
		}
		result, err := importCSVToDB(data, *dbPath, *profileID, *accountName)
		if err != nil {
			fail("import csv: %v", err)
		}
		fmt.Printf(
			"imported=%d skipped_duplicates=%d skipped_invalid=%d created_categories=%d created_account=%t account=%s\n",
			result.Imported,
			result.SkippedDuplicates,
			result.SkippedInvalid,
			result.CreatedCategories,
			result.CreatedAccount,
			result.AccountName,
		)
		return
	}

	if strings.TrimSpace(*outPath) == "" {
		fail("nothing to do: provide --preview-lines, --out, or --db with --profile-id")
	}
}

func normalizePath(path, mountName string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "./")
	if strings.HasPrefix(path, mountName+"/") {
		return strings.TrimPrefix(path, mountName+"/")
	}
	if path == mountName {
		return ""
	}
	return path
}

func printPreview(r io.Reader, lines int) {
	data, err := io.ReadAll(r)
	if err != nil {
		fail("read preview: %v", err)
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	rows := strings.Split(content, "\n")
	for i, row := range rows {
		if i >= lines {
			break
		}
		fmt.Println(row)
	}
}

type importResult struct {
	Imported          int
	SkippedDuplicates int
	SkippedInvalid    int
	CreatedCategories int
	CreatedAccount    bool
	AccountName       string
}

type expenseAccount struct {
	ID   string
	Name string
}

type expenseCategory struct {
	ID   string
	Name string
	Type string
}

func importCSVToDB(data []byte, dbPath, profileID, accountName string) (*importResult, error) {
	rows, err := parseRows(data)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var profileExists int
	if err := db.QueryRow(`SELECT COUNT(*) FROM expense_profiles WHERE id = ?`, profileID).Scan(&profileExists); err != nil {
		return nil, err
	}
	if profileExists == 0 {
		return nil, fmt.Errorf("expense profile %s not found", profileID)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	account, createdAccount, err := ensureImportAccount(tx, profileID, accountName)
	if err != nil {
		return nil, err
	}

	categories, err := loadCategories(tx, profileID)
	if err != nil {
		return nil, err
	}

	categoryNames := collectCategories(rows)
	createdCategories := 0
	for _, key := range categoryNames {
		if _, ok := categories[key]; ok {
			continue
		}
		parts := strings.SplitN(key, ":", 2)
		catType := parts[0]
		catName := parts[1]
		catID := generateID()
		_, err := tx.Exec(`
			INSERT INTO expense_categories (id, profile_id, name, type, icon, color, sort, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, catID, profileID, catName, catType, defaultCategoryIcon(catType), defaultCategoryColor(catType), 999)
		if err != nil {
			return nil, err
		}
		categories[key] = expenseCategory{ID: catID, Name: catName, Type: catType}
		createdCategories++
	}

	existing, err := loadExistingSignatures(tx, profileID)
	if err != nil {
		return nil, err
	}

	result := &importResult{
		CreatedAccount:    createdAccount,
		CreatedCategories: createdCategories,
		AccountName:       account.Name,
	}

	for _, row := range rows {
		rowType := parseType(row["类型"])
		if rowType == "" {
			result.SkippedInvalid++
			continue
		}
		date := strings.TrimSpace(row["日期"])
		if !validDate(date) {
			result.SkippedInvalid++
			continue
		}
		amount, err := strconv.ParseFloat(strings.TrimSpace(row["金额"]), 64)
		if err != nil || amount <= 0 {
			result.SkippedInvalid++
			continue
		}
		catName := strings.TrimSpace(row["分类"])
		if catName == "" {
			catName = fallbackCategoryName(rowType)
		}
		catKey := rowType + ":" + catName
		cat, ok := categories[catKey]
		if !ok {
			result.SkippedInvalid++
			continue
		}

		remark := buildRemark(row)
		sig := signature(date, rowType, amount, cat.Name, remark)
		if _, ok := existing[sig]; ok {
			result.SkippedDuplicates++
			continue
		}

		txID := generateID()
		tags := buildTags(row)
		_, err = tx.Exec(`
			INSERT INTO expense_transactions (
				id, profile_id, account_id, category_id, amount, type, date, remark, tags, voice_text, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, txID, profileID, account.ID, cat.ID, amount, rowType, date, remark, tags)
		if err != nil {
			return nil, err
		}

		if rowType == "expense" {
			_, err = tx.Exec(`UPDATE expense_accounts SET balance = balance - ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, amount, account.ID)
		} else {
			_, err = tx.Exec(`UPDATE expense_accounts SET balance = balance + ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, amount, account.ID)
		}
		if err != nil {
			return nil, err
		}

		existing[sig] = struct{}{}
		result.Imported++
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func parseRows(data []byte) ([]map[string]string, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("csv has no data rows")
	}

	headers := make([]string, len(records[0]))
	for i, header := range records[0] {
		headers[i] = strings.TrimPrefix(strings.TrimSpace(header), "\ufeff")
	}

	rows := make([]map[string]string, 0, len(records)-1)
	for _, record := range records[1:] {
		row := make(map[string]string, len(headers))
		for i, header := range headers {
			if i < len(record) {
				row[header] = strings.TrimSpace(record[i])
			} else {
				row[header] = ""
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func collectCategories(rows []map[string]string) []string {
	seen := map[string]struct{}{}
	for _, row := range rows {
		rowType := parseType(row["类型"])
		if rowType == "" {
			continue
		}
		name := strings.TrimSpace(row["分类"])
		if name == "" {
			name = fallbackCategoryName(rowType)
		}
		seen[rowType+":"+name] = struct{}{}
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func ensureImportAccount(tx *sql.Tx, profileID, accountName string) (expenseAccount, bool, error) {
	var account expenseAccount
	err := tx.QueryRow(`
		SELECT id, name FROM expense_accounts
		WHERE profile_id = ? AND name = ?
		LIMIT 1
	`, profileID, accountName).Scan(&account.ID, &account.Name)
	if err == nil {
		return account, false, nil
	}
	if err != sql.ErrNoRows {
		return expenseAccount{}, false, err
	}

	account = expenseAccount{ID: generateID(), Name: accountName}
	_, err = tx.Exec(`
		INSERT INTO expense_accounts (id, profile_id, name, type, balance, color, icon, sort, created_at, updated_at)
		VALUES (?, ?, ?, 'cash', 0, '#909399', 'Wallet', 999, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, account.ID, profileID, account.Name)
	return account, true, err
}

func loadCategories(tx *sql.Tx, profileID string) (map[string]expenseCategory, error) {
	rows, err := tx.Query(`SELECT id, name, type FROM expense_categories WHERE profile_id = ?`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]expenseCategory)
	for rows.Next() {
		var cat expenseCategory
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Type); err != nil {
			return nil, err
		}
		result[cat.Type+":"+cat.Name] = cat
	}
	return result, rows.Err()
}

func loadExistingSignatures(tx *sql.Tx, profileID string) (map[string]struct{}, error) {
	rows, err := tx.Query(`
		SELECT t.date, t.type, t.amount, COALESCE(c.name, ''), COALESCE(t.remark, '')
		FROM expense_transactions t
		LEFT JOIN expense_categories c ON c.id = t.category_id
		WHERE t.profile_id = ?
	`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]struct{})
	for rows.Next() {
		var date, rowType, catName, remark string
		var amount float64
		if err := rows.Scan(&date, &rowType, &amount, &catName, &remark); err != nil {
			return nil, err
		}
		result[signature(date, rowType, amount, catName, remark)] = struct{}{}
	}
	return result, rows.Err()
}

func parseType(raw string) string {
	raw = strings.TrimSpace(raw)
	switch raw {
	case "支出":
		return "expense"
	case "收入":
		return "income"
	default:
		return ""
	}
}

func fallbackCategoryName(rowType string) string {
	if rowType == "income" {
		return "其他收入"
	}
	return "其他支出"
}

func defaultCategoryColor(rowType string) string {
	if rowType == "income" {
		return "#67C23A"
	}
	return "#F56C6C"
}

func defaultCategoryIcon(rowType string) string {
	if rowType == "income" {
		return "Coin"
	}
	return "Folder"
}

func buildRemark(row map[string]string) string {
	parts := make([]string, 0, 2)
	remark := strings.TrimSpace(row["备注"])
	detail := strings.TrimSpace(row["详细备注"])
	if remark != "" {
		parts = append(parts, remark)
	}
	if detail != "" && detail != remark {
		parts = append(parts, detail)
	}
	return strings.Join(parts, " / ")
}

func buildTags(row map[string]string) string {
	parts := make([]string, 0, 4)
	if tm := strings.TrimSpace(row["时间"]); tm != "" {
		parts = append(parts, "time:"+tm)
	}
	if book := strings.TrimSpace(row["账本"]); book != "" {
		parts = append(parts, "book:"+book)
	}
	if reimbursement := strings.TrimSpace(row["报销"]); reimbursement != "" {
		parts = append(parts, "reimburse:"+reimbursement)
	}
	if budget := strings.TrimSpace(row["计入预算"]); budget != "" {
		parts = append(parts, "budget:"+budget)
	}
	return strings.Join(parts, ",")
}

func signature(date, rowType string, amount float64, catName, remark string) string {
	return fmt.Sprintf("%s|%s|%.2f|%s|%s", strings.TrimSpace(date), rowType, amount, strings.TrimSpace(catName), strings.TrimSpace(remark))
}

func validDate(value string) bool {
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func generateID() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		fail("generate random id: %v", err)
	}
	return hex.EncodeToString(buf)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

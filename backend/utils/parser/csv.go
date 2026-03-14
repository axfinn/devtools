package parser

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
)

// CSVParser CSV 解析器
type CSVParser struct{}

// NewCSVParser 创建 CSV 解析器
func NewCSVParser() *CSVParser {
	return &CSVParser{}
}

// CSVParseResult CSV 解析结果
type CSVParseResult struct {
	Original      string     `json:"original"`       // 原始内容
	Headers       []string   `json:"headers"`        // 表头
	Rows          [][]string  `json:"rows"`           // 数据行
	Formatted     string     `json:"formatted"`      // 格式化后的内容
	IsValid       bool       `json:"is_valid"`       // 是否为有效 CSV
	ParseError    string     `json:"parse_error"`    // 解析错误信息
	RowCount      int        `json:"row_count"`      // 行数（不含表头）
	ColumnCount   int        `json:"column_count"`   // 列数
	HasHeader     bool       `json:"has_header"`     // 是否有表头
	Delimiter     string     `json:"delimiter"`      // 分隔符
	ByteSize      int        `json:"byte_size"`      // 字节大小
	LineCount     int        `json:"line_count"`     // 行数
}

// TableData 表格数据结构
type TableData struct {
	Headers   []string   `json:"headers"`
	Rows      [][]string `json:"rows"`
	TotalRows int        `json:"total_rows"`
	TotalCols int        `json:"total_cols"`
}

// Parse 解析 CSV 内容
func (p *CSVParser) Parse(content string) *CSVParseResult {
	result := &CSVParseResult{
		Original:   content,
		IsValid:    false,
		LineCount:  strings.Count(content, "\n") + 1,
		ByteSize:   len(content),
		Delimiter:  ",",
	}

	if content == "" {
		result.ParseError = "内容为空"
		return result
	}

	// 检测分隔符
	delimiter := detectDelimiter(content)
	result.Delimiter = delimiter

	// 解析 CSV
	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = rune(delimiter[0])
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		result.ParseError = err.Error()
		return result
	}

	if len(records) == 0 {
		result.ParseError = "CSV 数据为空"
		return result
	}

	result.IsValid = true
	result.RowCount = len(records)
	result.HasHeader = true
	result.ColumnCount = len(records[0])

	// 判断第一行是否为表头
	if result.RowCount > 1 {
		// 检查第一行是否像表头（较短，全是字符串）
		firstRow := records[0]
		secondRow := records[1]

		isHeader := false
		if len(firstRow) == len(secondRow) {
			// 简单判断：第一行没有纯数字
			isHeader = true
			for _, cell := range firstRow {
				if _, err := strconv.ParseFloat(cell, 64); err == nil {
					isHeader = false
					break
				}
			}
		}

		if isHeader {
			result.Headers = firstRow
			result.Rows = records[1:]
			result.RowCount = len(result.Rows)
		} else {
			result.Headers = make([]string, len(records[0]))
			for i := range result.Headers {
				result.Headers[i] = fmt.Sprintf("Column%d", i+1)
			}
			result.Rows = records
		}
	} else {
		result.Headers = make([]string, len(records[0]))
		for i := range result.Headers {
			result.Headers[i] = fmt.Sprintf("Column%d", i+1)
		}
		result.Rows = records
	}

	// 格式化输出
	result.Formatted = p.Format(content)

	return result
}

// Format 格式化 CSV
func (p *CSVParser) Format(content string) string {
	records, err := csv.NewReader(strings.NewReader(content)).ReadAll()
	if err != nil {
		return content
	}

	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	writer.Comma = ','
	writer.UseCRLF = true

	for _, record := range records {
		writer.Write(record)
	}

	writer.Flush()
	return buf.String()
}

// ToTableData 转换为表格数据结构
func (p *CSVParser) ToTableData(content string) (*TableData, error) {
	result := p.Parse(content)
	if !result.IsValid {
		return nil, fmt.Errorf(result.ParseError)
	}

	return &TableData{
		Headers:   result.Headers,
		Rows:      result.Rows,
		TotalRows: result.RowCount,
		TotalCols: result.ColumnCount,
	}, nil
}

// ToMarkdown 将 CSV 转换为 Markdown 表格
func (p *CSVParser) ToMarkdown(content string) (string, error) {
	result := p.Parse(content)
	if !result.IsValid {
		return "", fmt.Errorf(result.ParseError)
	}

	var buf strings.Builder

	// 写入表头
	buf.WriteString("|")
	for _, header := range result.Headers {
		buf.WriteString(header)
		buf.WriteString("|")
	}
	buf.WriteString("\n")

	// 写入分隔行
	buf.WriteString("|")
	for range result.Headers {
		buf.WriteString("---|")
	}
	buf.WriteString("\n")

	// 写入数据行
	for _, row := range result.Rows {
		buf.WriteString("|")
		for _, cell := range row {
			// 转义 | 字符
			cell = strings.ReplaceAll(cell, "|", "\\|")
			// 转义换行
			cell = strings.ReplaceAll(cell, "\n", "<br>")
			buf.WriteString(cell)
			buf.WriteString("|")
		}
		buf.WriteString("\n")
	}

	return buf.String(), nil
}

// ToHTML 将 CSV 转换为 HTML 表格
func (p *CSVParser) ToHTML(content string) (string, error) {
	result := p.Parse(content)
	if !result.IsValid {
		return "", fmt.Errorf(result.ParseError)
	}

	var buf strings.Builder
	buf.WriteString("<table class=\"csv-table\">\n")

	// 写入表头
	buf.WriteString("  <thead>\n    <tr>\n")
	for _, header := range result.Headers {
		buf.WriteString("      <th>")
		buf.WriteString(escapeHTML(header))
		buf.WriteString("</th>\n")
	}
	buf.WriteString("    </tr>\n  </thead>\n")

	// 写入数据行
	buf.WriteString("  <tbody>\n")
	for _, row := range result.Rows {
		buf.WriteString("    <tr>\n")
		for _, cell := range row {
			buf.WriteString("      <td>")
			buf.WriteString(escapeHTML(cell))
			buf.WriteString("</td>\n")
		}
		buf.WriteString("    </tr>\n")
	}
	buf.WriteString("  </tbody>\n")

	buf.WriteString("</table>")
	return buf.String(), nil
}

// MergeCSV 合并多个 CSV
func (p *CSVParser) MergeCSV(csvContents []string) (string, error) {
	if len(csvContents) == 0 {
		return "", fmt.Errorf("没有内容可合并")
	}

	// 解析所有 CSV
	var allResults []*CSVParseResult
	for _, content := range csvContents {
		result := p.Parse(content)
		if !result.IsValid {
			return "", fmt.Errorf(result.ParseError)
		}
		allResults = append(allResults, result)
	}

	// 使用第一个 CSV 的表头
	headers := allResults[0].Headers

	// 合并所有数据行
	var mergedRows [][]string
	for _, result := range allResults {
		mergedRows = append(mergedRows, result.Rows...)
	}

	// 生成新的 CSV
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	writer.Write(headers)
	for _, row := range mergedRows {
		writer.Write(row)
	}

	writer.Flush()
	return buf.String(), nil
}

// FilterCSV 过滤 CSV 数据
func (p *CSVParser) FilterCSV(content string, columnIndex int, value string) (string, error) {
	result := p.Parse(content)
	if !result.IsValid {
		return "", fmt.Errorf(result.ParseError)
	}

	if columnIndex < 0 || columnIndex >= result.ColumnCount {
		return "", fmt.Errorf("列索引无效")
	}

	var filteredRows [][]string
	for _, row := range result.Rows {
		if len(row) > columnIndex && strings.Contains(row[columnIndex], value) {
			filteredRows = append(filteredRows, row)
		}
	}

	// 生成新的 CSV
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	writer.Write(result.Headers)
	for _, row := range filteredRows {
		writer.Write(row)
	}

	writer.Flush()
	return buf.String(), nil
}

// SortCSV 排序 CSV 数据
func (p *CSVParser) SortCSV(content string, columnIndex int, ascending bool) (string, error) {
	result := p.Parse(content)
	if !result.IsValid {
		return "", fmt.Errorf(result.ParseError)
	}

	if columnIndex < 0 || columnIndex >= result.ColumnCount {
		return "", fmt.Errorf("列索引无效")
	}

	// 简单排序
	sortedRows := make([][]string, len(result.Rows))
	copy(sortedRows, result.Rows)

	for i := 0; i < len(sortedRows)-1; i++ {
		for j := i + 1; j < len(sortedRows); j++ {
			compare := strings.Compare(sortedRows[i][columnIndex], sortedRows[j][columnIndex])
			if ascending && compare > 0 || !ascending && compare < 0 {
				sortedRows[i], sortedRows[j] = sortedRows[j], sortedRows[i]
			}
		}
	}

	// 生成新的 CSV
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	writer.Write(result.Headers)
	for _, row := range sortedRows {
		writer.Write(row)
	}

	writer.Flush()
	return buf.String(), nil
}

// Validate 验证 CSV 格式
func (p *CSVParser) Validate(content string) (bool, string) {
	if content == "" {
		return false, "内容为空"
	}

	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = -1 // 不强制字段数一致

	_, err := reader.ReadAll()
	if err != nil {
		return false, err.Error()
	}

	return true, ""
}

// GetStatistics 获取 CSV 统计信息
func (p *CSVParser) GetStatistics(content string) (map[string]interface{}, error) {
	result := p.Parse(content)
	if !result.IsValid {
		return nil, fmt.Errorf(result.ParseError)
	}

	stats := map[string]interface{}{
		"rows":         result.RowCount,
		"columns":      result.ColumnCount,
		"has_header":   result.HasHeader,
		"delimiter":    result.Delimiter,
		"empty_cells":  0,
		"numeric_cols": []int{},
	}

	// 统计空单元格
	emptyCount := 0
	for _, row := range result.Rows {
		for _, cell := range row {
			if strings.TrimSpace(cell) == "" {
				emptyCount++
			}
		}
	}
	stats["empty_cells"] = emptyCount

	// 检查数值列
	var numericCols []int
	for col := 0; col < result.ColumnCount; col++ {
		isNumeric := true
		for _, row := range result.Rows {
			if len(row) <= col {
				continue
			}
			if _, err := strconv.ParseFloat(row[col], 64); err != nil {
				isNumeric = false
				break
			}
		}
		if isNumeric && len(result.Rows) > 0 {
			numericCols = append(numericCols, col)
		}
	}
	stats["numeric_cols"] = numericCols

	return stats, nil
}

// detectDelimiter 检测 CSV 分隔符
func detectDelimiter(content string) string {
	firstLine := strings.Split(content, "\n")[0]

	// 统计常见分隔符出现次数
	delimiters := []string{",", ";", "\t", "|"}
	maxCount := 0
	bestDelimiter := ","

	for _, d := range delimiters {
		count := strings.Count(firstLine, d)
		if count > maxCount {
			maxCount = count
			bestDelimiter = d
		}
	}

	return bestDelimiter
}

// escapeHTML 转义 HTML 特殊字符
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// ConvertToTSV 将 CSV 转换为 TSV
func (p *CSVParser) ConvertToTSV(content string) (string, error) {
	result := p.Parse(content)
	if !result.IsValid {
		return "", fmt.Errorf(result.ParseError)
	}

	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	writer.Comma = '\t'

	for _, row := range result.Rows {
		writer.Write(row)
	}

	writer.Flush()
	return buf.String(), nil
}

// ConvertFromTSV 将 TSV 转换为 CSV
func (p *CSVParser) ConvertFromTSV(content string) (string, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = '\t'

	records, err := reader.ReadAll()
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	for _, record := range records {
		writer.Write(record)
	}

	writer.Flush()
	return buf.String(), nil
}

// DetectHeaders 检测并返回可能的表头行
func (p *CSVParser) DetectHeaders(content string) []string {
	result := p.Parse(content)
	if !result.IsValid {
		return nil
	}
	return result.Headers
}

// FillMissingValues 填充缺失值
func (p *CSVParser) FillMissingValues(content string, fillValue string) (string, error) {
	result := p.Parse(content)
	if !result.IsValid {
		return "", fmt.Errorf(result.ParseError)
	}

	maxCols := result.ColumnCount
	var filledRows [][]string

	for _, row := range result.Rows {
		if len(row) < maxCols {
			// 填充缺失值
			filledRow := make([]string, maxCols)
			copy(filledRow, row)
			for i := len(row); i < maxCols; i++ {
				filledRow[i] = fillValue
			}
			filledRows = append(filledRows, filledRow)
		} else {
			filledRows = append(filledRows, row)
		}
	}

	// 生成新的 CSV
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	writer.Write(result.Headers)
	for _, row := range filledRows {
		writer.Write(row)
	}

	writer.Flush()
	return buf.String(), nil
}

// GetColumn 获取指定列
func (p *CSVParser) GetColumn(content string, columnIndex int) ([]string, error) {
	result := p.Parse(content)
	if !result.IsValid {
		return nil, fmt.Errorf(result.ParseError)
	}

	if columnIndex < 0 || columnIndex >= result.ColumnCount {
		return nil, fmt.Errorf("列索引无效")
	}

	var column []string
	for _, row := range result.Rows {
		if len(row) > columnIndex {
			column = append(column, row[columnIndex])
		}
	}

	return column, nil
}

// ToJSON 将 CSV 转换为 JSON 数组
func (p *CSVParser) ToJSON(content string) (string, error) {
	result := p.Parse(content)
	if !result.IsValid {
		return "", fmt.Errorf(result.ParseError)
	}

	type RowData map[string]string
	var rows []RowData

	for _, row := range result.Rows {
		rowData := make(RowData)
		for i, header := range result.Headers {
			if i < len(row) {
				rowData[header] = row[i]
			} else {
				rowData[header] = ""
			}
		}
		rows = append(rows, rowData)
	}

	// 转换为 JSON
	return toJSON(rows)
}

// toJSON 转换为 JSON（简单实现）
func toJSON(data interface{}) (string, error) {
	switch v := data.(type) {
	case []map[string]string:
		var result strings.Builder
		result.WriteString("[")
		for i, row := range v {
			if i > 0 {
				result.WriteString(",")
			}
			result.WriteString("{")
			keys := make([]string, 0, len(row))
			for k := range row {
				keys = append(keys, k)
			}
			for j, k := range keys {
				if j > 0 {
					result.WriteString(",")
				}
				result.WriteString("\"")
				result.WriteString(k)
				result.WriteString("\":\"")
				result.WriteString(strings.ReplaceAll(row[k], "\"", "\\\""))
				result.WriteString("\"")
			}
			result.WriteString("}")
		}
		result.WriteString("]")
		return result.String(), nil
	default:
		return "", fmt.Errorf("不支持的类型")
	}
}

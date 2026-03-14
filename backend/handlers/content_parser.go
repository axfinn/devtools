package handlers

import (
	"net/http"

	"devtools/utils/parser"

	"github.com/gin-gonic/gin"
)

// ContentParserHandler 内容解析处理器
type ContentParserHandler struct {
	parser *parser.ContentParser
}

// NewContentParserHandler 创建内容解析处理器
func NewContentParserHandler() *ContentParserHandler {
	return &ContentParserHandler{
		parser: parser.NewContentParser(),
	}
}

// ParseRequest 解析请求
type ParseRequest struct {
	Content string `json:"content" binding:"required"`
}

// ParseResponse 解析响应
type ParseResponse struct {
	ContentType string      `json:"content_type"`
	IsValid     bool        `json:"is_valid"`
	CanParse    bool        `json:"can_parse"`
	ParsedData interface{} `json:"parsed_data,omitempty"`
	Metadata   interface{} `json:"metadata,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// ParseContent 解析内容
func (h *ContentParserHandler) ParseContent(c *gin.Context) {
	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	// 内容限制（1MB）
	if len(req.Content) > 1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容超过 1MB 限制", "code": 400})
		return
	}

	result := h.parser.Parse(req.Content)

	c.JSON(http.StatusOK, ParseResponse{
		ContentType: string(result.ContentType),
		IsValid:     result.IsValid,
		CanParse:    result.CanParse,
		ParsedData:  result.ParsedData,
		Metadata:    result.Metadata,
		Error:       result.Error,
	})
}

// DetectContentType 检测内容类型
func (h *ContentParserHandler) DetectContentType(c *gin.Context) {
	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	contentType := h.parser.DetectContentType(req.Content)

	c.JSON(http.StatusOK, gin.H{
		"content_type": contentType,
	})
}

// FormatContent 格式化内容
func (h *ContentParserHandler) FormatContent(c *gin.Context) {
	var req struct {
		Content     string `json:"content" binding:"required"`
		ContentType string `json:"content_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	var contentType parser.ContentType
	if req.ContentType != "" {
		contentType = parser.ContentType(req.ContentType)
	} else {
		contentType = h.parser.DetectContentType(req.Content)
	}

	formatted := h.parser.FormatContent(req.Content, contentType)

	c.JSON(http.StatusOK, gin.H{
		"formatted": formatted,
	})
}

// ConvertContent 转换内容格式
func (h *ContentParserHandler) ConvertContent(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
		FromType string `json:"from_type" binding:"required"`
		ToType   string `json:"to_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	result, err := h.parser.ConvertContent(req.Content, parser.ContentType(req.FromType), parser.ContentType(req.ToType))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result": result,
	})
}

// ParseMarkdown 解析 Markdown
func (h *ContentParserHandler) ParseMarkdown(c *gin.Context) {
	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	mdParser := parser.NewMarkdownParser()
	result := mdParser.Parse(req.Content)

	c.JSON(http.StatusOK, result)
}

// ParseJSON 解析 JSON
func (h *ContentParserHandler) ParseJSON(c *gin.Context) {
	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	jsonParser := parser.NewJSONParser()
	result := jsonParser.Parse(req.Content)

	c.JSON(http.StatusOK, result)
}

// ParseXML 解析 XML
func (h *ContentParserHandler) ParseXML(c *gin.Context) {
	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	xmlParser := parser.NewXMLParser()
	result := xmlParser.Parse(req.Content)

	c.JSON(http.StatusOK, result)
}

// ParseCSV 解析 CSV
func (h *ContentParserHandler) ParseCSV(c *gin.Context) {
	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	csvParser := parser.NewCSVParser()
	result := csvParser.Parse(req.Content)

	c.JSON(http.StatusOK, result)
}

// ParseRichText 解析富文本
func (h *ContentParserHandler) ParseRichText(c *gin.Context) {
	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	rtParser := parser.NewRichTextParser()
	result := rtParser.Parse(req.Content)

	c.JSON(http.StatusOK, result)
}

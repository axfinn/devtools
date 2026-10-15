package handlers

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

// SSRuleHandler 管理「Shadowsocks 规则文件」的编辑与公开分发。
//
// 本模块刻意保持极简:全站只有一份规则文件,页面填域名列表、生成规则正文,
// 公开链接给客户端直接拉。不做多套规则、不做规则类型建模。
type SSRuleHandler struct {
	db            *models.DB
	adminPassword string
}

func NewSSRuleHandler(db *models.DB, adminPassword string) *SSRuleHandler {
	return &SSRuleHandler{db: db, adminPassword: adminPassword}
}

// ssRuleFileNames 公开链接允许的文件名。
//
// 同一份内容用多个扩展名暴露:不同客户端对「规则文件 URL」的后缀要求不一致,
// 内容本身是纯文本,换个后缀就能被另一批客户端接受。
//
// 这里**刻意不含 .pac**:PAC 是 JavaScript,把 ACL 正文当 .pac 发出去,
// 客户端会静默拿到一个语法错误的脚本,报错信息完全指不到这里。
var ssRuleFileNames = []string{"rules.acl", "rules.txt", "rules.list", "rules.conf"}

// ssRuleFileContentType 全部是纯文本。ACL / txt / list / conf 对客户端而言
// 都是「一行一条的文本」,没有各自的 MIME 类型。
const ssRuleFileContentType = "text/plain; charset=utf-8"

// ssRuleIsAllowedFile 校验文件名是否在白名单内。
func ssRuleIsAllowedFile(name string) bool {
	for _, n := range ssRuleFileNames {
		if n == name {
			return true
		}
	}
	return false
}

// checkAdmin 校验管理密码。
//
// 走 header X-Super-Admin-Password(已在 CORS 白名单里),兼容 query 便于 curl 调试。
// 用定长比较而非 ==,避免时序侧信道。
func (h *SSRuleHandler) checkAdmin(c *gin.Context) bool {
	if h.adminPassword == "" {
		return false // 未配置密码 = 功能关闭,而不是「无密码放行」
	}
	given := c.GetHeader("X-Super-Admin-Password")
	if given == "" {
		given = c.Query("admin_password")
	}
	return subtle.ConstantTimeCompare([]byte(given), []byte(h.adminPassword)) == 1
}

type ssRuleSaveRequest struct {
	Source string `json:"source"`
}

// ssRuleView 是管理接口统一响应体。
//
// Paths 返回相对路径,由前端拼上 window.location.origin —— 服务端不知道
// 用户是从哪个域名访问的(可能走 CF、可能走内网),拼死反而会给出错链接。
type ssRuleView struct {
	Source    string    `json:"source"`
	Content   string    `json:"content"`
	Token     string    `json:"token"`
	UpdatedAt time.Time `json:"updated_at"`
	Paths     []string  `json:"paths"`
	Skipped   []string  `json:"skipped"`
}

func newSSRuleView(file *models.SSRuleFile, skipped []string) ssRuleView {
	paths := make([]string, 0, len(ssRuleFileNames))
	for _, name := range ssRuleFileNames {
		paths = append(paths, "/ss/"+file.Token+"/"+name)
	}
	if skipped == nil {
		skipped = []string{}
	}
	return ssRuleView{
		Source:    file.Source,
		Content:   file.Content,
		Token:     file.Token,
		UpdatedAt: file.UpdatedAt,
		Paths:     paths,
		Skipped:   skipped,
	}
}

// Verify POST /api/ssrules/verify —— 管理页密码门。
func (h *SSRuleHandler) Verify(c *gin.Context) {
	if !h.checkAdmin(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Get GET /api/ssrules —— 读取当前规则文件(含原始输入,用于回显编辑框)。
func (h *SSRuleHandler) Get(c *gin.Context) {
	if !h.checkAdmin(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	file, err := h.db.GetSSRuleFile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取规则文件失败"})
		return
	}

	// 历史内容可能是在校验放宽前存进去的,回显时重跑一遍生成,
	// 让页面上的 skipped 提示与现实一致。
	content, skipped, err := buildACLRules(file.Source)
	if err != nil {
		content, skipped = file.Content, []string{}
	}
	file.Content = content

	c.JSON(http.StatusOK, newSSRuleView(file, skipped))
}

// Save POST /api/ssrules —— 保存域名列表,重新生成规则正文。
func (h *SSRuleHandler) Save(c *gin.Context) {
	if !h.checkAdmin(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	var req ssRuleSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	content, skipped, err := buildACLRules(req.Source)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file, err := h.db.SaveSSRuleFile(req.Source, content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, newSSRuleView(file, skipped))
}

// RegenerateToken POST /api/ssrules/regenerate-token —— 换发 token。
// 公开链接外泄时的止损手段:换完旧链接立刻失效。
func (h *SSRuleHandler) RegenerateToken(c *gin.Context) {
	if !h.checkAdmin(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	file, err := h.db.RegenerateSSRuleToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "换发失败"})
		return
	}
	c.JSON(http.StatusOK, newSSRuleView(file, nil))
}

// Download GET /ss/:token/:filename —— 公开分发规则文件,不需密码。
//
// 安全性依赖 token 的不可枚举性(32 位随机 hex)。客户端拉规则时基本不支持
// 交互式认证,所以这里不能要求密码,只能靠 token。
func (h *SSRuleHandler) Download(c *gin.Context) {
	filename := c.Param("filename")
	if !ssRuleIsAllowedFile(filename) {
		c.String(http.StatusNotFound, "规则文件不存在")
		return
	}

	file, err := h.db.GetSSRuleFileByToken(c.Param("token"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.String(http.StatusNotFound, "规则文件不存在")
			return
		}
		c.String(http.StatusInternalServerError, "读取规则文件失败")
		return
	}

	// no-store:用户改完规则后客户端下次拉取必须拿到新内容,
	// 缓存住会让「改了规则但客户端没生效」变成极难排查的问题。
	c.Header("Content-Type", ssRuleFileContentType)
	c.Header("Cache-Control", "no-store")
	c.String(http.StatusOK, file.Content)
}

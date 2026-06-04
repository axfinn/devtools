package handlers

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/smtp"
	"regexp"
	"strconv"
	"strings"
	"time"

	"crypto/tls"

	"devtools/config"
	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

// AskitSyncHandler 处理 AskIt 扩展云同步：账号鉴权 + 增量同步。
// 注册受 cfg.RegistrationMode 控制(closed|invite|open)，默认 closed 不对外开放。
type AskitSyncHandler struct {
	db  *models.DB
	cfg config.AskitSyncConfig
}

func NewAskitSyncHandler(db *models.DB, cfg config.AskitSyncConfig) *AskitSyncHandler {
	return &AskitSyncHandler{db: db, cfg: cfg}
}

const askitUserIDKey = "askit_user_id"

var askitEmailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func askitNowMs() int64 { return time.Now().UnixMilli() }

// genNumericCode 生成 6 位数字验证码。
func genNumericCode() string {
	var sb strings.Builder
	for i := 0; i < 6; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		sb.WriteString(strconv.FormatInt(n.Int64(), 10))
	}
	return sb.String()
}

// issueTokens 为用户签发 access + refresh 不透明令牌并返回标准登录响应体。
func (h *AskitSyncHandler) issueTokens(userID string) (gin.H, error) {
	now := time.Now()
	access := utils.GenerateHexKey(32)
	refresh := utils.GenerateHexKey(32)
	accessExp := now.Add(time.Duration(h.cfg.AccessTTLHours) * time.Hour).UnixMilli()
	refreshExp := now.Add(time.Duration(h.cfg.RefreshTTLHours) * time.Hour).UnixMilli()
	if err := h.db.CreateAskitToken(access, userID, "access", accessExp, now.UnixMilli()); err != nil {
		return nil, err
	}
	if err := h.db.CreateAskitToken(refresh, userID, "refresh", refreshExp, now.UnixMilli()); err != nil {
		return nil, err
	}
	return gin.H{
		"accessToken":     access,
		"refreshToken":    refresh,
		"accessExpiresAt": accessExp,
	}, nil
}

// sendCodeMail 通过配置的 SMTP 发送验证码/重置码邮件。SMTP 未配置则返回错误。
func (h *AskitSyncHandler) sendCodeMail(to, subject, body string) error {
	if h.cfg.SMTPHost == "" || h.cfg.SMTPUser == "" || h.cfg.SMTPPass == "" {
		return fmt.Errorf("smtp not configured")
	}
	port := h.cfg.SMTPPort
	if port == 0 {
		port = 465
	}
	addr := net.JoinHostPort(h.cfg.SMTPHost, strconv.Itoa(port))
	msg := buildSMTPMessage(h.cfg.SMTPUser, []string{to}, subject, body, nil)

	var (
		conn   net.Conn
		client *smtp.Client
		err    error
	)
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	tlsCfg := &tls.Config{ServerName: h.cfg.SMTPHost}
	if port == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
		if err != nil {
			return err
		}
		client, err = smtp.NewClient(conn, h.cfg.SMTPHost)
	} else {
		conn, err = dialer.Dial("tcp", addr)
		if err != nil {
			return err
		}
		client, err = smtp.NewClient(conn, h.cfg.SMTPHost)
		if err == nil {
			if ok, _ := client.Extension("STARTTLS"); ok {
				if err = client.StartTLS(tlsCfg); err != nil {
					client.Close()
					return err
				}
			}
		}
	}
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension("AUTH"); ok {
		auth := smtp.PlainAuth("", h.cfg.SMTPUser, h.cfg.SMTPPass, h.cfg.SMTPHost)
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	if err = client.Mail(h.cfg.SMTPUser); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = writer.Write([]byte(msg)); err != nil {
		writer.Close()
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// ── 鉴权接口(无密码 · 邮箱验证码登录)─────────────────────────────
//
// 登录即注册：输入邮箱 → 收 6 位验证码 → 提交验证码即登录。
//   · 邮箱已有账号        → 直接登录。
//   · 邮箱无账号 + open    → 自动注册后登录。
//   · 邮箱无账号 + invite  → 校验邀请码，通过则自动注册后登录。
//   · 邮箱无账号 + closed  → 拒绝(registration_closed)。
// 整个流程不涉及密码;验证码本身即邮箱所有权证明,注册即视为已验证。

type askitRequestCodeReq struct {
	Email      string `json:"email"`
	InviteCode string `json:"inviteCode"`
}

// gateNewEmail 按注册模式判定一个「未注册邮箱」能否进入注册流程。
// 返回 (errorCode, httpStatus);允许时 errorCode 为空。
func (h *AskitSyncHandler) gateNewEmail(inviteCode string) (string, int) {
	switch h.cfg.RegistrationMode {
	case "open":
		return "", 0
	case "invite":
		ok, err := h.db.CheckAskitInviteCode(strings.TrimSpace(inviteCode), askitNowMs())
		if err != nil {
			return "server_error", http.StatusInternalServerError
		}
		if !ok {
			return "invalid_invite_code", http.StatusForbidden
		}
		return "", 0
	default: // closed
		return "registration_closed", http.StatusForbidden
	}
}

// RequestCode POST /auth/request-code —— 下发登录验证码。
// 邮箱无账号时按注册模式门控(closed 直接拒绝),已有账号直接放行下发。
func (h *AskitSyncHandler) RequestCode(c *gin.Context) {
	var req askitRequestCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !askitEmailRe.MatchString(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_email"})
		return
	}

	user, err := h.db.GetAskitUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	// 未注册邮箱:按注册模式决定能否继续(关闭即失败)。
	if user == nil {
		if code, status := h.gateNewEmail(req.InviteCode); code != "" {
			c.JSON(status, gin.H{"error": code})
			return
		}
	}

	code := genNumericCode()
	exp := time.Now().Add(time.Duration(h.cfg.CodeTTLMinutes) * time.Minute).UnixMilli()
	if err := h.db.UpsertAskitEmailCode(req.Email, code, "login", exp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	if err := h.sendCodeMail(req.Email, "AskIt 登录验证码", fmt.Sprintf("你的登录验证码是 %s，%d 分钟内有效。若非本人操作请忽略。", code, h.cfg.CodeTTLMinutes)); err != nil {
		c.JSON(http.StatusOK, gin.H{"emailSent": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"emailSent": true})
}

type askitLoginCodeReq struct {
	Email      string `json:"email"`
	Code       string `json:"code"`
	InviteCode string `json:"inviteCode"`
}

// LoginCode POST /auth/login-code —— 校验验证码并登录。
// 邮箱无账号则自动注册(没有就注册);注册再次受门控(关闭就失败)。
func (h *AskitSyncHandler) LoginCode(c *gin.Context) {
	var req askitLoginCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !askitEmailRe.MatchString(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_email"})
		return
	}

	ok, err := h.db.ConsumeAskitEmailCode(req.Email, strings.TrimSpace(req.Code), "login", askitNowMs())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_code"})
		return
	}

	user, err := h.db.GetAskitUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// 没有就注册:验证码已证明邮箱所有权,直接建号并置 verified。
	if user == nil {
		if code, status := h.gateNewEmail(req.InviteCode); code != "" {
			c.JSON(status, gin.H{"error": code})
			return
		}
		userID := utils.GenerateHexKey(16)
		if err := h.db.CreateAskitUser(userID, req.Email, "", true, askitNowMs()); err != nil {
			if err == models.ErrAskitEmailTaken {
				// 并发下已被建号,重新读取继续登录。
				if u, e := h.db.GetAskitUserByEmail(req.Email); e == nil && u != nil {
					user = u
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
					return
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
				return
			}
		} else {
			if h.cfg.RegistrationMode == "invite" {
				_ = h.db.MarkAskitInviteUsed(strings.TrimSpace(req.InviteCode), userID, askitNowMs())
			}
			user = &models.AskitUser{ID: userID, Email: req.Email, Verified: true}
		}
	}

	tokens, err := h.issueTokens(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, tokens)
}

type askitRefreshReq struct {
	RefreshToken string `json:"refreshToken"`
}

// Refresh POST /auth/refresh —— 用 refresh token 换新 access token。
func (h *AskitSyncHandler) Refresh(c *gin.Context) {
	var req askitRefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	userID, err := h.db.GetAskitTokenUser(req.RefreshToken, "refresh", askitNowMs())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}
	now := time.Now()
	access := utils.GenerateHexKey(32)
	accessExp := now.Add(time.Duration(h.cfg.AccessTTLHours) * time.Hour).UnixMilli()
	if err := h.db.CreateAskitToken(access, userID, "access", accessExp, now.UnixMilli()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"accessToken": access, "accessExpiresAt": accessExp})
}

// Me GET /auth/me —— 返回当前用户信息(需 Bearer)。
func (h *AskitSyncHandler) Me(c *gin.Context) {
	userID := c.GetString(askitUserIDKey)
	user, err := h.db.GetAskitUserByID(userID)
	if err != nil || user == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":        user.ID,
		"email":     user.Email,
		"verified":  user.Verified,
		"createdAt": user.CreatedAt,
	})
}

// Logout POST /auth/logout —— 吊销当前 access token。
func (h *AskitSyncHandler) Logout(c *gin.Context) {
	token := askitBearerToken(c)
	if token != "" {
		_ = h.db.DeleteAskitToken(token)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── Bearer 中间件 ─────────────────────────────────────────────

func askitBearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// AuthMiddleware 校验 access token，注入 askit_user_id。
func (h *AskitSyncHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := askitBearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing_token"})
			return
		}
		userID, err := h.db.GetAskitTokenUser(token, "access", askitNowMs())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
			return
		}
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			return
		}
		c.Set(askitUserIDKey, userID)
		c.Next()
	}
}

// ── 同步接口 ─────────────────────────────────────────────

// Pull GET /sync/pull?since=<version> —— 拉取增量(含墓碑)。
func (h *AskitSyncHandler) Pull(c *gin.Context) {
	userID := c.GetString(askitUserIDKey)
	since, _ := strconv.ParseInt(c.Query("since"), 10, 64)
	collections, serverVersion, err := h.db.PullAskitChanges(userID, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"serverVersion": serverVersion,
		"collections":   collections,
	})
}

type askitPushReq struct {
	BaseVersion int64                          `json:"baseVersion"`
	Collections models.AskitCollectionRecords `json:"collections"`
}

// Push POST /sync/push —— 记录级 LWW 合并 + 墓碑 + version++。
func (h *AskitSyncHandler) Push(c *gin.Context) {
	userID := c.GetString(askitUserIDKey)
	var req askitPushReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	res, err := h.db.PushAskitChanges(userID, req.Collections, h.cfg.BlobMaxBytes)
	if err != nil {
		if err == models.ErrAskitBlobTooLarge {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "blob_too_large"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"serverVersion": res.ServerVersion,
		"applied":       res.Applied,
		"conflicts":     res.Conflicts,
	})
}

// Snapshot GET /sync/snapshot —— 全量非墓碑快照(调试/迁移)。
func (h *AskitSyncHandler) Snapshot(c *gin.Context) {
	userID := c.GetString(askitUserIDKey)
	collections, serverVersion, err := h.db.GetAskitSnapshot(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"serverVersion": serverVersion,
		"collections":   collections,
	})
}

// ── 邀请码管理(需 admin_password)─────────────────────────────

func (h *AskitSyncHandler) checkAdmin(c *gin.Context) bool {
	if h.cfg.AdminPassword == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin_disabled"})
		return false
	}
	pw := c.GetHeader("X-Admin-Password")
	if pw != h.cfg.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return false
	}
	return true
}

type askitCreateInviteReq struct {
	Count       int    `json:"count"`
	ExpiresDays int    `json:"expiresDays"`
}

// CreateInvites POST /admin/invites —— 批量生成邀请码。
func (h *AskitSyncHandler) CreateInvites(c *gin.Context) {
	if !h.checkAdmin(c) {
		return
	}
	var req askitCreateInviteReq
	_ = c.ShouldBindJSON(&req)
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 100 {
		req.Count = 100
	}
	now := askitNowMs()
	var expiresAt *int64
	if req.ExpiresDays > 0 {
		exp := time.Now().AddDate(0, 0, req.ExpiresDays).UnixMilli()
		expiresAt = &exp
	}
	codes := make([]string, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		code := utils.GenerateHexKey(8)
		if err := h.db.CreateAskitInviteCode(code, now, expiresAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
			return
		}
		codes = append(codes, code)
	}
	c.JSON(http.StatusOK, gin.H{"codes": codes})
}

// AdminUsersOverview GET /admin/users —— 返回所有用户的备份「元数据」视图。
// 只暴露有没有备份、密钥备份与更新时间,绝不返回任何 data 内容(密钥端到端加密,
// 服务器也无法解密)。供管理页确认「密钥同步是否生效」。
func (h *AskitSyncHandler) AdminUsersOverview(c *gin.Context) {
	if !h.checkAdmin(c) {
		return
	}
	users, err := h.db.GetAskitUsersOverview()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

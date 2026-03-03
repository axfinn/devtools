package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

type ExpenseHandler struct {
	db             *models.DB
	defaultExpDays int
	maxDataSize    int
	maxPerIP       int
	ipWindow       time.Duration
	cfg            *config.Config
}

func NewExpenseHandler(db *models.DB, cfg *config.Config) *ExpenseHandler {
	expenseCfg := cfg.Expense
	if expenseCfg.DefaultExpiresDays <= 0 {
		expenseCfg.DefaultExpiresDays = 365
	}
	if expenseCfg.MaxDataSize <= 0 {
		expenseCfg.MaxDataSize = 1024 * 1024
	}
	return &ExpenseHandler{
		db:             db,
		defaultExpDays: expenseCfg.DefaultExpiresDays,
		maxDataSize:    expenseCfg.MaxDataSize,
		maxPerIP:       5,
		ipWindow:       time.Hour,
		cfg:            cfg,
	}
}

func expensePasswordIndex(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}

// getPassword 获取密码，支持从 Query 或 Header 获取
func getPassword(c *gin.Context) string {
	// 优先从 Header 获取
	if p := c.GetHeader("X-Password"); p != "" {
		return p
	}
	// 其次从 Query 参数获取
	return c.Query("password")
}

// Request/Response types

type CreateExpenseRequest struct {
	Password string `json:"password" binding:"required"`
}

type CreateExpenseResponse struct {
	ID         string     `json:"id"`
	CreatorKey string     `json:"creator_key"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

type LoginExpenseRequest struct {
	Password string `json:"password" binding:"required"`
}

type ExpenseProfileResponse struct {
	ID         string     `json:"id"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type CreateAccountRequest struct {
	Name    string  `json:"name" binding:"required"`
	Type    string  `json:"type"`
	Balance float64 `json:"balance"`
	Color   string  `json:"color"`
	Icon    string  `json:"icon"`
	Sort    int     `json:"sort"`
}

type UpdateAccountRequest struct {
	Name    string  `json:"name"`
	Type    string  `json:"type"`
	Balance float64 `json:"balance"`
	Color   string  `json:"color"`
	Icon    string  `json:"icon"`
	Sort    int     `json:"sort"`
}

type CreateCategoryRequest struct {
	Name  string `json:"name" binding:"required"`
	Type  string `json:"type" binding:"required"` // income, expense
	Icon  string `json:"icon"`
	Color string `json:"color"`
	Sort  int    `json:"sort"`
}

type UpdateCategoryRequest struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
	Sort  int    `json:"sort"`
}

type CreateTransactionRequest struct {
	AccountID  string  `json:"account_id" binding:"required"`
	CategoryID string  `json:"category_id" binding:"required"`
	Amount     float64 `json:"amount" binding:"required"`
	Type       string  `json:"type" binding:"required"` // income, expense
	Date       string  `json:"date" binding:"required"` // YYYY-MM-DD
	Remark     string  `json:"remark"`
	Tags       string  `json:"tags"`
}

type UpdateTransactionRequest struct {
	AccountID  string  `json:"account_id"`
	CategoryID string  `json:"category_id"`
	Amount     float64 `json:"amount"`
	Type       string  `json:"type"`
	Date       string  `json:"date"`
	Remark     string  `json:"remark"`
	Tags       string  `json:"tags"`
}

type AnalyzeRequest struct {
	Period string `json:"period"` // week, month, year
}

type VoiceParseRequest struct {
	Text string `json:"text" binding:"required"`
}

type VoiceParseResponse struct {
	Amount     float64 `json:"amount"`
	Category   string  `json:"category"`
	Type       string  `json:"type"` // income, expense
	Remark     string  `json:"remark"`
	Confidence float64 `json:"confidence"`
}

// Handlers

func (h *ExpenseHandler) Create(c *gin.Context) {
	var req CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	if len(req.Password) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码至少需要4个字符", "code": 400})
		return
	}

	// Check if password already in use
	pwIndex := expensePasswordIndex(req.Password)
	existing, _ := h.db.GetExpenseProfileByPasswordIndex(pwIndex)

	if existing != nil {
		if existing.ExpiresAt != nil && time.Now().After(*existing.ExpiresAt) {
			h.db.DeleteExpenseProfile(existing.ID)
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": "该密码已被使用，请直接登录或使用其他密码", "code": 409})
			return
		}
	}

	ip := c.ClientIP()

	// IP rate limiting
	count, err := h.db.CountExpenseProfilesByIP(ip, time.Now().Add(-h.ipWindow))
	if err == nil && count >= h.maxPerIP {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "创建过于频繁，请稍后再试", "code": 429})
		return
	}

	creatorKey := utils.GenerateHexKey(16)
	hashedCreatorKey, _ := utils.HashPassword(creatorKey)
	hashedPassword, _ := utils.HashPassword(req.Password)

	exp := time.Now().Add(time.Duration(h.defaultExpDays) * 24 * time.Hour)

	profile := &models.ExpenseProfile{
		Password:      hashedPassword,
		PasswordIndex: pwIndex,
		CreatorKey:    hashedCreatorKey,
		ExpiresAt:     &exp,
		CreatorIP:     ip,
	}

	if err := h.db.CreateExpenseProfile(profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	// Create default accounts
	defaultAccounts := []struct {
		Name  string
		Type  string
		Color string
	}{
		{"现金", "cash", "#67C23A"},
		{"银行卡", "bank", "#409EFF"},
		{"支付宝", "alipay", "#E6A23C"},
		{"微信", "wechat", "#07C160"},
	}

	for i, acc := range defaultAccounts {
		account := &models.ExpenseAccount{
			ProfileID: profile.ID,
			Name:      acc.Name,
			Type:      acc.Type,
			Color:     acc.Color,
			Sort:      i,
		}
		h.db.CreateExpenseAccount(account)
	}

	// Create default categories
	defaultCategories := []struct {
		Name  string
		Type  string
		Color string
	}{
		// Expense categories
		{"餐饮", "expense", "#F56C6C"},
		{"交通", "expense", "#409EFF"},
		{"购物", "expense", "#E6A23C"},
		{"居住", "expense", "#909399"},
		{"医疗", "expense", "#F56C6C"},
		{"教育", "expense", "#909399"},
		{"娱乐", "expense", "#E6A23C"},
		{"其他支出", "expense", "#C0C4CC"},
		// Income categories
		{"工资", "income", "#67C23A"},
		{"奖金", "income", "#67C23A"},
		{"投资收益", "income", "#67C23A"},
		{"其他收入", "income", "#67C23A"},
	}

	for i, cat := range defaultCategories {
		category := &models.ExpenseCategory{
			ProfileID: profile.ID,
			Name:      cat.Name,
			Type:      cat.Type,
			Color:     cat.Color,
			Sort:      i,
		}
		h.db.CreateExpenseCategory(category)
	}

	c.JSON(http.StatusCreated, CreateExpenseResponse{
		ID:         profile.ID,
		CreatorKey: creatorKey,
		ExpiresAt:  profile.ExpiresAt,
	})
}

func (h *ExpenseHandler) Login(c *gin.Context) {
	var req LoginExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入密码", "code": 400})
		return
	}

	pwIndex := expensePasswordIndex(req.Password)
	profile, err := h.db.GetExpenseProfileByPasswordIndex(pwIndex)

	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到匹配的档案", "code": 404})
		return
	}

	// Check expiration
	if profile.ExpiresAt != nil && time.Now().After(*profile.ExpiresAt) {
		h.db.DeleteExpenseProfile(profile.ID)
		c.JSON(http.StatusGone, gin.H{"error": "该档案已过期", "code": 410})
		return
	}

	c.JSON(http.StatusOK, ExpenseProfileResponse{
		ID:        profile.ID,
		ExpiresAt: profile.ExpiresAt,
		CreatedAt: profile.CreatedAt,
	})
}

func (h *ExpenseHandler) Get(c *gin.Context) {
	id := c.Param("id")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if profile.ExpiresAt != nil && time.Now().After(*profile.ExpiresAt) {
		h.db.DeleteExpenseProfile(profile.ID)
		c.JSON(http.StatusGone, gin.H{"error": "该档案已过期", "code": 410})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	c.JSON(http.StatusOK, ExpenseProfileResponse{
		ID:        profile.ID,
		ExpiresAt: profile.ExpiresAt,
		CreatedAt: profile.CreatedAt,
	})
}

func (h *ExpenseHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	creatorKey := c.Query("creator_key")

	profile, err := h.db.GetExpenseProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if creatorKey == "" || !utils.VerifyPassword(creatorKey, profile.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	h.db.DeleteExpenseProfile(profile.ID)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ExpenseHandler) Extend(c *gin.Context) {
	id := c.Param("id")
	creatorKey := c.Query("creator_key")
	days := c.DefaultQuery("days", "365")

	profile, err := h.db.GetExpenseProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if creatorKey == "" || !utils.VerifyPassword(creatorKey, profile.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	daysInt := 365
	fmt.Sscanf(days, "%d", &daysInt)
	if daysInt > 730 {
		daysInt = 730
	}

	newExpires := time.Now().Add(time.Duration(daysInt) * 24 * time.Hour)
	if err := h.db.ExtendExpenseProfile(id, &newExpires); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "延期失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "expires_at": newExpires})
}

// Account handlers

func (h *ExpenseHandler) GetAccounts(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	accounts, err := h.db.GetExpenseAccounts(profileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, accounts)
}

func (h *ExpenseHandler) CreateAccount(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	accountType := req.Type
	if accountType == "" {
		accountType = "cash"
	}

	account := &models.ExpenseAccount{
		ProfileID: profileID,
		Name:      req.Name,
		Type:      accountType,
		Balance:   req.Balance,
		Color:     req.Color,
		Icon:      req.Icon,
		Sort:      req.Sort,
	}

	if account.Color == "" {
		account.Color = "#409EFF"
	}
	if account.Icon == "" {
		account.Icon = "Wallet"
	}

	if err := h.db.CreateExpenseAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	c.JSON(http.StatusCreated, account)
}

func (h *ExpenseHandler) UpdateAccount(c *gin.Context) {
	profileID := c.Param("id")
	accountID := c.Param("accountId")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	account, err := h.db.GetExpenseAccount(accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该账户", "code": 404})
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	if req.Name != "" {
		account.Name = req.Name
	}
	if req.Type != "" {
		account.Type = req.Type
	}
	if req.Balance != 0 {
		account.Balance = req.Balance
	}
	if req.Color != "" {
		account.Color = req.Color
	}
	if req.Icon != "" {
		account.Icon = req.Icon
	}
	if req.Sort != 0 {
		account.Sort = req.Sort
	}

	if err := h.db.UpdateExpenseAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, account)
}

func (h *ExpenseHandler) DeleteAccount(c *gin.Context) {
	profileID := c.Param("id")
	accountID := c.Param("accountId")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	if err := h.db.DeleteExpenseAccount(accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Category handlers

func (h *ExpenseHandler) GetCategories(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	categories, err := h.db.GetExpenseCategories(profileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, categories)
}

func (h *ExpenseHandler) CreateCategory(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	category := &models.ExpenseCategory{
		ProfileID: profileID,
		Name:      req.Name,
		Type:      req.Type,
		Icon:      req.Icon,
		Color:     req.Color,
		Sort:      req.Sort,
	}

	if category.Icon == "" {
		category.Icon = "Folder"
	}
	if category.Color == "" {
		category.Color = "#909399"
	}

	if err := h.db.CreateExpenseCategory(category); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	c.JSON(http.StatusCreated, category)
}

func (h *ExpenseHandler) UpdateCategory(c *gin.Context) {
	profileID := c.Param("id")
	categoryID := c.Param("categoryId")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	category, err := h.db.GetExpenseCategory(categoryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分类", "code": 404})
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	if req.Name != "" {
		category.Name = req.Name
	}
	if req.Type != "" {
		category.Type = req.Type
	}
	if req.Icon != "" {
		category.Icon = req.Icon
	}
	if req.Color != "" {
		category.Color = req.Color
	}
	if req.Sort != 0 {
		category.Sort = req.Sort
	}

	if err := h.db.UpdateExpenseCategory(category); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, category)
}

func (h *ExpenseHandler) DeleteCategory(c *gin.Context) {
	profileID := c.Param("id")
	categoryID := c.Param("categoryId")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	if err := h.db.DeleteExpenseCategory(categoryID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Transaction handlers

func (h *ExpenseHandler) GetTransactions(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	txs, err := h.db.GetExpenseTransactions(profileID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败", "code": 500})
		return
	}

	// Enrich with account and category names
	type EnrichTx struct {
		*models.ExpenseTransaction
		AccountName  string `json:"account_name"`
		CategoryName string `json:"category_name"`
		CategoryColor string `json:"category_color"`
	}

	result := make([]EnrichTx, 0, len(txs))
	for _, t := range txs {
		acc, _ := h.db.GetExpenseAccount(t.AccountID)
		cat, _ := h.db.GetExpenseCategory(t.CategoryID)

		et := EnrichTx{
			ExpenseTransaction: t,
			AccountName:        "",
			CategoryName:       "",
			CategoryColor:      "",
		}
		if acc != nil {
			et.AccountName = acc.Name
		}
		if cat != nil {
			et.CategoryName = cat.Name
			et.CategoryColor = cat.Color
		}
		result = append(result, et)
	}

	c.JSON(http.StatusOK, result)
}

func (h *ExpenseHandler) CreateTransaction(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	var req CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	tx := &models.ExpenseTransaction{
		ProfileID:  profileID,
		AccountID:  req.AccountID,
		CategoryID: req.CategoryID,
		Amount:     req.Amount,
		Type:       req.Type,
		Date:       req.Date,
		Remark:     req.Remark,
		Tags:       req.Tags,
	}

	if tx.Date == "" {
		tx.Date = time.Now().Format("2006-01-02")
	}

	if err := h.db.CreateExpenseTransaction(tx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	// Update account balance
	account, err := h.db.GetExpenseAccount(req.AccountID)
	if err == nil {
		var newBalance float64
		if req.Type == "expense" {
			newBalance = account.Balance - req.Amount
		} else {
			newBalance = account.Balance + req.Amount
		}
		h.db.UpdateExpenseAccountBalance(req.AccountID, newBalance)
	}

	c.JSON(http.StatusCreated, tx)
}

func (h *ExpenseHandler) UpdateTransaction(c *gin.Context) {
	profileID := c.Param("id")
	txID := c.Param("txId")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	oldTx, err := h.db.GetExpenseTransaction(txID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该记录", "code": 404})
		return
	}

	var req UpdateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	tx := &models.ExpenseTransaction{
		ID:         txID,
		ProfileID:  profileID,
		AccountID:  oldTx.AccountID,
		CategoryID: oldTx.CategoryID,
		Amount:     oldTx.Amount,
		Type:       oldTx.Type,
		Date:       oldTx.Date,
		Remark:     oldTx.Remark,
		Tags:       oldTx.Tags,
	}

	// Apply updates
	if req.AccountID != "" {
		tx.AccountID = req.AccountID
	}
	if req.CategoryID != "" {
		tx.CategoryID = req.CategoryID
	}
	if req.Amount != 0 {
		tx.Amount = req.Amount
	}
	if req.Type != "" {
		tx.Type = req.Type
	}
	if req.Date != "" {
		tx.Date = req.Date
	}
	tx.Remark = req.Remark
	if req.Tags != "" {
		tx.Tags = req.Tags
	}

	// Reverse old balance change
	oldAccount, _ := h.db.GetExpenseAccount(oldTx.AccountID)
	if oldAccount != nil {
		var oldBalance float64
		if oldTx.Type == "expense" {
			oldBalance = oldAccount.Balance + oldTx.Amount
		} else {
			oldBalance = oldAccount.Balance - oldTx.Amount
		}
		h.db.UpdateExpenseAccountBalance(oldTx.AccountID, oldBalance)
	}

	if err := h.db.UpdateExpenseTransaction(tx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}

	// Apply new balance change
	newAccount, _ := h.db.GetExpenseAccount(tx.AccountID)
	if newAccount != nil {
		var newBalance float64
		if tx.Type == "expense" {
			newBalance = newAccount.Balance - tx.Amount
		} else {
			newBalance = newAccount.Balance + tx.Amount
		}
		h.db.UpdateExpenseAccountBalance(tx.AccountID, newBalance)
	}

	c.JSON(http.StatusOK, tx)
}

func (h *ExpenseHandler) DeleteTransaction(c *gin.Context) {
	profileID := c.Param("id")
	txID := c.Param("txId")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	tx, err := h.db.GetExpenseTransaction(txID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该记录", "code": 404})
		return
	}

	// Reverse balance change
	account, _ := h.db.GetExpenseAccount(tx.AccountID)
	if account != nil {
		var newBalance float64
		if tx.Type == "expense" {
			newBalance = account.Balance + tx.Amount
		} else {
			newBalance = account.Balance - tx.Amount
		}
		h.db.UpdateExpenseAccountBalance(tx.AccountID, newBalance)
	}

	h.db.DeleteExpenseTransaction(txID)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Stats

func (h *ExpenseHandler) GetStats(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)
	period := c.DefaultQuery("period", "month")

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	stats, err := h.db.GetExpenseStats(profileID, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// AI Analyze

func (h *ExpenseHandler) Analyze(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	var req AnalyzeRequest
	c.ShouldBindJSON(&req)

	period := req.Period
	if period == "" {
		period = "month"
	}

	// Get stats data
	stats, err := h.db.GetExpenseStats(profileID, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取数据失败", "code": 500})
		return
	}

	// Calculate date range for transactions
	startDate, endDate := calculateDateRange(period)
	// Get transactions for the period
	txs, err := h.db.GetExpenseTransactions(profileID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取记录失败", "code": 500})
		return
	}

	// Check if DeepSeek is configured
	if h.cfg.DeepSeek.APIKey == "" {
		// Return basic analysis without AI
		c.JSON(http.StatusOK, gin.H{
			"analysis": getBasicAnalysis(stats, period),
			"ai_enabled": false,
		})
		return
	}

	// Check if there's any data to analyze
	if stats.TransactionCount == 0 {
		c.JSON(http.StatusOK, gin.H{
			"analysis": "暂无消费记录，无法进行分析。建议先添加一些记账记录后再使用AI分析功能。",
			"ai_enabled": true,
		})
		return
	}

	// Prepare prompt
	prompt := prepareAnalysisPrompt(stats, txs, period)

	// Call DeepSeek API
	result, err := h.callDeepSeekAPI(prompt)
	if err != nil {
		// API调用失败时返回基础分析
		c.JSON(http.StatusOK, gin.H{
			"analysis": getBasicAnalysis(stats, period) + "\n\n⚠️ AI分析暂时不可用: " + err.Error(),
			"ai_enabled": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": result,
		"ai_enabled": true,
	})
}

func (h *ExpenseHandler) callDeepSeekAPI(prompt string) (string, error) {
	apiKey := h.cfg.DeepSeek.APIKey
	model := h.cfg.DeepSeek.Model
	if model == "" {
		model = "deepseek-chat"
	}

	url := "https://api.deepseek.com/chat/completions"
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个专业的个人财务顾问，擅长分析消费习惯并给出理财建议。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API返回错误(%d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析JSON失败: %v, 响应: %s", err, string(respBody))
	}

	// 安全地获取choices
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("API响应格式错误: 无choices, 响应: %s", string(respBody))
	}

	firstChoice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("API响应格式错误: choices[0]格式错误")
	}

	message, ok := firstChoice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("API响应格式错误: 无message")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("API响应格式错误: 无content")
	}

	return content, nil
}

func prepareAnalysisPrompt(stats *models.ExpenseStats, txs []*models.ExpenseTransaction, period string) string {
	// Get recent transactions for context
	var recentTxStrings []string
	for i, tx := range txs {
		if i >= 20 {
			break
		}
		recentTxStrings = append(recentTxStrings, fmt.Sprintf("%s %s %.2f %s", tx.Date, tx.Type, tx.Amount, tx.Remark))
	}

	prompt := fmt.Sprintf(`请分析以下消费数据，给出专业的财务建议：

统计周期: %s
总收入: %.2f
总支出: %.2f
结余: %.2f
交易笔数: %d

支出分类统计:
%s

月度支出趋势:
%s

请从以下几个方面进行分析：
1. 消费结构分析 - 主要支出在哪些方面
2. 异常消费提醒 - 是否有不合理的支出
3. 理财建议 - 如何优化支出结构
4. 预算建议 - 建议每月支出控制在多少

请用中文回答，语言要自然流畅。`, period, stats.TotalIncome, stats.TotalExpense, stats.Balance, stats.TransactionCount,
		formatMap(stats.ByCategory),
		formatMap(stats.ByMonth))

	return prompt
}

func formatMap(m map[string]float64) string {
	var parts []string
	for k, v := range m {
		parts = append(parts, fmt.Sprintf("%s: %.2f", k, v))
	}
	return strings.Join(parts, ", ")
}

// calculateDateRange calculates start and end dates based on period
func calculateDateRange(period string) (string, string) {
	now := time.Now()
	endDate := now.Format("2006-01-02")

	// Get start of current week (Monday)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -(weekday - 1))

	var startDate string
	switch period {
	case "week":
		startDate = startOfWeek.Format("2006-01-02")
	case "month":
		startDate = fmt.Sprintf("%d-%02d-01", now.Year(), now.Month())
	case "year":
		startDate = fmt.Sprintf("%d-01-01", now.Year())
	default:
		startDate = fmt.Sprintf("%d-%02d-01", now.Year(), now.Month())
	}

	return startDate, endDate
}

func getBasicAnalysis(stats *models.ExpenseStats, period string) string {
	periodCN := map[string]string{"week": "本周", "month": "本月", "year": "今年"}[period]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【%s财务概要】\n\n", periodCN))
	sb.WriteString(fmt.Sprintf("收入: %.2f 元\n", stats.TotalIncome))
	sb.WriteString(fmt.Sprintf("支出: %.2f 元\n", stats.TotalExpense))
	sb.WriteString(fmt.Sprintf("结余: %.2f 元\n\n", stats.Balance))

	if len(stats.ByCategory) > 0 {
		sb.WriteString("【支出分类】\n")
		for k, v := range stats.ByCategory {
			sb.WriteString(fmt.Sprintf("• %s: %.2f 元\n", k, v))
		}
	}

	sb.WriteString(fmt.Sprintf("\n共 %d 笔交易\n", stats.TransactionCount))

	if stats.Balance < 0 {
		sb.WriteString("\n⚠️ 提示: 本期支出超过收入，请注意控制开支。")
	} else if stats.Balance > stats.TotalIncome*0.5 {
		sb.WriteString("\n✓ 储蓄率良好，建议将部分结余用于投资理财。")
	}

	return sb.String()
}

// VoiceParse 智能语音解析
func (h *ExpenseHandler) VoiceParse(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)

	profile, err := h.db.GetExpenseProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	var req VoiceParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供语音文本", "code": 400})
		return
	}

	// Get categories for reference
	categories, _ := h.db.GetExpenseCategories(profileID)

	// Check if DeepSeek is configured
	if h.cfg.DeepSeek.APIKey == "" {
		// Fallback to basic parsing
		result := basicVoiceParse(req.Text, categories)
		c.JSON(http.StatusOK, result)
		return
	}

	// Use DeepSeek for intelligent parsing
	result, err := h.callDeepSeekVoiceParse(req.Text, categories)
	if err != nil {
		// Fallback to basic parsing on error
		result = basicVoiceParse(req.Text, categories)
		result.Confidence = 0.3
		c.JSON(http.StatusOK, result)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ExpenseHandler) callDeepSeekVoiceParse(text string, categories []*models.ExpenseCategory) (*VoiceParseResponse, error) {
	// Build category reference
	var catList []string
	for _, cat := range categories {
		catList = append(catList, fmt.Sprintf("%s(%s)", cat.Name, cat.Type))
	}
	categoriesStr := strings.Join(catList, ", ")

	prompt := fmt.Sprintf(`请分析以下记账语音输入，提取出金额、分类和类型（收入/支出）。

可用的分类: %s

语音内容: %s

请以 JSON 格式返回分析结果，格式如下:
{"amount": 金额数字, "category": "匹配的分类名称", "type": "income 或 expense", "remark": "简要备注", "confidence": 0.0-1.0 的置信度}

注意事项:
1. amount 如果无法确定则返回 0
2. category 必须在可用分类中选择，如果无法匹配返回空字符串
3. type: 支出相关(如花钱、消费、买了、付了等)返回 "expense"，收入相关(如工资、收款、收到、赚钱等)返回 "income"
4. remark 提取关键信息作为备注（如"早餐"、"打车"等，不超过20字）
5. confidence 根据你对解析结果的自信程度给出一个 0.0-1.0 的分数

只返回 JSON，不要其他内容。`, categoriesStr, text)

	apiKey := h.cfg.DeepSeek.APIKey
	model := h.cfg.DeepSeek.Model
	if model == "" {
		model = "deepseek-chat"
	}

	url := "https://api.deepseek.com/chat/completions"
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "助手，擅长从你是一个专业的记账语音输入中提取记账信息。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	choices := result["choices"].([]interface{})
	firstChoice := choices[0].(map[string]interface{})
	message := firstChoice["message"].(map[string]interface{})
	content := message["content"].(string)

	// Parse JSON from response
	content = strings.TrimSpace(content)
	// Handle potential markdown code blocks
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var parseResult VoiceParseResponse
	if err := json.Unmarshal([]byte(content), &parseResult); err != nil {
		return nil, err
	}

	return &parseResult, nil
}

// 中文数字转阿拉伯数字
func parseChineseNumber(text string) float64 {
	// 数字映射
	digitMap := map[rune]int{
		'零': 0, '一': 1, '二': 2, '三': 3, '四': 4,
		'五': 5, '六': 6, '七': 7, '八': 8, '九': 9,
	}

	// 单位权重
	unitMap := map[rune]int{
		'十': 10,
		'百': 100,
		'千': 1000,
		'万': 10000,
	}

	// 特殊处理纯 "十" -> 10
	if text == "十" {
		return 10
	}

	var total int
	var current int

	for _, ch := range text {
		if d, ok := digitMap[ch]; ok {
			current = current*10 + d
		} else if u, ok := unitMap[ch]; ok {
			if ch == '十' {
				// "十" 特殊处理：如果前面没有数字，如"十五"，则十=10；如果前面有数字，如"二十五"，则十=10*前面的数字
				if current == 0 {
					current = 1
				}
				total += current * u
				current = 0
			} else if ch == '万' {
				// 万：前面的总和乘以10000
				if current > 0 {
					total += current
					current = 0
				}
				total = total * u
			} else {
				// 百、千
				if current == 0 {
					current = 1
				}
				total += current * u
				current = 0
			}
		}
	}

	// 加上剩余的个位数
	total += current

	if total > 0 {
		return float64(total)
	}

	return 0
}

// 解析中文金额表达
func parseChineseAmount(text string) float64 {
	// 优先处理 "花了350块/元" "付了500" 这种模式（阿拉伯数字）
	// 用贪婪匹配确保匹配完整数字
	re := regexp.MustCompile(`(?:花|付)[了]?\s*(\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		amount, _ := strconv.ParseFloat(matches[1], 64)
		if amount > 0 {
			return amount
		}
	}

	// 处理 "350块" "350元" 这种模式
	re = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:块|元|rmb|RMB|人民币|块钱)`)
	matches = re.FindStringSubmatch(text)
	if len(matches) > 1 {
		amount, _ := strconv.ParseFloat(matches[1], 64)
		if amount > 0 {
			return amount
		}
	}

	// 处理中文数字 + 单位 "三十五块" "二十五元"
	re = regexp.MustCompile(`([零一二三四五六七八九十百]+)(?:块|元|rmb|RMB|人民币|块钱)?`)
	matches = re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return parseChineseNumber(matches[1])
	}

	// 处理 "一百二十五" -> 125
	re = regexp.MustCompile(`([零一二三四五六七八九十百]+)`)
	matches = re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return parseChineseNumber(matches[1])
	}

	// 处理纯数字（最后兜底）
	re = regexp.MustCompile(`(\d+(?:\.\d+)?)`)
	matches = re.FindStringSubmatch(text)
	if len(matches) > 1 {
		amount, _ := strconv.ParseFloat(matches[1], 64)
		return amount
	}

	return 0
}

func basicVoiceParse(text string, categories []*models.ExpenseCategory) *VoiceParseResponse {
	result := &VoiceParseResponse{
		Amount:     0,
		Category:   "",
		Type:       "expense",
		Remark:     text,
		Confidence: 0.5,
	}

	lowerText := strings.ToLower(text)

	// 先尝试解析中文金额
	result.Amount = parseChineseAmount(text)
	if result.Amount == 0 {
		// 回退到纯数字解析
		amountPatterns := []string{
			`(\d+(?:\.\d{1,2})?)\s*元`,
			`(\d+(?:\.\d{1,2})?)\s*块`,
			`花[了]?\s*(\d+(?:\.\d{1,2})?)`,
			`付[了]?\s*(\d+(?:\.\d{1,2})?)`,
			`(\d+)`,
		}

		for _, pattern := range amountPatterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(text)
			if len(matches) > 1 {
				result.Amount, _ = strconv.ParseFloat(matches[1], 64)
				break
			}
		}
	}

	// Parse category - 增强关键词匹配
	categoryKeywords := map[string][]string{
		// 支出分类
		"餐饮": {"吃饭", "餐饮", "午餐", "晚餐", "早餐", "外卖", "奶茶", "咖啡", "水果", "零食", "面", "饺子", "包子", "快餐", "火锅", "烧烤", "炸鸡", "披萨", "麻辣烫", "沙拉", "便利店", "超市", "买菜", "零食", "餐馆", "餐厅", "小卖部", "商店", "买水", "喝奶茶", "喝咖啡", "买水果", "零食"},
		"交通": {"打车", "地铁", "公交", "巴士", "加油", "停车", "出租车", "滴滴", "打车", "油费", "过路费", "高速", "网约车", "共享单车", "骑车", "坐车", "乘车", "打车费", "公交车", "轻轨", "高铁", "火车", "飞机", "票"},
		"购物": {"购物", "买", "淘宝", "京东", "快递", "网购", "拼多多", "天猫", "唯品会", "衣服", "鞋子", "包包", "背包", "裙子", "裤子", "帽子", "手套", "围巾", "化妆品", "护肤品", "口红", "面膜", "洗面奶", "洗衣液", "洗洁精", "纸巾", "毛巾", "牙刷", "牙膏", "沐浴露", "洗发水", "护发素"},
		"居住": {"房租", "水电费", "物业费", "住房", "房租", "燃气费", "宽带费", "话费", "日用品", "家具", "床", "桌子", "椅子", "柜子", "被子", "枕头", "蚊帐", "电费", "水费", "燃气", "房租", "租金"},
		"医疗": {"医疗", "药", "医院", "看病", "药店", "体检", "医保", "挂号", "门诊", "住院", "手术", "检查", "化验", "CT", "B超", "买药", "诊所", "疫苗", "核酸"},
		"教育": {"教育", "培训", "学费", "书", "课程", "考试", "文具", "培训费", "辅导班", "家教", "补习", "资料", "教材", "笔记本", "笔", "橡皮", "尺子", "书包", "报名费"},
		"娱乐": {"电影", "游戏", "旅游", "KTV", "唱歌", "上网", "视频会员", "会员", "爱奇艺", "腾讯视频", "优酷", "芒果TV", "B站", "羽毛球", "健身", "健身房", "游泳", "话剧", "演出", "演唱会", "音乐会", "展览", "博物馆", "动物园", "游乐园", "门票", "充值", "点卡", "皮肤", "英雄", "会员", "VIP", "外卖会员"},
		"其他支出": {"其他", "杂费", "花费", "支出"},
		// 收入分类
		"工资": {"工资", "发工资", "薪资", "月薪", "底薪", "月工资", "工资到账", "发薪", "薪酬"},
		"奖金": {"奖金", "年终奖", "分红", "提成", "佣金", "外快", "绩效", "奖励", "补贴"},
		"投资收益": {"理财", "利息", "投资收益", "分红", "股票", "基金", "利息", "赚钱", "投资"},
		"其他收入": {"收入", "收款", "收到", "到账", "转账", "红包"},
	}

	for cat, keywords := range categoryKeywords {
		for _, keyword := range keywords {
			if strings.Contains(lowerText, keyword) {
				result.Category = cat
				break
			}
		}
		if result.Category != "" {
			break
		}
	}

	// Determine type
	incomeKeywords := []string{"工资", "奖金", "收入", "赚钱", "收款", "收到", "发工资", "发钱"}
	for _, keyword := range incomeKeywords {
		if strings.Contains(lowerText, keyword) {
			result.Type = "income"
			break
		}
	}

	return result
}

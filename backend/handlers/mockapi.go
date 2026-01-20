package handlers

import (
	"devtools/models"
	"devtools/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type MockAPIHandler struct {
	db *models.DB
}

func NewMockAPIHandler(db *models.DB) *MockAPIHandler {
	return &MockAPIHandler{db: db}
}

type CreateMockAPIRequest struct {
	Name            string            `json:"name"`
	Method          string            `json:"method" binding:"required"`
	ResponseBody    string            `json:"response_body"`
	ResponseStatus  int               `json:"response_status"`
	ResponseHeaders map[string]string `json:"response_headers"`
	ResponseDelay   int               `json:"response_delay"`
	ExpiresIn       int               `json:"expires_in"` // hours, default 24
	MaxCalls        int               `json:"max_calls"`  // default 1000
	CustomPath      string            `json:"custom_path"`
	Password        string            `json:"password"`
}

type CreateMockAPIResponse struct {
	ID        string    `json:"id"`
	MockURL   string    `json:"mock_url"`
	ExpiresAt time.Time `json:"expires_at"`
	MaxCalls  int       `json:"max_calls"`
}

type GetMockAPIResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Method          string    `json:"method"`
	ResponseBody    string    `json:"response_body"`
	ResponseStatus  int       `json:"response_status"`
	ResponseHeaders string    `json:"response_headers"`
	ResponseDelay   int       `json:"response_delay"`
	ExpiresAt       time.Time `json:"expires_at"`
	MaxCalls        int       `json:"max_calls"`
	CallCount       int       `json:"call_count"`
	CreatedAt       time.Time `json:"created_at"`
	HasPassword     bool      `json:"has_password"`
}

type UpdateMockAPIRequest struct {
	ResponseBody    *string            `json:"response_body,omitempty"`
	ResponseStatus  *int               `json:"response_status,omitempty"`
	ResponseHeaders *map[string]string `json:"response_headers,omitempty"`
	ResponseDelay   *int               `json:"response_delay,omitempty"`
	Password        string             `json:"password"`
}

// Create handles POST /api/mockapi
func (h *MockAPIHandler) Create(c *gin.Context) {
	var req CreateMockAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true, "OPTIONS": true,
	}
	req.Method = strings.ToUpper(req.Method)
	if !validMethods[req.Method] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid HTTP method"})
		return
	}

	// Get client IP
	clientIP := c.ClientIP()

	// Check rate limit: 10 per hour per IP
	count, err := h.db.CountMockAPIsByIP(clientIP, time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check rate limit"})
		return
	}
	if count >= 10 {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Rate limit exceeded: maximum 10 mock APIs per hour per IP",
		})
		return
	}

	// Set defaults
	if req.ResponseStatus == 0 {
		req.ResponseStatus = 200
	}
	if req.ExpiresIn == 0 {
		req.ExpiresIn = 24 // 24 hours
	}
	if req.MaxCalls == 0 {
		req.MaxCalls = 1000
	}

	// Validate expires_in (1-168 hours / 1-7 days)
	if req.ExpiresIn < 1 || req.ExpiresIn > 168 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expires_in must be between 1 and 168 hours"})
		return
	}

	// Validate response status
	if req.ResponseStatus < 100 || req.ResponseStatus > 599 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "response_status must be between 100 and 599"})
		return
	}

	// Validate response delay
	if req.ResponseDelay < 0 || req.ResponseDelay > 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "response_delay must be between 0 and 30 seconds"})
		return
	}

	// Validate response body size (100KB)
	if len(req.ResponseBody) > 100*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "response_body exceeds 100KB limit"})
		return
	}

	// Serialize headers
	headersJSON, err := models.SerializeHeadersJSON(req.ResponseHeaders)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid response headers format"})
		return
	}

	// Hash password if provided
	hashedPassword := ""
	if req.Password != "" {
		hashedPassword = utils.HashPassword(req.Password)
	}

	// Calculate expiration
	expiresAt := time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)

	// Create mock API
	mockAPI := &models.MockAPI{
		Name:            req.Name,
		Method:          req.Method,
		ResponseBody:    req.ResponseBody,
		ResponseStatus:  req.ResponseStatus,
		ResponseHeaders: headersJSON,
		ResponseDelay:   req.ResponseDelay,
		ExpiresAt:       expiresAt,
		MaxCalls:        req.MaxCalls,
		CreatorIP:       clientIP,
		Password:        hashedPassword,
	}

	// Create with custom ID if provided
	if req.CustomPath != "" {
		err = h.db.CreateMockAPIWithCustomID(mockAPI, req.CustomPath)
	} else {
		err = h.db.CreateMockAPI(mockAPI)
	}

	if err != nil {
		if strings.Contains(err.Error(), "storage limit") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": "Custom path already exists"})
			return
		}
		if strings.Contains(err.Error(), "custom path") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create mock API"})
		return
	}

	// Build mock URL
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	mockURL := fmt.Sprintf("%s://%s/mock/%s", scheme, c.Request.Host, mockAPI.ID)

	c.JSON(http.StatusCreated, CreateMockAPIResponse{
		ID:        mockAPI.ID,
		MockURL:   mockURL,
		ExpiresAt: mockAPI.ExpiresAt,
		MaxCalls:  mockAPI.MaxCalls,
	})
}

// Get handles GET /api/mockapi/:id
func (h *MockAPIHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mock API ID is required"})
		return
	}

	mockAPI, err := h.db.GetMockAPI(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Mock API not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve mock API"})
		return
	}

	c.JSON(http.StatusOK, GetMockAPIResponse{
		ID:              mockAPI.ID,
		Name:            mockAPI.Name,
		Method:          mockAPI.Method,
		ResponseBody:    mockAPI.ResponseBody,
		ResponseStatus:  mockAPI.ResponseStatus,
		ResponseHeaders: mockAPI.ResponseHeaders,
		ResponseDelay:   mockAPI.ResponseDelay,
		ExpiresAt:       mockAPI.ExpiresAt,
		MaxCalls:        mockAPI.MaxCalls,
		CallCount:       mockAPI.CallCount,
		CreatedAt:       mockAPI.CreatedAt,
		HasPassword:     mockAPI.Password != "",
	})
}

// GetLogs handles GET /api/mockapi/:id/logs
func (h *MockAPIHandler) GetLogs(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mock API ID is required"})
		return
	}

	// Get mock API to check password
	mockAPI, err := h.db.GetMockAPI(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Mock API not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve mock API"})
		return
	}

	// Check password if set
	if mockAPI.Password != "" {
		password := c.Query("password")
		if !utils.VerifyPassword(password, mockAPI.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
			return
		}
	}

	// Get logs (last 100)
	logs, err := h.db.GetMockAPILogs(id, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// Update handles PUT /api/mockapi/:id
func (h *MockAPIHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mock API ID is required"})
		return
	}

	var req UpdateMockAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get mock API to check password
	mockAPI, err := h.db.GetMockAPI(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Mock API not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve mock API"})
		return
	}

	// Check password if set
	if mockAPI.Password != "" {
		if !utils.VerifyPassword(req.Password, mockAPI.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
			return
		}
	}

	// Build updates map
	updates := make(map[string]interface{})

	if req.ResponseBody != nil {
		if len(*req.ResponseBody) > 100*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "response_body exceeds 100KB limit"})
			return
		}
		updates["response_body"] = *req.ResponseBody
	}

	if req.ResponseStatus != nil {
		if *req.ResponseStatus < 100 || *req.ResponseStatus > 599 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "response_status must be between 100 and 599"})
			return
		}
		updates["response_status"] = *req.ResponseStatus
	}

	if req.ResponseHeaders != nil {
		headersJSON, err := models.SerializeHeadersJSON(*req.ResponseHeaders)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid response headers format"})
			return
		}
		updates["response_headers"] = headersJSON
	}

	if req.ResponseDelay != nil {
		if *req.ResponseDelay < 0 || *req.ResponseDelay > 30 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "response_delay must be between 0 and 30 seconds"})
			return
		}
		updates["response_delay"] = *req.ResponseDelay
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Update mock API
	err = h.db.UpdateMockAPI(id, updates)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Mock API not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update mock API"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mock API updated successfully"})
}

// Delete handles DELETE /api/mockapi/:id
func (h *MockAPIHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mock API ID is required"})
		return
	}

	password := c.Query("password")

	// Get mock API to check password
	mockAPI, err := h.db.GetMockAPI(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Mock API not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve mock API"})
		return
	}

	// Check password if set
	if mockAPI.Password != "" {
		if !utils.VerifyPassword(password, mockAPI.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
			return
		}
	}

	// Delete mock API
	err = h.db.DeleteMockAPI(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Mock API not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete mock API"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mock API deleted successfully"})
}

// Execute handles ANY /mock/:id
func (h *MockAPIHandler) Execute(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mock API ID is required"})
		return
	}

	// Get mock API
	mockAPI, err := h.db.GetMockAPI(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Mock API not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve mock API"})
		return
	}

	// Check if expired
	if time.Now().After(mockAPI.ExpiresAt) {
		h.db.DeleteMockAPI(id)
		c.JSON(http.StatusGone, gin.H{"error": "Mock API has expired"})
		return
	}

	// Check if max calls reached
	if mockAPI.CallCount >= mockAPI.MaxCalls {
		h.db.DeleteMockAPI(id)
		c.JSON(http.StatusGone, gin.H{"error": "Mock API has reached its maximum call limit"})
		return
	}

	// Log request
	requestBody, _ := io.ReadAll(c.Request.Body)
	if len(requestBody) > 10*1024 {
		requestBody = requestBody[:10*1024]
	}

	// Serialize request headers
	headersMap := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headersMap[key] = values[0]
		}
	}
	headersJSON, _ := json.Marshal(headersMap)

	// Serialize query params
	queryMap := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryMap[key] = values[0]
		}
	}
	queryJSON, _ := json.Marshal(queryMap)

	log := &models.MockAPILog{
		MockAPIID:   id,
		Method:      c.Request.Method,
		Headers:     string(headersJSON),
		QueryParams: string(queryJSON),
		Body:        string(requestBody),
		ClientIP:    c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	}

	// Best-effort logging (don't fail request if logging fails)
	h.db.LogMockAPIRequest(log)

	// Clean old logs if needed (keep last 100)
	h.db.CleanOldLogs(id)

	// Apply delay
	if mockAPI.ResponseDelay > 0 {
		time.Sleep(time.Duration(mockAPI.ResponseDelay) * time.Second)
	}

	// Increment call count (best-effort)
	h.db.IncrementCallCount(id)

	// Ensure CORS headers are set first (before user headers)
	if c.GetHeader("Access-Control-Allow-Origin") == "" {
		c.Header("Access-Control-Allow-Origin", "*")
	}
	if c.GetHeader("Access-Control-Allow-Methods") == "" {
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
	}
	if c.GetHeader("Access-Control-Allow-Headers") == "" {
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
	}

	// Parse and set response headers (may override CORS headers if user specified)
	headers, err := models.ParseHeadersJSON(mockAPI.ResponseHeaders)
	if err == nil {
		for key, value := range headers {
			c.Header(key, value)
		}
	}

	// Determine Content-Type with UTF-8 charset for Chinese support
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		// Default to text/plain with UTF-8 if not set
		contentType = "text/plain; charset=utf-8"
	} else if !strings.Contains(strings.ToLower(contentType), "charset") {
		// Add charset=utf-8 if not present
		contentType += "; charset=utf-8"
	}

	// Send response
	c.Data(mockAPI.ResponseStatus, contentType, []byte(mockAPI.ResponseBody))
}

package handlers

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"devtools/config"

	"github.com/gin-gonic/gin"
)

type NPSHandler struct {
	cfg        config.NPSConfig
	tunnelPort string // proxy.tunnel_port，用于前端展示一键映射
	clientID   int
	mu         sync.Mutex
}

func NewNPSHandler(cfg config.NPSConfig, tunnelPort string) *NPSHandler {
	return &NPSHandler{cfg: cfg, tunnelPort: tunnelPort}
}

func (h *NPSHandler) checkAdmin(password string) bool {
	if h.cfg.AdminPassword == "" {
		return false
	}
	return password == h.cfg.AdminPassword
}

// npsAuthParams 生成 NPS API 认证参数
func (h *NPSHandler) npsAuthParams() url.Values {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sum := md5.Sum([]byte(h.cfg.AuthKey + ts))
	authKey := fmt.Sprintf("%x", sum)
	v := url.Values{}
	v.Set("auth_key", authKey)
	v.Set("timestamp", ts)
	return v
}

// npsPost 调用 NPS API
func (h *NPSHandler) npsPost(path string, params url.Values) (map[string]interface{}, error) {
	auth := h.npsAuthParams()
	for k, vs := range auth {
		params[k] = vs
	}
	resp, err := http.PostForm(h.cfg.ServerURL+path, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf("NPS 响应非 JSON（可能 auth_key 错误或 URL 不对）: %s", preview)
	}
	return result, nil
}

// getClientID 懒加载：通过 vkey 找到 client_id
func (h *NPSHandler) getClientID() (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clientID != 0 {
		return h.clientID, nil
	}
	params := url.Values{}
	params.Set("search", h.cfg.VKey)
	params.Set("offset", "0")
	params.Set("limit", "50")
	result, err := h.npsPost("/client/list/", params)
	if err != nil {
		return 0, err
	}
	rows, ok := result["rows"].([]interface{})
	if !ok {
		return 0, fmt.Errorf("未找到客户端列表")
	}
	for _, row := range rows {
		m, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		vkey, _ := m["VerifyKey"].(string)
		if vkey == h.cfg.VKey {
			id, err := toInt(m["Id"])
			if err != nil {
				continue
			}
			h.clientID = id
			return id, nil
		}
	}
	return 0, fmt.Errorf("未找到 vkey=%s 对应的客户端", h.cfg.VKey)
}

// findFreePort 从配置区间找第一个未被占用的端口
func (h *NPSHandler) findFreePort(usedPorts map[int]bool) (int, error) {
	if h.cfg.PortRangeStart <= 0 || h.cfg.PortRangeEnd <= h.cfg.PortRangeStart {
		return 0, fmt.Errorf("未配置自动端口区间（port_range_start/port_range_end）")
	}
	for p := h.cfg.PortRangeStart; p <= h.cfg.PortRangeEnd; p++ {
		if !usedPorts[p] {
			return p, nil
		}
	}
	return 0, fmt.Errorf("端口区间 %d-%d 已全部占用", h.cfg.PortRangeStart, h.cfg.PortRangeEnd)
}

func toInt(v interface{}) (int, error) {
	switch x := v.(type) {
	case float64:
		return int(x), nil
	case int:
		return x, nil
	case string:
		return strconv.Atoi(x)
	}
	return 0, fmt.Errorf("无法转换为 int: %v", v)
}

// Status GET /api/nps/status?admin_password=xxx
func (h *NPSHandler) Status(c *gin.Context) {
	if !h.checkAdmin(c.Query("admin_password")) {
		c.JSON(403, gin.H{"error": "密码错误"})
		return
	}
	if h.cfg.ServerURL == "" || h.cfg.VKey == "" {
		c.JSON(400, gin.H{"error": "NPS 未配置"})
		return
	}
	clientID, err := h.getClientID()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	params := url.Values{}
	params.Set("id", strconv.Itoa(clientID))
	result, err := h.npsPost("/client/getclient/", params)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"client":           result,
		"client_id":        clientID,
		"vkey":             h.cfg.VKey,
		"port_range_start": h.cfg.PortRangeStart,
		"port_range_end":   h.cfg.PortRangeEnd,
		"tunnel_port":      h.tunnelPort,
	})
}

// ListTunnels GET /api/nps/tunnels?admin_password=xxx
func (h *NPSHandler) ListTunnels(c *gin.Context) {
	if !h.checkAdmin(c.Query("admin_password")) {
		c.JSON(403, gin.H{"error": "密码错误"})
		return
	}
	clientID, err := h.getClientID()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	params := url.Values{}
	params.Set("client_id", strconv.Itoa(clientID))
	params.Set("offset", "0")
	params.Set("limit", "200")
	result, err := h.npsPost("/index/gettunnel/", params)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, result)
}

// AddTunnel POST /api/nps/tunnels
// body: { admin_password, type, port(可选), target, remark }
func (h *NPSHandler) AddTunnel(c *gin.Context) {
	var req struct {
		AdminPassword string `json:"admin_password"`
		Type          string `json:"type"`
		Port          int    `json:"port"`
		Target        string `json:"target"`
		Remark        string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	if !h.checkAdmin(req.AdminPassword) {
		c.JSON(403, gin.H{"error": "密码错误"})
		return
	}
	if req.Target == "" {
		c.JSON(400, gin.H{"error": "target 不能为空"})
		return
	}
	if req.Type == "" {
		req.Type = "tcp"
	}

	clientID, err := h.getClientID()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	port := req.Port
	if port == 0 {
		// 自动分配端口：先获取已用端口
		listParams := url.Values{}
		listParams.Set("client_id", strconv.Itoa(clientID))
		listParams.Set("offset", "0")
		listParams.Set("limit", "500")
		listResult, err := h.npsPost("/index/gettunnel/", listParams)
		if err != nil {
			c.JSON(500, gin.H{"error": "获取已用端口失败: " + err.Error()})
			return
		}
		usedPorts := map[int]bool{}
		if rows, ok := listResult["rows"].([]interface{}); ok {
			for _, row := range rows {
				if m, ok := row.(map[string]interface{}); ok {
					if p, err := toInt(m["Port"]); err == nil && p > 0 {
						usedPorts[p] = true
					}
				}
			}
		}
		port, err = h.findFreePort(usedPorts)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
	}

	params := url.Values{}
	params.Set("type", req.Type)
	params.Set("port", strconv.Itoa(port))
	params.Set("target", req.Target)
	params.Set("client_id", strconv.Itoa(clientID))
	params.Set("remark", req.Remark)
	result, err := h.npsPost("/index/add/", params)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if status, _ := result["status"].(float64); status != 1 {
		msg, _ := result["msg"].(string)
		c.JSON(400, gin.H{"error": msg})
		return
	}
	c.JSON(200, gin.H{"port": port, "result": result})
}

// DeleteTunnel DELETE /api/nps/tunnels/:id?admin_password=xxx
func (h *NPSHandler) DeleteTunnel(c *gin.Context) {
	if !h.checkAdmin(c.Query("admin_password")) {
		c.JSON(403, gin.H{"error": "密码错误"})
		return
	}
	id := c.Param("id")
	if id == "" || strings.TrimSpace(id) == "" {
		c.JSON(400, gin.H{"error": "缺少 id"})
		return
	}
	params := url.Values{}
	params.Set("id", id)
	result, err := h.npsPost("/index/del/", params)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, result)
}

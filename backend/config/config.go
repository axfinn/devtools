package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用程序配置
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Security   SecurityConfig   `yaml:"security"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
	Limits     LimitsConfig     `yaml:"limits"`
	ShortURL   ShortURLConfig   `yaml:"shorturl"`
	Paste      PasteConfig      `yaml:"paste"`
	Chat       ChatConfig       `yaml:"chat"`
	MDShare    MDShareConfig    `yaml:"mdshare"`
	Excalidraw ExcalidrawConfig `yaml:"excalidraw"`
	Pregnancy  PregnancyConfig  `yaml:"pregnancy"`
	SSH        SSHConfig        `yaml:"ssh"`
	Expense    ExpenseConfig    `yaml:"expense"`
	Glucose    GlucoseConfig    `yaml:"glucose"`
	Household  HouseholdConfig  `yaml:"household"`
	DeepSeek           DeepSeekConfig           `yaml:"deepseek"`
	MiniMax            MiniMaxConfig            `yaml:"minimax"`
	MiniMaxMCP         MiniMaxMCPConfig         `yaml:"minimax_mcp"`
	MiniMaxTTS          MiniMaxTTSConfig          `yaml:"minimax_tts"`
	MiniMaxTokenPlan    MiniMaxTokenPlanConfig    `yaml:"minimax_token_plan"`
	MiniMaxVoiceCloning MiniMaxVoiceCloningConfig `yaml:"minimax_voice_cloning"`
	DashScope          DashScopeConfig          `yaml:"dashscope"`
	Bailian            BailianConfig            `yaml:"bailian"`
	AIGateway          AIGatewayConfig          `yaml:"ai_gateway"`
	NFSShare           NFSShareConfig           `yaml:"nfs_share"`
	ImageUnderstanding ImageUnderstandingConfig `yaml:"image_understanding"`
	TURN               TURNConfig               `yaml:"turn"`
	AutoDev            AutoDevConfig            `yaml:"autodev"`
	Proxy              ProxyConfig              `yaml:"proxy"`
	NPS                NPSConfig                `yaml:"nps"`
}

// NPSConfig NPS 端口映射管理配置
type NPSConfig struct {
	ServerURL      string `yaml:"server_url"`       // NPS 管理后台地址，如 http://nps.example.com:8080
	AuthKey        string `yaml:"auth_key"`         // NPS 配置文件里的 auth_key
	VKey           string `yaml:"vkey"`             // 客户端验证密钥（已在 NPS 里创建好）
	BridgeHost     string `yaml:"bridge_host"`      // npc 连接的 bridge 主机，留空则从 server_url 提取
	BridgePort     string `yaml:"bridge_port"`      // npc 连接 NPS 的端口，默认 8024
	AdminPassword  string `yaml:"admin_password"`   // devtools 访问密码（为空则禁用）
	PortRangeStart int    `yaml:"port_range_start"` // 自动分配端口区间起始（0 表示不启用）
	PortRangeEnd   int    `yaml:"port_range_end"`   // 自动分配端口区间结束
}

// ProxyConfig 科学上网代理配置
type ProxyConfig struct {
	AdminPassword string `yaml:"admin_password"` // 管理员密码，为空则禁用
	TunnelPort    string `yaml:"tunnel_port"`    // 独立 CONNECT 代理端口，留空则不启动（用于绕过 nginx）
	LocalPort     string `yaml:"local_port"`     // 本地节点代理端口，固定监听 127.0.0.1:port，留空则随机
	AlertEmail    string `yaml:"alert_email"`    // 所有节点不可用时的告警邮件收件人
	SMTPHost      string `yaml:"smtp_host"`      // SMTP 服务器，如 smtp.qq.com
	SMTPPort      int    `yaml:"smtp_port"`      // SMTP 端口，如 465（SSL）或 587（STARTTLS）
	SMTPUser      string `yaml:"smtp_user"`      // SMTP 登录用户名（发件人邮箱）
	SMTPPass      string `yaml:"smtp_pass"`      // SMTP 授权码/密码
}

// AutoDevConfig AutoDev AI 任务配置
type AutoDevConfig struct {
	AdminPassword string `yaml:"admin_password"` // 访问密码（必填）
	AutodevPath   string `yaml:"autodev_path"`   // autodev 可执行文件路径，默认 /opt/autodev/autodev
	DataDir       string `yaml:"data_dir"`       // 任务工作目录，默认 ./data/autodev
}

// TURNConfig WebRTC TURN 服务器配置（coturn use-auth-secret 模式）
type TURNConfig struct {
	Host   string `yaml:"host"`   // TURN 服务器地址，如 nps.jaxiu.cn
	Port   int    `yaml:"port"`   // TURN 端口，默认 3478
	Secret string `yaml:"secret"` // coturn static-auth-secret
	TTL    int    `yaml:"ttl"`    // 凭证有效期（秒），默认 3600
}

// NFSShareConfig NFS/SMB 文件分享配置
type NFSShareConfig struct {
	Enabled              bool          `yaml:"enabled"`               // 是否启用
	AdminPassword        string        `yaml:"admin_password"`        // 超管密码，为空则禁用
	MaxFileSizeMB        int64         `yaml:"max_file_size_mb"`      // 单文件最大大小（MB），0 不限
	DisableVideoDownload bool          `yaml:"disable_video_download"` // 禁止视频直接下载，只允许 HLS 播放
	DisableDownload      bool          `yaml:"disable_download"`       // 禁止所有文件直接下载
	Mounts               []MountConfig `yaml:"mounts"`                // 挂载点列表
}

// MountConfig 单个挂载点配置
type MountConfig struct {
	Name       string `yaml:"name"`        // 唯一标识名（英文，用作路径前缀）
	Type       string `yaml:"type"`        // "nfs" | "smb" | "local"
	Host       string `yaml:"host"`        // 服务器 IP 或主机名
	Export     string `yaml:"export"`      // NFS: 导出路径，如 /exports/data
	Share      string `yaml:"share"`       // SMB: 共享名，如 data
	Username   string `yaml:"username"`    // SMB: 用户名
	Password   string `yaml:"password"`    // SMB: 密码
	Domain     string `yaml:"domain"`      // SMB: 域名（可选）
	Options    string `yaml:"options"`     // 附加挂载选项（逗号分隔）
	MountPoint string `yaml:"mount_point"` // 本地挂载目录，留空则自动生成 ./data/mounts/<name>
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `yaml:"port"` // 服务端口
	Mode string `yaml:"mode"` // 运行模式: debug, release
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path           string `yaml:"path"`            // 数据库文件路径
	MaxConnections int    `yaml:"max_connections"` // 最大连接数
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	CORSOrigins []string `yaml:"cors_origins"` // 允许的跨域来源
	BcryptCost  int      `yaml:"bcrypt_cost"`  // Bcrypt 加密强度
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	CreatePerMinute int           `yaml:"create_per_minute"` // 创建操作每分钟限制
	Enabled         bool          `yaml:"enabled"`           // 是否启用限流
	Window          time.Duration `yaml:"-"`                 // 限流时间窗口
}

// LimitsConfig 内容大小限制配置
type LimitsConfig struct {
	PasteMaxContentSize int `yaml:"paste_max_content_size"` // 粘贴板最大文本大小（字节）
	PasteMaxImages      int `yaml:"paste_max_images"`       // 粘贴板最大图片数量
	PasteMaxTotalSize   int `yaml:"paste_max_total_size"`   // 粘贴板总大小（包含图片）
	MaxUploadSize       int `yaml:"max_upload_size"`        // 最大上传文件大小
}

// ShortURLConfig 短链服务配置
type ShortURLConfig struct {
	Password string `yaml:"password"` // 管理密码，为空则不需要密码
}

// PasteConfig 粘贴板配置
type PasteConfig struct {
	AdminPassword        string `yaml:"admin_password"`          // 管理员密码，可设置更多访问次数或永久
	DefaultVideoMaxViews int    `yaml:"default_video_max_views"` // 视频默认最大访问次数，默认10
	MaxFileSize          int64  `yaml:"max_file_size"`           // 最大文件大小，默认200MB
}

// ChatConfig 聊天室配置
type ChatConfig struct {
	AdminPassword  string `yaml:"admin_password"`   // 管理员密码，可管理所有聊天室
	TTSServiceURL  string `yaml:"tts_service_url"`  // TTS HTTP 服务地址，默认 http://127.0.0.1:8083
}

// MDShareConfig Markdown 分享配置
type MDShareConfig struct {
	AdminPassword      string `yaml:"admin_password"`       // 管理员密码，可管理所有分享
	DefaultMaxViews    int    `yaml:"default_max_views"`    // 默认最大查看次数，默认5
	DefaultExpiresDays int    `yaml:"default_expires_days"` // 默认过期天数，默认30
}

// ExcalidrawConfig Excalidraw 画图配置
type ExcalidrawConfig struct {
	AdminPassword      string `yaml:"admin_password"`       // 管理员密码，可永久保存
	DefaultExpiresDays int    `yaml:"default_expires_days"` // 默认过期天数，默认30
	MaxContentSize     int    `yaml:"max_content_size"`     // 最大内容大小，默认10MB
}

// PregnancyConfig 孕期管理配置
type PregnancyConfig struct {
	DefaultExpiresDays int `yaml:"default_expires_days"` // 默认过期天数，默认365
	MaxDataSize        int `yaml:"max_data_size"`        // 最大数据大小，默认1MB
}

// SSHConfig SSH 终端配置
type SSHConfig struct {
	AdminPassword       string `yaml:"admin_password"`        // 管理员密码
	HostKeyVerification bool   `yaml:"host_key_verification"` // 是否验证主机密钥（默认true）
	MaxSessionsPerUser  int    `yaml:"max_sessions_per_user"` // 每个用户最大会话数（默认10）
	SessionIdleTimeout  int    `yaml:"session_idle_timeout"`  // 会话空闲超时（分钟，默认5）
	HistoryMaxAgeDays   int    `yaml:"history_max_age_days"`  // 历史记录最大保存天数（默认30）
	SessionMaxAgeDays   int    `yaml:"session_max_age_days"`  // 不活跃会话最大保存天数（默认7）
	EncryptionKey       string `yaml:"encryption_key"`        // 加密密钥（Base64 编码的 32 字节密钥）
}

// ExpenseConfig 生活记账配置
type ExpenseConfig struct {
	DefaultExpiresDays int `yaml:"default_expires_days"` // 默认过期天数，默认365
	MaxDataSize        int `yaml:"max_data_size"`        // 最大数据大小，默认1MB
}

// GlucoseConfig 血糖监测配置
type GlucoseConfig struct {
	DefaultExpiresDays int `yaml:"default_expires_days"` // 默认过期天数，默认365
}

// DeepSeekConfig DeepSeek AI 配置
type DeepSeekConfig struct {
	APIKey string `yaml:"api_key"` // API Key，从 https://platform.deepseek.com 获取
	Model  string `yaml:"model"`   // 模型名称，默认 deepseek-chat
}

// MiniMaxConfig MiniMax AI 配置
type MiniMaxConfig struct {
	APIKey string `yaml:"api_key"` // API Key，从 https://platform.minimax.io 获取
	Model  string `yaml:"model"`   // 模型名称，默认 abab6.5s-chat
}

// MiniMaxMCPConfig MiniMax MCP 图像理解配置
type MiniMaxMCPConfig struct {
	APIKey         string   `yaml:"api_key"`          // MCP API Key
	APIHost        string   `yaml:"api_host"`         // MCP API Host
	Command        string   `yaml:"command"`          // MCP Server 启动命令
	Args           []string `yaml:"args"`             // MCP Server 启动参数
	TimeoutSeconds int      `yaml:"timeout_seconds"`  // 请求超时
	Transport      string   `yaml:"transport"`        // 传输协议: line 或 header
}

func (cfg MiniMaxMCPConfig) Timeout() time.Duration {
	if cfg.TimeoutSeconds <= 0 {
		return 120 * time.Second
	}
	return time.Duration(cfg.TimeoutSeconds) * time.Second
}

// MiniMaxTTSConfig MiniMax TTS 语音合成配置
type MiniMaxTTSConfig struct {
	APIKey  string `yaml:"api_key"`  // API Key（可与 MiniMax APIKey 共用）
	BaseURL string `yaml:"base_url"` // TTS 上游地址，默认 https://api.minimaxi.com
}

// MiniMaxTokenPlanConfig MiniMax Token Plan 媒体生成配置（TTS HD / Hailuo 视频 / Music / image-01）
type MiniMaxTokenPlanConfig struct {
	APIKey  string `yaml:"api_key"`  // API Key（可与 MiniMax APIKey 共用）
	BaseURL string `yaml:"base_url"` // 上游地址，默认 https://api.minimaxi.com
}

// MiniMaxVoiceCloningConfig MiniMax Voice Cloning 音色克隆配置
type MiniMaxVoiceCloningConfig struct {
	APIKey  string `yaml:"api_key"`  // API Key（可与 MiniMax APIKey 共用）
	BaseURL string `yaml:"base_url"` // 上游地址，默认 https://api.minimaxi.com
}

// HouseholdConfig 家庭物品整理模块配置
type HouseholdConfig struct {
	DefaultExpiresDays int `yaml:"default_expires_days"` // 默认过期天数
}

// DashScopeConfig 阿里云 DashScope 文本模型配置（coding 端点，支持多品牌模型）
type DashScopeConfig struct {
	APIKey  string `yaml:"api_key"`  // DashScope API Key（可共用百炼 APIKey）
	BaseURL string `yaml:"base_url"` // API 基础地址，默认 https://coding.dashscope.aliyuncs.com/v1
}

// BailianConfig 阿里云百炼图片模型配置
type BailianConfig struct {
	APIKey             string               `yaml:"api_key"`              // DashScope API Key
	AdminPassword      string               `yaml:"admin_password"`       // 调试与开放 API 管理密码
	BaseURL            string               `yaml:"base_url"`             // API 基础地址
	DefaultWaitSeconds int                  `yaml:"default_wait_seconds"` // 默认同步等待秒数
	TaskRetentionDays  int                  `yaml:"task_retention_days"`  // 任务保留天数
	Models             []BailianModelConfig `yaml:"models"`               // 模型配额配置
}

type BailianModelConfig struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Enabled     bool   `yaml:"enabled"`
	TotalQuota  int    `yaml:"total_quota"`
	ExpiresAt   string `yaml:"expires_at"`
	DefaultSize string `yaml:"default_size"`
	Description string `yaml:"description"`
}

type AIGatewayConfig struct {
	SuperAdminPassword      string                   `yaml:"super_admin_password"`
	DefaultKeyExpiresDays   int                      `yaml:"default_key_expires_days"`
	DefaultRateLimitPerHour int                      `yaml:"default_rate_limit_per_hour"`
	RequestRetentionDays    int                      `yaml:"request_retention_days"`
	UpstreamTimeoutSeconds  int                      `yaml:"upstream_timeout_seconds"`
	Proxy                   AIGatewayProxyConfig     `yaml:"proxy"`
	Pricing                 []AIGatewayPricingConfig `yaml:"pricing"`
}

type AIGatewayProxyConfig struct {
	Model         string                      `yaml:"model"`
	UpstreamModel string                      `yaml:"upstream_model"`
	APIURL        string                      `yaml:"api_url"`
	APIKey        string                      `yaml:"api_key"`
	Models        []AIGatewayProxyModelConfig `yaml:"models"`
}

type AIGatewayProxyModelConfig struct {
	Model         string `yaml:"model"`
	UpstreamModel string `yaml:"upstream_model"`
	Description   string `yaml:"description"`
}

// ImageUnderstandingConfig 图像理解模块配置
type ImageUnderstandingConfig struct {
	// AdminPassword 用于查看 Qwen 视觉理解请求流水，独立于 super_admin_password
	AdminPassword string `yaml:"admin_password"`
}

type AIGatewayPricingConfig struct {
	Model             string  `yaml:"model"`
	Provider          string  `yaml:"provider"`
	InputPer1KTokens  float64 `yaml:"input_per_1k_tokens"`
	OutputPer1KTokens float64 `yaml:"output_per_1k_tokens"`
	RequestCost       float64 `yaml:"request_cost"`
	Currency          string  `yaml:"currency"`
}

var globalConfig *Config

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: "8080",
			Mode: "release",
		},
		Database: DatabaseConfig{
			Path:           "./data/paste.db",
			MaxConnections: 10,
		},
		Security: SecurityConfig{
			CORSOrigins: []string{"*"},
			BcryptCost:  10,
		},
		RateLimit: RateLimitConfig{
			CreatePerMinute: 10,
			Enabled:         true,
			Window:          time.Minute,
		},
		Limits: LimitsConfig{
			PasteMaxContentSize: 10 * 1024 * 1024, // 10MB
			PasteMaxImages:      15,
			PasteMaxTotalSize:   50 * 1024 * 1024, // 50MB
			MaxUploadSize:       200 * 1024 * 1024, // 200MB
		},
		Paste: PasteConfig{
			DefaultVideoMaxViews: 10,
			MaxFileSize:          200 * 1024 * 1024, // 200MB
		},
		MDShare: MDShareConfig{
			DefaultMaxViews:    5,
			DefaultExpiresDays: 30,
		},
		Excalidraw: ExcalidrawConfig{
			DefaultExpiresDays: 30,
			MaxContentSize:     10 * 1024 * 1024, // 10MB
		},
		Pregnancy: PregnancyConfig{
			DefaultExpiresDays: 365,
			MaxDataSize:        1024 * 1024, // 1MB
		},
		SSH: SSHConfig{
			HostKeyVerification: true,
			MaxSessionsPerUser:  10,
			SessionIdleTimeout:  5,  // 5分钟
			HistoryMaxAgeDays:   30, // 30天
			SessionMaxAgeDays:   7,  // 7天
		},
		Expense: ExpenseConfig{
			DefaultExpiresDays: 365,
			MaxDataSize:        1024 * 1024, // 1MB
		},
		Glucose: GlucoseConfig{
			DefaultExpiresDays: 365,
		},
		Household: HouseholdConfig{
			DefaultExpiresDays: 365,
		},
		DeepSeek: DeepSeekConfig{
			Model: "deepseek-chat",
		},
		MiniMax: MiniMaxConfig{
			Model: "abab6.5s-chat",
		},
		MiniMaxMCP: MiniMaxMCPConfig{
			APIHost:        "https://api.minimaxi.com",
			Command:        "uvx",
			Args:           []string{"minimax-coding-plan-mcp"},
			TimeoutSeconds: 600,
			Transport:      "line",
		},
		MiniMaxTokenPlan: MiniMaxTokenPlanConfig{
			BaseURL: "https://api.minimaxi.com",
		},
		DashScope: DashScopeConfig{
			BaseURL: "https://coding.dashscope.aliyuncs.com/v1",
		},
		Bailian: BailianConfig{
			BaseURL:            "https://dashscope.aliyuncs.com",
			DefaultWaitSeconds: 45,
			TaskRetentionDays:  180,
			Models: []BailianModelConfig{
				{Name: "qwen-image-2.0-pro", Type: "multimodal", Enabled: true, TotalQuota: 100, ExpiresAt: "2026-06-01", DefaultSize: "1328x1328", Description: "文生图/图像编辑，同步接口"},
				{Name: "qwen-image-2.0", Type: "multimodal", Enabled: true, TotalQuota: 100, ExpiresAt: "2026-06-01", DefaultSize: "1328x1328", Description: "文生图/图像编辑，同步接口"},
				{Name: "qwen-image-2.0-pro-2026-03-03", Type: "multimodal", Enabled: true, TotalQuota: 100, ExpiresAt: "2026-06-01", DefaultSize: "1328x1328", Description: "快照版本，同步接口"},
				{Name: "qwen-image-2.0-2026-03-03", Type: "multimodal", Enabled: true, TotalQuota: 100, ExpiresAt: "2026-06-01", DefaultSize: "1328x1328", Description: "快照版本，同步接口"},
				{Name: "qwen-image-plus-2026-01-09", Type: "text2image", Enabled: true, TotalQuota: 100, ExpiresAt: "2026-04-09", DefaultSize: "1024*1024", Description: "异步文生图快照版本"},
				{Name: "wan2.6-i2v-flash", Type: "image2video", Enabled: true, TotalQuota: 50, ExpiresAt: "2026-04-16", DefaultSize: "1280x720", Description: "图生视频 Flash"},
				{Name: "wan2.6-i2v", Type: "image2video", Enabled: false, TotalQuota: 0, ExpiresAt: "", DefaultSize: "1280x720", Description: "无免费额度，默认禁用"},
				{Name: "qwen-image-plus", Type: "text2image", Enabled: false, TotalQuota: 0, ExpiresAt: "", DefaultSize: "1024*1024", Description: "无免费额度，默认禁用"},
				{Name: "qwen-image-edit-plus", Type: "multimodal", Enabled: false, TotalQuota: 0, ExpiresAt: "", DefaultSize: "1328x1328", Description: "无免费额度，默认禁用"},
				{Name: "qwen-image-max", Type: "text2image", Enabled: false, TotalQuota: 0, ExpiresAt: "", DefaultSize: "1024*1024", Description: "无免费额度，默认禁用"},
			},
		},
		AIGateway: AIGatewayConfig{
			DefaultKeyExpiresDays:   90,
			DefaultRateLimitPerHour: 1000,
			RequestRetentionDays:    180,
			UpstreamTimeoutSeconds:  180,
			Proxy: AIGatewayProxyConfig{
				Model: "proxy-chat",
			},
			Pricing: []AIGatewayPricingConfig{},
		},
		NFSShare: NFSShareConfig{
			Enabled:              false,
			MaxFileSizeMB:        0,
			DisableVideoDownload: true,
			DisableDownload:      true,
			Mounts:               []MountConfig{},
		},
		TURN: TURNConfig{
			Port: 3478,
			TTL:  3600,
		},
		AutoDev: AutoDevConfig{
			AutodevPath: "/opt/autodev/autodev",
			DataDir:     "./data/autodev",
		},
	}
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	// 先加载默认配置
	cfg := DefaultConfig()

	// 尝试读取配置文件
	data, err := os.ReadFile(path)
	if err != nil {
		// 如果配置文件不存在，返回默认配置
		if os.IsNotExist(err) {
			globalConfig = cfg
			return globalConfig, nil
		}
		return nil, err
	}

	// 解析配置文件，覆盖默认值
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// 从环境变量覆盖配置
	if port := os.Getenv("PORT"); port != "" {
		cfg.Server.Port = port
	}
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}
	// DeepSeek API Key 支持环境变量覆盖
	if apiKey := os.Getenv("DEEPSEEK_API_KEY"); apiKey != "" {
		cfg.DeepSeek.APIKey = apiKey
	}
	// MiniMax API Key 支持环境变量覆盖
	if apiKey := os.Getenv("MINIMAX_API_KEY"); apiKey != "" {
		cfg.MiniMax.APIKey = apiKey
	}
	// MiniMax MCP API Key 支持环境变量覆盖
	if apiKey := os.Getenv("MINIMAX_MCP_API_KEY"); apiKey != "" {
		cfg.MiniMaxMCP.APIKey = apiKey
	}
	if apiHost := os.Getenv("MINIMAX_MCP_API_HOST"); apiHost != "" {
		cfg.MiniMaxMCP.APIHost = apiHost
	}
	// MiniMax TTS API Key 支持环境变量覆盖
	if apiKey := os.Getenv("MINIMAX_TTS_API_KEY"); apiKey != "" {
		cfg.MiniMaxTTS.APIKey = apiKey
	}
	// MiniMax Token Plan API Key 支持环境变量覆盖
	if apiKey := os.Getenv("MINIMAX_TOKEN_PLAN_API_KEY"); apiKey != "" {
		cfg.MiniMaxTokenPlan.APIKey = apiKey
	}
	// DashScope API Key 支持环境变量覆盖
	if apiKey := os.Getenv("DASHSCOPE_API_KEY"); apiKey != "" {
		cfg.DashScope.APIKey = apiKey
	}
	// 百炼 API Key 支持环境变量覆盖
	if apiKey := os.Getenv("BAILIAN_API_KEY"); apiKey != "" {
		cfg.Bailian.APIKey = apiKey
	}
	if adminPassword := os.Getenv("BAILIAN_ADMIN_PASSWORD"); adminPassword != "" {
		cfg.Bailian.AdminPassword = adminPassword
	}
	if superAdmin := os.Getenv("AI_GATEWAY_SUPER_ADMIN_PASSWORD"); superAdmin != "" {
		cfg.AIGateway.SuperAdminPassword = superAdmin
	}
	if proxyAPIURL := os.Getenv("AI_GATEWAY_PROXY_API_URL"); proxyAPIURL != "" {
		cfg.AIGateway.Proxy.APIURL = proxyAPIURL
	}
	if proxyAPIKey := os.Getenv("AI_GATEWAY_PROXY_API_KEY"); proxyAPIKey != "" {
		cfg.AIGateway.Proxy.APIKey = proxyAPIKey
	}
	if proxyModel := os.Getenv("AI_GATEWAY_PROXY_MODEL"); proxyModel != "" {
		cfg.AIGateway.Proxy.Model = proxyModel
	}
	if upstreamModel := os.Getenv("AI_GATEWAY_PROXY_UPSTREAM_MODEL"); upstreamModel != "" {
		cfg.AIGateway.Proxy.UpstreamModel = upstreamModel
	}
	// AutoDev 配置支持环境变量覆盖
	if autodevPassword := os.Getenv("AUTODEV_ADMIN_PASSWORD"); autodevPassword != "" {
		cfg.AutoDev.AdminPassword = autodevPassword
	}
	if autodevPath := os.Getenv("AUTODEV_PATH"); autodevPath != "" {
		cfg.AutoDev.AutodevPath = autodevPath
	}
	if autodevDataDir := os.Getenv("AUTODEV_DATA_DIR"); autodevDataDir != "" {
		cfg.AutoDev.DataDir = autodevDataDir
	}
	// TTS 服务地址支持环境变量覆盖
	if ttsURL := os.Getenv("TTS_SERVICE_URL"); ttsURL != "" {
		cfg.Chat.TTSServiceURL = ttsURL
	}
	if cfg.Chat.TTSServiceURL == "" {
		cfg.Chat.TTSServiceURL = "http://127.0.0.1:8083"
	}

	globalConfig = cfg
	return globalConfig, nil
}

// Get 返回全局配置
func Get() *Config {
	if globalConfig == nil {
		globalConfig = DefaultConfig()
	}
	return globalConfig
}

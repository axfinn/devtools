package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ShortURL   ShortURLConfig   `yaml:"shorturl"`
	MDShare    MDShareConfig    `yaml:"mdshare"`
	Excalidraw ExcalidrawConfig `yaml:"excalidraw"`
}

type ShortURLConfig struct {
	Password string `yaml:"password"` // 管理密码，为空则不需要密码
}

type MDShareConfig struct {
	AdminPassword      string `yaml:"admin_password"`       // 管理员密码，可管理所有分享
	DefaultMaxViews    int    `yaml:"default_max_views"`    // 默认最大查看次数，默认5
	DefaultExpiresDays int    `yaml:"default_expires_days"` // 默认过期天数，默认30
}

type ExcalidrawConfig struct {
	AdminPassword      string `yaml:"admin_password"`       // 管理员密码，可永久保存
	DefaultExpiresDays int    `yaml:"default_expires_days"` // 默认过期天数，默认30
	MaxContentSize     int    `yaml:"max_content_size"`     // 最大内容大小，默认10MB
}

var globalConfig *Config

// Load loads configuration from file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// 如果配置文件不存在，返回默认配置
		if os.IsNotExist(err) {
			globalConfig = &Config{}
			return globalConfig, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	globalConfig = &cfg
	return globalConfig, nil
}

// Get returns the global config
func Get() *Config {
	if globalConfig == nil {
		globalConfig = &Config{}
	}
	return globalConfig
}

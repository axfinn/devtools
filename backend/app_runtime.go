package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/state"
)

type appRuntime struct {
	cfg            *config.Config
	port           string
	db             *models.DB
	transientStore state.TransientStore
}

func newAppRuntime() (*appRuntime, error) {
	cfg := loadConfig()

	transientStore, err := state.New(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("Redis 初始化失败: %w", err)
	}

	imageTaskTTL := time.Duration(cfg.Redis.ImageTaskTTLHours) * time.Hour
	if imageTaskTTL <= 0 {
		imageTaskTTL = 7 * 24 * time.Hour
	}
	if err := transientStore.FailProcessingImageTasks(
		context.Background(),
		"服务已重启，任务中断，请重新提交",
		imageTaskTTL,
	); err != nil {
		log.Printf("恢复图像任务状态失败: %v", err)
	}

	if err := os.MkdirAll("./data", 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	dbPath := envOrDefault("DB_PATH", "./data/paste.db")
	db, err := models.NewDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("数据库初始化失败: %w", err)
	}

	if err := db.InitAll(); err != nil {
		db.Close()
		return nil, err
	}

	return &appRuntime{
		cfg:            cfg,
		port:           envOrDefault("PORT", "8080"),
		db:             db,
		transientStore: transientStore,
	}, nil
}

func (rt *appRuntime) Close() {
	if rt == nil || rt.db == nil {
		return
	}
	_ = rt.db.Close()
}

func loadConfig() *config.Config {
	configPath := envOrDefault("CONFIG_PATH", "./config.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("加载配置文件失败，使用默认配置: %v", err)
		return config.Get()
	}
	return cfg
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

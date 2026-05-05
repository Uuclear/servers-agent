package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/slouch/vps-monitor/internal/config"
	"github.com/slouch/vps-monitor/internal/server/api"
	"github.com/slouch/vps-monitor/internal/server/storage"
)

func main() {
	// 命令行参数
	port := flag.Int("port", 0, "服务器端口（默认 8080）")
	dbPath := flag.String("db", "", "数据库文件路径")
	tokens := flag.String("tokens", "", "有效 token 列表（逗号分隔）")
	configPath := flag.String("config", "", "配置文件路径（YAML）")
	flag.Parse()

	var cfg *config.ServerConfig

	// 优先使用配置文件
	if *configPath != "" {
		c, err := config.LoadServerConfig(*configPath)
		if err != nil {
			log.Fatalf("加载配置文件失败: %v", err)
		}
		cfg = c
		log.Printf("已加载配置文件: %s", *configPath)
	} else {
		// 使用命令行参数或默认值
		cfg = &config.ServerConfig{
			Port:        8080,
			Database:    "./vps-monitor.db",
			Tokens:      []string{},
			RetentionDays: 7,
		}

		if *port > 0 {
			cfg.Port = *port
		}
		if *dbPath != "" {
			cfg.Database = *dbPath
		}
		if *tokens != "" {
			cfg.Tokens = splitTokens(*tokens)
		}
	}

	// 确保数据目录存在
	if cfg.Database != "./vps-monitor.db" {
		dir := getDir(cfg.Database)
		if dir != "" {
			os.MkdirAll(dir, 0755)
		}
	}

	// 初始化存储
	store, err := storage.New(cfg.Database)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer store.Close()

	// Token 验证
	validTokens := cfg.Tokens
	if len(validTokens) == 0 {
		log.Println("警告: 未配置有效 token，将接受所有上报")
		validTokens = []string{"*"}
	}

	// 启动定时清理任务
	go startCleanupTask(store, cfg.RetentionDays)

	// 启动状态检查任务
	go startStatusCheckTask(store)

	// 创建 API Handler
	handler := api.NewHandler(store, validTokens)

	// 设置 Gin
	//gin.SetMode(gin.ReleaseMode)  // 暂时禁用 Release 模式进行调试
	r := gin.New()
	r.Use(gin.Recovery())

	// 添加测试路由
	r.POST("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	handler.SetupRoutes(r)

	log.Printf("路由已设置完成")

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("VPS Monitor Server 启动于 %s", addr)
	log.Printf("Web UI: http://localhost%s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// splitTokens - 按逗号分割 token
func splitTokens(s string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	return result
}

// getDir - 获取文件目录路径
func getDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return ""
}

// startCleanupTask - 启动数据清理任务
func startCleanupTask(store *storage.Storage, retentionDays int) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if err := store.CleanOldMetrics(retentionDays); err != nil {
			log.Printf("数据清理失败: %v", err)
		} else {
			log.Printf("数据清理完成，保留 %d 天", retentionDays)
		}
	}
}

// startStatusCheckTask - 启动状态检查任务
func startStatusCheckTask(store *storage.Storage) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		servers, err := store.GetServers()
		if err != nil {
			continue
		}
		for _, s := range servers {
			if time.Since(s.LastSeen) > 60*time.Second && s.Status == "online" {
				store.UpdateServerStatus(s.ID, "offline")
				log.Printf("服务器 %s 状态更新为 offline", s.Hostname)
			}
		}
	}
}
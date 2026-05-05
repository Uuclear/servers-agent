package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/slouch/vps-monitor/internal/collector"
	"github.com/slouch/vps-monitor/internal/config"
)

func main() {
	// 命令行参数
	serverURL := flag.String("server", "", "监控服务器地址")
	token := flag.String("token", "", "Agent 注册 token")
	interval := flag.Int("interval", 0, "采集间隔（秒）")
	configPath := flag.String("config", "", "配置文件路径（YAML）")
	flag.Parse()

	var cfg *config.AgentConfig

	// 优先使用配置文件
	if *configPath != "" {
		c, err := config.LoadAgentConfig(*configPath)
		if err != nil {
			log.Fatalf("加载配置文件失败: %v", err)
		}
		cfg = c
		log.Printf("已加载配置文件: %s", *configPath)
	} else {
		// 使用命令行参数
		if *serverURL == "" {
			log.Fatal("必须指定 -server 参数或使用配置文件")
		}
		if *token == "" {
			log.Fatal("必须指定 -token 参数或使用配置文件")
		}

		cfg = &config.AgentConfig{
			Server:   *serverURL,
			Token:    *token,
			Interval: 10,
			DiskPaths: []string{},
			Services: []config.ServiceCheck{},
		}

		if *interval > 0 {
			cfg.Interval = *interval
		}
	}

	// 转换服务检测配置
	services := make([]collector.ServiceCheckConfig, len(cfg.Services))
	for i, s := range cfg.Services {
		services[i] = collector.ServiceCheckConfig{
			Name:   s.Name,
			Type:   s.Type,
			Target: s.Target,
		}
	}

	// 创建采集器
	coll := collector.NewCollector(services, cfg.DiskPaths)

	log.Printf("VPS Monitor Agent 启动")
	log.Printf("服务器: %s", cfg.Server)
	log.Printf("采集间隔: %d 秒", cfg.Interval)
	log.Printf("磁盘监控: %v", cfg.DiskPaths)
	log.Printf("服务检测: %d 个", len(services))

	// 定时采集上报
	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer ticker.Stop()

	// 立即执行一次
	report(coll, cfg.Server, cfg.Token)

	for range ticker.C {
		report(coll, cfg.Server, cfg.Token)
	}
}

// report - 采集并上报
func report(coll *collector.Collector, serverURL, token string) {
	data, err := coll.Collect()
	if err != nil {
		log.Printf("采集失败: %v", err)
		return
	}

	data.Token = token

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("JSON 序列化失败: %v", err)
		return
	}

	url := fmt.Sprintf("%s/api/report", serverURL)
	resp, err := http.Post(url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		log.Printf("上报失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("上报失败: HTTP %d", resp.StatusCode)
	} else {
		log.Printf("上报成功: CPU=%.1f%%, 内存=%.1f%%, Swap=%.1f%%",
			data.CPU.UsagePercent,
			data.Memory.UsedPercent,
			data.Swap.UsedPercent)

		// 打印磁盘信息
		for _, d := range data.Disk {
			log.Printf("  磁盘 %s: %.1f%%", d.Path, d.UsedPercent)
		}

		// 打印服务检测结果
		for _, svc := range data.Services {
			log.Printf("  服务 %s: %s (%dms)", svc.Name, svc.Status, svc.ResponseMs)
		}
	}
}
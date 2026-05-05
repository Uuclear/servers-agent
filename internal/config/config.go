package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// ServerConfig - Server 配置
type ServerConfig struct {
	Port        int      `yaml:"port"`
	Database    string   `yaml:"database"`
	Tokens      []string `yaml:"tokens"`
	RetentionDays int    `yaml:"retention_days"`
	Web         WebConfig `yaml:"web"`
	Logging     LoggingConfig `yaml:"logging"`
}

// WebConfig - Web UI 配置
type WebConfig struct {
	StaticDir string `yaml:"static_dir"`
}

// LoggingConfig - 日志配置
type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// AgentConfig - Agent 配置
type AgentConfig struct {
	Server    string         `yaml:"server"`
	Token     string         `yaml:"token"`
	Interval  int            `yaml:"interval"`
	DiskPaths []string       `yaml:"disk_paths"`
	Services  []ServiceCheck `yaml:"services"`
	Logging   LoggingConfig  `yaml:"logging"`
}

// ServiceCheck - 服务检测配置
type ServiceCheck struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`
	Target string `yaml:"target"`
}

// LoadServerConfig - 加载 Server 配置
func LoadServerConfig(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ServerConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 设置默认值
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	if cfg.Database == "" {
		cfg.Database = "./vps-monitor.db"
	}
	if cfg.RetentionDays == 0 {
		cfg.RetentionDays = 7
	}

	return &cfg, nil
}

// LoadAgentConfig - 加载 Agent 配置
func LoadAgentConfig(path string) (*AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg AgentConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 设置默认值
	if cfg.Interval == 0 {
		cfg.Interval = 10
	}

	return &cfg, nil
}
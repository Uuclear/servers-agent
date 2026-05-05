package models

import "time"

// Server - 被监控的服务器注册信息
type Server struct {
	ID        int64     `json:"id"`
	Token     string    `json:"token"`     // Agent 注册 token
	Name      string    `json:"name"`      // 显示名称
	Hostname  string    `json:"hostname"`  // 主机名
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	LastSeen  time.Time `json:"last_seen"` // 最后上报时间
	Status    string    `json:"status"`    // online/offline/warning
	CreatedAt time.Time `json:"created_at"`
}

// MetricRecord - 时序数据记录
type MetricRecord struct {
	ID        int64     `json:"id"`
	ServerID  int64     `json:"server_id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`      // cpu/memory/disk/network
	Path      string    `json:"path"`      // 磁盘分区路径（disk 类型）
	Value     float64   `json:"value"`     // 主值（使用率等）
	Detail    string    `json:"detail"`    // JSON 详情（可选）
}

// ServiceCheck - 服务检测结果
type ServiceCheck struct {
	ID        int64     `json:"id"`
	ServerID  int64     `json:"server_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Target    string    `json:"target"`
	Status    string    `json:"status"`
	ResponseMs int64    `json:"response_ms"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// DashboardSummary - Dashboard 概览数据
type DashboardSummary struct {
	TotalServers   int `json:"total_servers"`
	OnlineCount    int `json:"online_count"`
	OfflineCount   int `json:"offline_count"`
	WarningCount   int `json:"warning_count"`
}

// ServerMetricDetail - 单服务器指标详情
type ServerMetricDetail struct {
	Server       Server             `json:"server"`
	CPUHistory   []MetricPoint      `json:"cpu_history"`
	MemoryHistory []MetricPoint     `json:"memory_history"`
	DiskHistory  map[string][]MetricPoint `json:"disk_history"` // 按分区
	NetworkHistory []MetricPoint    `json:"network_history"`
}

// MetricPoint - 单个时序数据点
type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}
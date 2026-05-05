package protocol

// Agent 向 Server 上报的数据结构
type MetricsReport struct {
	Token     string    `json:"token"`      // Agent 注册 token
	Hostname  string    `json:"hostname"`   // 主机名
	Timestamp int64     `json:"timestamp"`  // Unix 时间戳
	System    System    `json:"system"`     // 系统信息
	CPU       CPU       `json:"cpu"`        // CPU 指标
	Memory    Memory    `json:"memory"`     // 内存指标
	Swap      Swap      `json:"swap"`       // Swap 指标
	Disk      []Disk    `json:"disk"`       // 磁盘指标（多分区）
	Network   Network   `json:"network"`    // 网络指标
	Services  []Service `json:"services"`   // 服务可用性（可选）
}

// 系统基础信息
type System struct {
	OS     string  `json:"os"`     // 操作系统
	Arch   string  `json:"arch"`   // 架构
	Uptime uint64  `json:"uptime"` // 运行时间（秒）
	Load1  float64 `json:"load1"`  // 1分钟负载
	Load5  float64 `json:"load5"`  // 5分钟负载
	Load15 float64 `json:"load15"` // 15分钟负载
}

// CPU 指标
type CPU struct {
	UsagePercent   float64 `json:"usage_percent"`   // 总使用率
	UserPercent    float64 `json:"user_percent"`    // 用户态
	SystemPercent  float64 `json:"system_percent"`  // 系统态
	IdlePercent    float64 `json:"idle_percent"`    // 空闲
	IowaitPercent  float64 `json:"iowait_percent"`  // IO等待
	CoreCount      int     `json:"core_count"`      // CPU 核数
}

// 内存指标
type Memory struct {
	Total       uint64  `json:"total"`        // 总内存 (bytes)
	Used        uint64  `json:"used"`         // 已用内存
	Available   uint64  `json:"available"`    // 可用内存
	UsedPercent float64 `json:"used_percent"` // 使用率
	Buffers     uint64  `json:"buffers"`      // 缓冲区内存
	Cached      uint64  `json:"cached"`       // 缓存内存
}

// Swap 指标
type Swap struct {
	Total       uint64  `json:"total"`        // 总 Swap (bytes)
	Used        uint64  `json:"used"`         // 已用 Swap
	Free        uint64  `json:"free"`         // 剩余 Swap
	UsedPercent float64 `json:"used_percent"` // 使用率
}

// 单个磁盘分区指标
type Disk struct {
	Path        string  `json:"path"`         // 挂载路径
	Total       uint64  `json:"total"`        // 总容量 (bytes)
	Used        uint64  `json:"used"`         // 已用容量
	Free        uint64  `json:"free"`         // 剩余容量
	UsedPercent float64 `json:"used_percent"` // 使用率
	Fstype      string  `json:"fstype"`       // 文件系统类型
}

// 网络指标（累计流量）
type Network struct {
	BytesSent   uint64 `json:"bytes_sent"`   // 发送总量 (bytes)
	BytesRecv   uint64 `json:"bytes_recv"`   // 接收总量
	PacketsSent uint64 `json:"packets_sent"` // 发送包数
	PacketsRecv uint64 `json:"packets_recv"` // 接收包数
	ErrIn       uint64 `json:"err_in"`       // 接收错误数
	ErrOut      uint64 `json:"err_out"`      // 发送错误数
	DropIn      uint64 `json:"drop_in"`      // 接收丢包数
	DropOut     uint64 `json:"drop_out"`     // 发送丢包数
}

// 服务可用性检测结果
type Service struct {
	Name       string `json:"name"`        // 服务名称
	Type       string `json:"type"`        // 类型: http/tcp/ssl
	Target     string `json:"target"`      // 目标地址
	Status     string `json:"status"`      // 状态: ok/error/warning
	ResponseMs int64  `json:"response_ms"` // 响应时间（毫秒）
	Message    string `json:"message"`     // 错误信息（如有）
}
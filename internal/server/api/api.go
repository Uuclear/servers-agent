package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/slouch/vps-monitor/internal/server/storage"
	"github.com/slouch/vps-monitor/pkg/protocol"
)

// Handler - API handlers
type Handler struct {
	store  *storage.Storage
	tokens map[string]bool
}

// NewHandler - 创建 Handler
func NewHandler(store *storage.Storage, validTokens []string) *Handler {
	tokens := make(map[string]bool)
	for _, t := range validTokens {
		tokens[t] = true
	}
	return &Handler{store: store, tokens: tokens}
}

// SetupRoutes - 设置路由
func (h *Handler) SetupRoutes(r *gin.Engine) {
	// 添加测试路由
	r.POST("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "test ok"})
	})

	// 直接定义路由
	r.POST("/api/report", h.Report)
	r.GET("/api/servers", h.GetServers)
	r.GET("/api/servers/:id", h.GetServerDetail)
	r.PUT("/api/servers/:id/name", h.SetServerName)
	r.GET("/api/dashboard", h.GetDashboard)
	r.GET("/api/metrics/:id/:type", h.GetMetrics)
	r.GET("/api/metrics/:id/disk/*path", h.GetDiskMetrics)
	r.GET("/api/disks/:id", h.GetDiskPaths)

	// 静态文件（放在最后）
	r.GET("/", func(c *gin.Context) {
		c.File("./web/static/index.html")
	})
	r.GET("/static/*filepath", func(c *gin.Context) {
		c.File("./web/static" + c.Param("filepath"))
	})
}

// Report - Agent 上报数据
func (h *Handler) Report(c *gin.Context) {
	var report protocol.MetricsReport
	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !h.tokens[report.Token] {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	server, err := h.store.RegisterServer(report.Token, report.Hostname, report.System.OS, report.System.Arch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.SaveMetrics(server.ID, &report); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(report.Services) > 0 {
		if err := h.store.SaveServiceChecks(server.ID, report.Services); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	status := h.determineStatus(&report)
	if err := h.store.UpdateServerStatus(server.ID, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetServers - 获取服务器列表
func (h *Handler) GetServers(c *gin.Context) {
	servers, err := h.store.GetServers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i, s := range servers {
		if time.Since(s.LastSeen) > 60*time.Second && s.Status == "online" {
			h.store.UpdateServerStatus(s.ID, "offline")
			servers[i].Status = "offline"
		}
	}

	c.JSON(http.StatusOK, servers)
}

// GetServerDetail - 获取服务器详情
func (h *Handler) GetServerDetail(c *gin.Context) {
	id := parseID(c.Param("id"))

	server, err := h.store.GetServerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	rangeParam := c.DefaultQuery("range", "1h")
	since := parseTimeRange(rangeParam)

	cpuHistory, _ := h.store.GetMetricsHistory(id, "cpu", since)
	memoryHistory, _ := h.store.GetMetricsHistory(id, "memory", since)
	swapHistory, _ := h.store.GetMetricsHistory(id, "swap", since)
	loadHistory, _ := h.store.GetMetricsHistory(id, "load1", since)
	diskHistory, _ := h.store.GetAllDiskHistory(id, since)

	sentRate, recvRate, _ := h.store.GetNetworkRate(id)

	detail := gin.H{
		"server":        server,
		"cpu_history":   cpuHistory,
		"memory_history": memoryHistory,
		"swap_history":  swapHistory,
		"load_history":  loadHistory,
		"disk_history":  diskHistory,
		"network_sent_rate": sentRate,
		"network_recv_rate": recvRate,
	}

	c.JSON(http.StatusOK, detail)
}

// SetServerName - 设置服务器名称
func (h *Handler) SetServerName(c *gin.Context) {
	id := parseID(c.Param("id"))

	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.SetServerName(id, req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetDashboard - 获取 Dashboard 概览
func (h *Handler) GetDashboard(c *gin.Context) {
	summary, err := h.store.GetDashboardSummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	servers, err := h.store.GetServers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	latestMetrics := make(map[int64]map[string]float64)
	for _, s := range servers {
		m, _ := h.store.GetLatestMetrics(s.ID)
		latestMetrics[s.ID] = m
	}

	c.JSON(http.StatusOK, gin.H{
		"summary":        summary,
		"servers":        servers,
		"latest_metrics": latestMetrics,
	})
}

// GetMetrics - 获取指定类型历史指标
func (h *Handler) GetMetrics(c *gin.Context) {
	id := parseID(c.Param("id"))
	metricType := c.Param("type")

	since := parseTimeRange(c.DefaultQuery("range", "1h"))

	history, err := h.store.GetMetricsHistory(id, metricType, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetDiskMetrics - 获取特定分区的历史数据
func (h *Handler) GetDiskMetrics(c *gin.Context) {
	id := parseID(c.Param("id"))
	path := c.Param("path")

	since := parseTimeRange(c.DefaultQuery("range", "1h"))

	history, err := h.store.GetDiskHistory(id, path, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetDiskPaths - 获取服务器所有磁盘分区路径
func (h *Handler) GetDiskPaths(c *gin.Context) {
	id := parseID(c.Param("id"))

	paths, err := h.store.GetDiskPaths(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, paths)
}

// determineStatus - 根据指标判断服务器状态
func (h *Handler) determineStatus(report *protocol.MetricsReport) string {
	if report.CPU.UsagePercent > 90 || report.Memory.UsedPercent > 90 || report.Swap.UsedPercent > 95 {
		return "warning"
	}

	for _, d := range report.Disk {
		if d.UsedPercent > 90 {
			return "warning"
		}
	}

	for _, svc := range report.Services {
		if svc.Status == "error" {
			return "warning"
		}
	}

	return "online"
}

// parseID - 解析 ID
func parseID(s string) int64 {
	var id int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			id = id*10 + int64(c-'0')
		}
	}
	return id
}

// parseTimeRange - 解析时间范围参数
func parseTimeRange(s string) time.Time {
	switch s {
	case "6h":
		return time.Now().Add(-6 * time.Hour)
	case "24h":
		return time.Now().Add(-24 * time.Hour)
	case "7d":
		return time.Now().AddDate(0, 0, -7)
	case "30d":
		return time.Now().AddDate(0, 0, -30)
	default:
		return time.Now().Add(-1 * time.Hour)
	}
}
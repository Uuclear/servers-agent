package storage

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/slouch/vps-monitor/internal/models"
	"github.com/slouch/vps-monitor/pkg/protocol"
)

// Storage - SQLite 存储层
type Storage struct {
	db *sql.DB
}

// New - 创建存储实例
func New(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	s := &Storage{db: db}
	if err := s.initSchema(); err != nil {
		return nil, err
	}

	return s, nil
}

// initSchema - 初始化数据库表
func (s *Storage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS servers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		token TEXT UNIQUE NOT NULL,
		name TEXT,
		hostname TEXT,
		os TEXT,
		arch TEXT,
		last_seen DATETIME,
		status TEXT DEFAULT 'offline',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server_id INTEGER NOT NULL,
		timestamp DATETIME NOT NULL,
		type TEXT NOT NULL,
		path TEXT,
		value REAL NOT NULL,
		detail TEXT,
		FOREIGN KEY (server_id) REFERENCES servers(id)
	);

	CREATE TABLE IF NOT EXISTS service_checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		target TEXT NOT NULL,
		status TEXT NOT NULL,
		response_ms INTEGER,
		message TEXT,
		timestamp DATETIME NOT NULL,
		FOREIGN KEY (server_id) REFERENCES servers(id)
	);

	CREATE INDEX IF NOT EXISTS idx_metrics_server_time ON metrics(server_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_metrics_type ON metrics(type);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Close - 关闭数据库连接
func (s *Storage) Close() error {
	return s.db.Close()
}

// RegisterServer - 注册或更新服务器
func (s *Storage) RegisterServer(token, hostname, os, arch string) (*models.Server, error) {
	now := time.Now()

	var server models.Server
	var nameNull sql.NullString
	var hostnameNull, osNull, archNull sql.NullString

	err := s.db.QueryRow("SELECT id, token, name, hostname, os, arch, last_seen, status, created_at FROM servers WHERE token = ?", token).Scan(
		&server.ID, &server.Token, &nameNull, &hostnameNull, &osNull, &archNull, &server.LastSeen, &server.Status, &server.CreatedAt)

	if err == sql.ErrNoRows {
		// 新服务器
		result, err := s.db.Exec(`
			INSERT INTO servers (token, hostname, os, arch, last_seen, status, created_at)
			VALUES (?, ?, ?, ?, ?, 'online', ?)
		`, token, hostname, os, arch, now, now)
		if err != nil {
			return nil, err
		}

		id, _ := result.LastInsertId()
		server = models.Server{
			ID:        id,
			Token:     token,
			Hostname:  hostname,
			OS:        os,
			Arch:      arch,
			LastSeen:  now,
			Status:    "online",
			CreatedAt: now,
		}
	} else if err != nil {
		return nil, err
	} else {
		// 已存在，更新
		_, err = s.db.Exec("UPDATE servers SET hostname = ?, os = ?, arch = ?, last_seen = ?, status = 'online' WHERE token = ?",
			hostname, os, arch, now, token)
		if err != nil {
			return nil, err
		}
		server.Name = nameNull.String
		server.Hostname = hostnameNull.String
		server.OS = osNull.String
		server.Arch = archNull.String
		server.LastSeen = now
		server.Status = "online"
	}

	return &server, nil
}

// SaveMetrics - 保存指标数据
func (s *Storage) SaveMetrics(serverID int64, report *protocol.MetricsReport) error {
	now := time.Unix(report.Timestamp, 0)

	// CPU
	_, err := s.db.Exec(`
		INSERT INTO metrics (server_id, timestamp, type, value)
		VALUES (?, ?, 'cpu', ?)
	`, serverID, now, report.CPU.UsagePercent)
	if err != nil {
		return err
	}

	// Memory
	_, err = s.db.Exec(`
		INSERT INTO metrics (server_id, timestamp, type, value)
		VALUES (?, ?, 'memory', ?)
	`, serverID, now, report.Memory.UsedPercent)
	if err != nil {
		return err
	}

	// Swap
	_, err = s.db.Exec(`
		INSERT INTO metrics (server_id, timestamp, type, value)
		VALUES (?, ?, 'swap', ?)
	`, serverID, now, report.Swap.UsedPercent)
	if err != nil {
		return err
	}

	// Disk - 每个分区单独存储
	for _, d := range report.Disk {
		_, err = s.db.Exec(`
			INSERT INTO metrics (server_id, timestamp, type, path, value)
			VALUES (?, ?, 'disk', ?, ?)
		`, serverID, now, d.Path, d.UsedPercent)
		if err != nil {
			return err
		}
	}

	// Network 发送/接收累计值
	_, err = s.db.Exec(`
		INSERT INTO metrics (server_id, timestamp, type, value)
		VALUES (?, ?, 'network_sent', ?)
	`, serverID, now, float64(report.Network.BytesSent))
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO metrics (server_id, timestamp, type, value)
		VALUES (?, ?, 'network_recv', ?)
	`, serverID, now, float64(report.Network.BytesRecv))
	if err != nil {
		return err
	}

	// 系统负载
	_, err = s.db.Exec(`
		INSERT INTO metrics (server_id, timestamp, type, value)
		VALUES (?, ?, 'load1', ?)
	`, serverID, now, report.System.Load1)
	if err != nil {
		return err
	}

	return nil
}

// SaveServiceChecks - 保存服务检测结果
func (s *Storage) SaveServiceChecks(serverID int64, services []protocol.Service) error {
	now := time.Now()

	for _, svc := range services {
		_, err := s.db.Exec(`
			INSERT INTO service_checks (server_id, name, type, target, status, response_ms, message, timestamp)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, serverID, svc.Name, svc.Type, svc.Target, svc.Status, svc.ResponseMs, svc.Message, now)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetServers - 获取所有服务器列表
func (s *Storage) GetServers() ([]models.Server, error) {
	rows, err := s.db.Query("SELECT id, token, name, hostname, os, arch, last_seen, status, created_at FROM servers ORDER BY hostname")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []models.Server
	for rows.Next() {
		var server models.Server
		var nameNull sql.NullString
		var hostnameNull, osNull, archNull sql.NullString
		if err := rows.Scan(&server.ID, &server.Token, &nameNull, &hostnameNull, &osNull, &archNull, &server.LastSeen, &server.Status, &server.CreatedAt); err != nil {
			return nil, err
		}
		server.Name = nameNull.String
		server.Hostname = hostnameNull.String
		server.OS = osNull.String
		server.Arch = archNull.String
		servers = append(servers, server)
	}

	return servers, nil
}

// GetServerByID - 获取单个服务器
func (s *Storage) GetServerByID(id int64) (*models.Server, error) {
	var server models.Server
	var nameNull sql.NullString
	var hostnameNull, osNull, archNull sql.NullString
	err := s.db.QueryRow("SELECT id, token, name, hostname, os, arch, last_seen, status, created_at FROM servers WHERE id = ?", id).Scan(
		&server.ID, &server.Token, &nameNull, &hostnameNull, &osNull, &archNull, &server.LastSeen, &server.Status, &server.CreatedAt)
	if err != nil {
		return nil, err
	}
	server.Name = nameNull.String
	server.Hostname = hostnameNull.String
	server.OS = osNull.String
	server.Arch = archNull.String
	return &server, nil
}

// GetMetricsHistory - 获取指标历史
func (s *Storage) GetMetricsHistory(serverID int64, metricType string, since time.Time) ([]models.MetricPoint, error) {
	rows, err := s.db.Query(`
		SELECT timestamp, value FROM metrics
		WHERE server_id = ? AND type = ? AND timestamp >= ?
		ORDER BY timestamp ASC
	`, serverID, metricType, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []models.MetricPoint
	for rows.Next() {
		var p models.MetricPoint
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			return nil, err
		}
		points = append(points, p)
	}

	return points, nil
}

// GetDashboardSummary - 获取 Dashboard 概览
func (s *Storage) GetDashboardSummary() (*models.DashboardSummary, error) {
	var summary models.DashboardSummary

	err := s.db.QueryRow("SELECT COUNT(*) FROM servers").Scan(&summary.TotalServers)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM servers WHERE status = 'online'").Scan(&summary.OnlineCount)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM servers WHERE status = 'offline'").Scan(&summary.OfflineCount)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM servers WHERE status = 'warning'").Scan(&summary.WarningCount)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

// UpdateServerStatus - 更新服务器状态
func (s *Storage) UpdateServerStatus(serverID int64, status string) error {
	_, err := s.db.Exec("UPDATE servers SET status = ? WHERE id = ?", status, serverID)
	return err
}

// CleanOldMetrics - 清理过期数据
func (s *Storage) CleanOldMetrics(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	_, err := s.db.Exec("DELETE FROM metrics WHERE timestamp < ?", cutoff)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM service_checks WHERE timestamp < ?", cutoff)
	return err
}

// SetServerName - 设置服务器显示名称
func (s *Storage) SetServerName(serverID int64, name string) error {
	_, err := s.db.Exec("UPDATE servers SET name = ? WHERE id = ?", name, serverID)
	return err
}

// GetDiskPaths - 获取服务器所有磁盘分区路径
func (s *Storage) GetDiskPaths(serverID int64) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT path FROM metrics
		WHERE server_id = ? AND type = 'disk' AND path IS NOT NULL
		ORDER BY path
	`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

// GetDiskHistory - 获取特定分区的历史数据
func (s *Storage) GetDiskHistory(serverID int64, path string, since time.Time) ([]models.MetricPoint, error) {
	rows, err := s.db.Query(`
		SELECT timestamp, value FROM metrics
		WHERE server_id = ? AND type = 'disk' AND path = ? AND timestamp >= ?
		ORDER BY timestamp ASC
	`, serverID, path, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []models.MetricPoint
	for rows.Next() {
		var p models.MetricPoint
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, nil
}

// GetAllDiskHistory - 获取所有分区的历史数据（按分区分组）
func (s *Storage) GetAllDiskHistory(serverID int64, since time.Time) (map[string][]models.MetricPoint, error) {
	paths, err := s.GetDiskPaths(serverID)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]models.MetricPoint)
	for _, path := range paths {
		history, err := s.GetDiskHistory(serverID, path, since)
		if err != nil {
			continue
		}
		result[path] = history
	}
	return result, nil
}

// GetLatestMetrics - 获取服务器最新指标
func (s *Storage) GetLatestMetrics(serverID int64) (map[string]float64, error) {
	metrics := make(map[string]float64)

	types := []string{"cpu", "memory", "swap", "load1"}
	for _, t := range types {
		row := s.db.QueryRow(`
			SELECT value FROM metrics
			WHERE server_id = ? AND type = ?
			ORDER BY timestamp DESC LIMIT 1
		`, serverID, t)
		var v float64
		if row.Scan(&v) == nil {
			metrics[t] = v
		}
	}

	// 磁盘（取第一个分区）
	row := s.db.QueryRow(`
		SELECT value FROM metrics
		WHERE server_id = ? AND type = 'disk'
		ORDER BY timestamp DESC LIMIT 1
	`, serverID)
	var disk float64
	if row.Scan(&disk) == nil {
		metrics["disk"] = disk
	}

	return metrics, nil
}

// GetNetworkRate - 计算网络速率（需要两次采样差值）
func (s *Storage) GetNetworkRate(serverID int64) (sentRate, recvRate float64, err error) {
	// 获取最近两次采样
	rows, err := s.db.Query(`
		SELECT timestamp, type, value FROM metrics
		WHERE server_id = ? AND type IN ('network_sent', 'network_recv')
		ORDER BY timestamp DESC LIMIT 4
	`, serverID)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var timestamps []time.Time
	var sentValues []float64
	var recvValues []float64

	for rows.Next() {
		var ts time.Time
		var t string
		var v float64
		if err := rows.Scan(&ts, &t, &v); err != nil {
			continue
		}
		timestamps = append(timestamps, ts)
		if t == "network_sent" {
			sentValues = append(sentValues, v)
		} else {
			recvValues = append(recvValues, v)
		}
	}

	// 计算速率（bytes/second）
	if len(sentValues) >= 2 && len(timestamps) >= 2 {
		delta := timestamps[0].Sub(timestamps[1]).Seconds()
		if delta > 0 {
			sentRate = (sentValues[0] - sentValues[1]) / delta
		}
	}
	if len(recvValues) >= 2 && len(timestamps) >= 2 {
		delta := timestamps[0].Sub(timestamps[1]).Seconds()
		if delta > 0 {
			recvRate = (recvValues[0] - recvValues[1]) / delta
		}
	}

	return sentRate, recvRate, nil
}

// GetServiceChecksLatest - 获取最新服务检测结果
func (s *Storage) GetServiceChecksLatest(serverID int64) ([]models.ServiceCheck, error) {
	rows, err := s.db.Query(`
		SELECT name, type, target, status, response_ms, message, timestamp FROM service_checks
		WHERE server_id = ?
		ORDER BY timestamp DESC
	`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.ServiceCheck
	seen := make(map[string]bool)
	for rows.Next() {
		var c models.ServiceCheck
		if err := rows.Scan(&c.Name, &c.Type, &c.Target, &c.Status, &c.ResponseMs, &c.Message, &c.Timestamp); err != nil {
			continue
		}
		if !seen[c.Name] {
			seen[c.Name] = true
			checks = append(checks, c)
		}
	}
	return checks, nil
}
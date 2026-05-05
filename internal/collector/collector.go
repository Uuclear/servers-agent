package collector

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/slouch/vps-monitor/pkg/protocol"
)

// Collector - 系统指标采集器
type Collector struct {
	services  []ServiceCheckConfig // 服务可用性检测配置
	diskPaths []string             // 要监控的磁盘分区路径
}

// ServiceCheckConfig - 服务检测配置
type ServiceCheckConfig struct {
	Name   string
	Type   string // http/tcp/ssl
	Target string
}

// NewCollector - 创建采集器
func NewCollector(services []ServiceCheckConfig, diskPaths []string) *Collector {
	if len(diskPaths) == 0 {
		diskPaths = []string{"/"}
	}
	return &Collector{
		services:  services,
		diskPaths: diskPaths,
	}
}

// Collect - 采集所有指标
func (c *Collector) Collect() (*protocol.MetricsReport, error) {
	report := &protocol.MetricsReport{
		Timestamp: time.Now().Unix(),
	}

	// 获取主机信息
	hostInfo, err := host.Info()
	if err == nil {
		report.Hostname = hostInfo.Hostname
		report.System.OS = hostInfo.OS
		report.System.Arch = hostInfo.KernelArch
		report.System.Uptime = hostInfo.Uptime
	}

	// 获取系统负载
	loadAvg, err := load.Avg()
	if err == nil {
		report.System.Load1 = loadAvg.Load1
		report.System.Load5 = loadAvg.Load5
		report.System.Load15 = loadAvg.Load15
	}

	// CPU 使用率
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		report.CPU.UsagePercent = cpuPercent[0]
	}

	// CPU 时间分布
	cpuTimes, err := cpu.Times(false)
	if err == nil && len(cpuTimes) > 0 {
		t := cpuTimes[0]
		total := t.User + t.System + t.Idle + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal + t.Guest
		if total > 0 {
			report.CPU.UserPercent = t.User / total * 100
			report.CPU.SystemPercent = t.System / total * 100
			report.CPU.IdlePercent = t.Idle / total * 100
			report.CPU.IowaitPercent = t.Iowait / total * 100
		}
	}

	// CPU 核数
	cpuCounts, err := cpu.Counts(true)
	if err == nil {
		report.CPU.CoreCount = cpuCounts
	}

	// 内存
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		report.Memory.Total = memInfo.Total
		report.Memory.Used = memInfo.Used
		report.Memory.Available = memInfo.Available
		report.Memory.UsedPercent = memInfo.UsedPercent
		report.Memory.Buffers = memInfo.Buffers
		report.Memory.Cached = memInfo.Cached
	}

	// Swap
	swapInfo, err := mem.SwapMemory()
	if err == nil {
		report.Swap.Total = swapInfo.Total
		report.Swap.Used = swapInfo.Used
		report.Swap.Free = swapInfo.Free
		report.Swap.UsedPercent = swapInfo.UsedPercent
	}

	// Disk - 采集配置的分区
	for _, path := range c.diskPaths {
		diskUsage, err := disk.Usage(path)
		if err == nil {
			report.Disk = append(report.Disk, protocol.Disk{
				Path:        path,
				Total:       diskUsage.Total,
				Used:        diskUsage.Used,
				Free:        diskUsage.Free,
				UsedPercent: diskUsage.UsedPercent,
				Fstype:      diskUsage.Fstype,
			})
		}
	}

	// 自动采集所有挂载分区（排除已采集的）
	partitions, err := disk.Partitions(false)
	if err == nil {
		for _, p := range partitions {
			found := false
			for _, existing := range report.Disk {
				if existing.Path == p.Mountpoint {
					found = true
					break
				}
			}
			if !found && p.Mountpoint != "" {
				usage, err := disk.Usage(p.Mountpoint)
				if err == nil && usage.Total > 0 {
					report.Disk = append(report.Disk, protocol.Disk{
						Path:        p.Mountpoint,
						Total:       usage.Total,
						Used:        usage.Used,
						Free:        usage.Free,
						UsedPercent: usage.UsedPercent,
						Fstype:      usage.Fstype,
					})
				}
			}
		}
	}

	// Network
	netIO, err := psnet.IOCounters(false)
	if err == nil && len(netIO) > 0 {
		report.Network.BytesSent = netIO[0].BytesSent
		report.Network.BytesRecv = netIO[0].BytesRecv
		report.Network.PacketsSent = netIO[0].PacketsSent
		report.Network.PacketsRecv = netIO[0].PacketsRecv
		report.Network.ErrIn = netIO[0].Errin
		report.Network.ErrOut = netIO[0].Errout
		report.Network.DropIn = netIO[0].Dropin
		report.Network.DropOut = netIO[0].Dropout
	}

	// 服务可用性检测
	if len(c.services) > 0 {
		report.Services = c.checkServices()
	}

	return report, nil
}

// checkServices - 检测服务可用性
func (c *Collector) checkServices() []protocol.Service {
	var results []protocol.Service

	for _, svc := range c.services {
		result := protocol.Service{
			Name:   svc.Name,
			Type:   svc.Type,
			Target: svc.Target,
		}

		switch svc.Type {
		case "http":
			start := time.Now()
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Get(svc.Target)
			result.ResponseMs = time.Since(start).Milliseconds()

			if err != nil {
				result.Status = "error"
				result.Message = err.Error()
			} else {
				resp.Body.Close()
				if resp.StatusCode >= 400 {
					result.Status = "error"
					result.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
				} else {
					result.Status = "ok"
				}
			}

		case "tcp":
			start := time.Now()
			conn, err := net.DialTimeout("tcp", svc.Target, 5*time.Second)
			result.ResponseMs = time.Since(start).Milliseconds()

			if err != nil {
				result.Status = "error"
				result.Message = err.Error()
			} else {
				result.Status = "ok"
				conn.Close()
			}

		case "ssl":
			start := time.Now()
			conn, err := tls.DialWithDialer(
				&net.Dialer{Timeout: 5 * time.Second},
				"tcp",
				svc.Target,
				&tls.Config{InsecureSkipVerify: false},
			)
			result.ResponseMs = time.Since(start).Milliseconds()

			if err != nil {
				result.Status = "error"
				result.Message = err.Error()
			} else {
				state := conn.ConnectionState()
				certs := state.PeerCertificates
				if len(certs) > 0 {
					expiry := certs[0].NotAfter
					daysLeft := int(time.Until(expiry).Hours() / 24)
					if daysLeft < 0 {
						result.Status = "error"
						result.Message = "证书已过期"
					} else if daysLeft < 7 {
						result.Status = "warning"
						result.Message = fmt.Sprintf("证书 %d 天后过期", daysLeft)
					} else {
						result.Status = "ok"
						result.Message = fmt.Sprintf("有效期 %d 天", daysLeft)
					}
				} else {
					result.Status = "ok"
				}
				conn.Close()
			}
		}

		results = append(results, result)
	}

	return results
}
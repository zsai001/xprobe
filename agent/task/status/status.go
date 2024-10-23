package status

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"xprobe_agent/log"

	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type ServerDynamicData struct {
	ID              string     `json:"id"`
	Load            [3]float64 `json:"load"`
	CPUUsage        float64    `json:"cpuUsage"`
	MemoryUsed      uint64     `json:"memoryUsed"`
	DiskUsed        uint64     `json:"diskUsed"`
	NetworkDownload uint64     `json:"networkDownload"`
	NetworkUpload   uint64     `json:"networkUpload"`
	TrafficDownload uint64     `json:"trafficDownload"`
	TrafficUpload   uint64     `json:"trafficUpload"`
	TCPCount        int        `json:"tcpCount"`
	UDPCount        int        `json:"udpCount"`
	ProcessCount    int        `json:"processCount"`
	ThreadCount     int        `json:"threadCount"`
	Actions         []string   `json:"actions"`
}

type NetworkSpeed struct {
	Download uint64
	Upload   uint64
}

type StatusTask struct {
	NodeID       string
	currentSpeed NetworkSpeed
	speedMutex   sync.RWMutex
	Config       *StatusConfig
	Data         []ServerDynamicData
}

func NewStatusTask(nodeID string) *StatusTask {
	task := &StatusTask{
		NodeID: nodeID,
	}
	go task.maintainNetworkSpeed()
	return task
}

func (t *StatusTask) GetData() []byte {
	data, _ := json.Marshal(t.Data)
	t.Data = t.Data[:]
	return data
}

type StatusConfig struct {
	NodeID string
}

func (t *StatusTask) SetConfig(cfg string) string {
	t.Config = &StatusConfig{}
	json.Unmarshal([]byte(cfg), t.Config)
	return ""
}

func (t *StatusTask) Execute() error {
	data := ServerDynamicData{}
	data.ID = t.NodeID
	loadAvg, err := load.Avg()
	if err != nil {
		return err
	}
	data.Load = [3]float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15}
	cpuUsage, err := t.getCPUUsage()
	if err != nil {
		return err
	}
	data.CPUUsage = cpuUsage

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	data.MemoryUsed = memInfo.Used
	diskInfo, err := t.getDiskInfo()
	if err != nil {
		return err
	}
	data.DiskUsed = diskInfo.Used

	networkInfo, err := t.getNetworkInfo()
	if err != nil {
		return err
	}
	data.TrafficDownload = networkInfo.BytesRecv
	data.TrafficUpload = networkInfo.BytesSent

	tcpCount, udpCount, err := t.getConnectionCounts()
	if err != nil {
		return err
	}
	data.TCPCount = tcpCount
	data.UDPCount = udpCount

	processCount, threadCount, err := t.getProcessAndThreadCounts()
	if err != nil {
		return err
	}
	data.ProcessCount = processCount
	data.ThreadCount = threadCount

	t.speedMutex.RLock()
	speed := t.currentSpeed
	t.speedMutex.RUnlock()
	data.NetworkDownload = speed.Download
	data.NetworkUpload = speed.Upload

	t.Data = append(t.Data, data)
	if len(t.Data) > 10 {
		t.Data = t.Data[:10]
	}
	log.Info("run status with", data)
	return nil
}

func (t *StatusTask) maintainNetworkSpeed() {
	for {
		downloadSpeed, uploadSpeed, err := t.getCurrentNetworkSpeed()
		if err == nil {
			t.speedMutex.Lock()
			t.currentSpeed = NetworkSpeed{
				Download: downloadSpeed,
				Upload:   uploadSpeed,
			}
			t.speedMutex.Unlock()
		}
		time.Sleep(time.Second)
	}
}

func (t *StatusTask) getCurrentNetworkSpeed() (uint64, uint64, error) {
	initialStats, err := psnet.IOCounters(false)
	if err != nil {
		return 0, 0, err
	}

	time.Sleep(time.Second)

	finalStats, err := psnet.IOCounters(false)
	if err != nil {
		return 0, 0, err
	}

	downloadSpeed := finalStats[0].BytesRecv - initialStats[0].BytesRecv
	uploadSpeed := finalStats[0].BytesSent - initialStats[0].BytesSent

	return downloadSpeed, uploadSpeed, nil
}

func (t *StatusTask) getCPUUsage() (float64, error) {
	percentage, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, err
	}
	if len(percentage) > 0 {
		return percentage[0], nil
	}
	return 0, fmt.Errorf("no CPU usage data available")
}

func (t *StatusTask) getDiskInfo() (struct{ Used, Total uint64 }, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return struct{ Used, Total uint64 }{}, err
	}

	var totalUsed, totalSpace uint64

	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}
		totalUsed += usage.Used
		totalSpace += usage.Total
	}

	return struct{ Used, Total uint64 }{totalUsed, totalSpace}, nil
}

func (t *StatusTask) getNetworkInfo() (psnet.IOCountersStat, error) {
	ioCounters, err := psnet.IOCounters(false)
	if err != nil {
		return psnet.IOCountersStat{}, err
	}

	if len(ioCounters) > 0 {
		return ioCounters[0], nil
	}

	return psnet.IOCountersStat{}, fmt.Errorf("no network data available")
}

const (
	TCP uint32 = 1
	UDP uint32 = 2
)

func (t *StatusTask) getConnectionCounts() (int, int, error) {
	conns, err := psnet.Connections("all")
	if err != nil {
		return 0, 0, err
	}

	tcpCount := 0
	udpCount := 0
	for _, conn := range conns {
		switch conn.Type {
		case TCP:
			tcpCount++
		case UDP:
			udpCount++
		}
	}

	return tcpCount, udpCount, nil
}

func (t *StatusTask) getProcessAndThreadCounts() (int, int, error) {
	processes, err := process.Processes()
	if err != nil {
		return 0, 0, err
	}

	processCount := len(processes)
	threadCount := 0

	for _, p := range processes {
		numThreads, err := p.NumThreads()
		if err == nil {
			threadCount += int(numThreads)
		}
	}

	return processCount, threadCount, nil
}

package ping

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	ping2 "github.com/go-ping/ping"
)

type PingConfig struct {
	Nodes   []PingNode `json:"nodes"`
	Version string     `json:"version"`
}

type PingNode struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	UseTCP   bool   `json:"useTCP"`
	Interval int    `json:"interval"` // 以秒为单位
}

type PingResult struct {
	NodeName  string    `json:"node_name"`
	Address   string    `json:"address"`
	Latency   float64   `json:"latency"`
	Timestamp time.Time `json:"timestamp"`
}

type PingManager struct {
	config     PingConfig
	configLock sync.RWMutex
	stopChan   chan struct{}
	once       sync.Once
	Data       []PingResult
}

type PingTaskData struct {
	Data    []PingResult
	Version string
}

type PingTask struct {
	Data   PingTaskData
	Config PingConfig
}

func (t *PingTask) Execute() error {
	t.Data.Data = m.Data
	t.Data.Version = t.Config.Version
	m.Data = []PingResult{}
	return nil
}

func (t *PingTask) GetData() interface{} {
	data := t.Data
	t.Data.Data = t.Data.Data[:]
	return data
}

func (t *PingTask) SetConfig(cfg string) string {
	// panic("ping task set config")
	t.Config = PingConfig{}
	json.Unmarshal([]byte(cfg), &t.Config)
	m.UpdateConfig(t.Config)
	return ""
}

var m = newPingManager()

func newPingManager() *PingManager {
	return &PingManager{
		stopChan: make(chan struct{}),
	}
}

func (pm *PingManager) Start() {
	pm.once.Do(func() {
		go pm.run()
	})
}

func (pm *PingManager) Stop() {
	close(pm.stopChan)
}

func (pm *PingManager) UpdateConfig(newConfig PingConfig) error {
	pm.configLock.Lock()
	pm.config = newConfig
	pm.configLock.Unlock()
	pm.Start()
	return nil
}

func (pm *PingManager) run() {
	nodeTickers := make(map[string]*time.Ticker)

	for {
		select {
		case <-time.After(5 * time.Second):
			pm.configLock.RLock()
			nodes := pm.config.Nodes
			pm.configLock.RUnlock()

			// 停止不再需要的ticker
			for name, ticker := range nodeTickers {
				found := false
				for _, node := range nodes {
					if node.Name == name {
						found = true
						break
					}
				}
				if !found {
					ticker.Stop()
					delete(nodeTickers, name)
				}
			}

			// 为新节点创建ticker或更新现有的
			for _, node := range nodes {
				if _, exists := nodeTickers[node.Name]; !exists {
					nodeTickers[node.Name] = time.NewTicker(time.Duration(node.Interval) * time.Second)
					go pm.pingNode(node, nodeTickers[node.Name].C)
				}
			}

		case <-pm.stopChan:
			for _, ticker := range nodeTickers {
				ticker.Stop()
			}
			return
		}
	}
}

func (pm *PingManager) pingNode(node PingNode, ticker <-chan time.Time) {
	for {
		select {
		case <-ticker:
			result, err := pm.pingHost(node)
			if err != nil {
				log.Printf("Error pinging %s (%s): %v", node.Name, node.Address, err)
				continue
			}
			pm.Data = append(pm.Data, result)
		case <-pm.stopChan:
			return
		}
	}
}

func (pm *PingManager) pingHost(node PingNode) (PingResult, error) {
	if node.UseTCP && false {
		return pm.tcpPing(node)
	}
	return pm.icmpPing(node)
}

func (pm *PingManager) icmpPing(node PingNode) (PingResult, error) {
	pinger, err := ping2.NewPinger(node.Address)
	if err != nil {
		return PingResult{}, fmt.Errorf("failed to create pinger: %v", err)
	}

	pinger.Count = 1
	pinger.Timeout = time.Second * 2

	err = pinger.Run()
	if err != nil {
		return PingResult{}, fmt.Errorf("ping failed: %v", err)
	}

	stats := pinger.Statistics()

	return PingResult{
		NodeName:  node.Name,
		Address:   node.Address,
		Latency:   float64(stats.AvgRtt) / float64(time.Millisecond),
		Timestamp: time.Now(),
	}, nil
}

func (pm *PingManager) tcpPing(node PingNode) (PingResult, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", node.Address+":80", time.Second*2)
	if err != nil {
		return PingResult{NodeName: node.Name, Address: node.Address, Latency: 0, Timestamp: time.Now()}, err
	}
	defer conn.Close()

	latency := time.Since(start).Seconds() * 1000 // 转换为毫秒

	// ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	return PingResult{
		NodeName:  node.Name,
		Address:   node.Address,
		Latency:   latency,
		Timestamp: time.Now(),
	}, nil
}

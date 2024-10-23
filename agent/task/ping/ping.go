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
	Nodes []PingNode `json:"nodes"`
}

type PingNode struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	UseTCP   bool   `json:"useTCP"`
	Interval int    `json:"interval"` // 以秒为单位
}

type PingResult struct {
	NodeName  string
	IP        string
	Latency   float64
	Timestamp time.Time
}

type PingManager struct {
	config     PingConfig
	configLock sync.RWMutex
	stopChan   chan struct{}
	reportFunc func(PingResult)
}

type PingTask struct {
	Data   [][]PingResult
	Config *PingConfig
}

func (t *PingTask) Execute() error {

	return nil
}

func (t *PingTask) GetData() []byte {
	data, _ := json.Marshal(t.Data)
	t.Data = t.Data[:]
	return data
}

func (t *PingTask) SetConfig(cfg string) string {
	t.Config = &PingConfig{}
	json.Unmarshal([]byte(cfg), t.Config)
	return ""
}

func NewPingManager(configJSON []byte, reportFunc func(PingResult)) (*PingManager, error) {
	var config PingConfig
	err := json.Unmarshal(configJSON, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	return &PingManager{
		config:     config,
		stopChan:   make(chan struct{}),
		reportFunc: reportFunc,
	}, nil
}

func (pm *PingManager) Start() {
	go pm.run()
}

func (pm *PingManager) Stop() {
	close(pm.stopChan)
}

func (pm *PingManager) UpdateConfig(configJSON []byte) error {
	var newConfig PingConfig
	err := json.Unmarshal(configJSON, &newConfig)
	if err != nil {
		return fmt.Errorf("failed to parse new config: %v", err)
	}

	pm.configLock.Lock()
	pm.config = newConfig
	pm.configLock.Unlock()

	return nil
}

func (pm *PingManager) run() {
	nodeTickers := make(map[string]*time.Ticker)

	for {
		select {
		case <-time.After(1 * time.Second):
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
			pm.reportFunc(result)
		case <-pm.stopChan:
			return
		}
	}
}

func (pm *PingManager) pingHost(node PingNode) (PingResult, error) {
	if node.UseTCP {
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
		IP:        stats.IPAddr.String(),
		Latency:   float64(stats.AvgRtt) / float64(time.Millisecond),
		Timestamp: time.Now(),
	}, nil
}

func (pm *PingManager) tcpPing(node PingNode) (PingResult, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", node.Address+":80", time.Second*2)
	if err != nil {
		return PingResult{}, err
	}
	defer conn.Close()

	latency := time.Since(start).Seconds() * 1000 // 转换为毫秒

	ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	return PingResult{
		NodeName:  node.Name,
		IP:        ip,
		Latency:   latency,
		Timestamp: time.Now(),
	}, nil
}

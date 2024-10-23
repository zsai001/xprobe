package action

import (
	"fmt"
	"sync"
	"xprobe_agent/action/bench"
	"xprobe_agent/action/config"
	"xprobe_agent/action/install"
	"xprobe_agent/action/upgrade"
	"xprobe_agent/log"
)

// Action 代表一个可执行的动作
type Action interface {
	Execute(string, interface{}) error
}

// ActionConfig 代表动作的配置
type ActionConfig struct {
	Enabled bool           `json:"enabled"`
	Params  map[string]any `json:"params"`
}

// Manager 是动作管理器
type Manager struct {
	actions map[string]Action
	mu      sync.RWMutex
}

// NewManager 创建一个新的动作管理器
func NewManager() *Manager {
	return &Manager{
		actions: make(map[string]Action),
	}
}

// Register 注册一个新动作
func (m *Manager) Register(name string, action Action) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.actions[name] = action
}

// Execute 执行指定名称的动作
func (m *Manager) Execute(name string, topic string, data interface{}) error {
	m.mu.RLock()
	action, exists := m.actions[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("动作 '%s' 未找到", name)
	}

	return action.Execute(topic, data)
}

type ActionItem struct {
	Name  string      `json:"name"`
	Topic string      `json:"topic"`
	Data  interface{} `json:"data"`
}

var m *Manager

func init() {
	m = NewManager()
	m.Register("config", &config.ConfigAction{})
	m.Register("upgrade", &upgrade.UpgradeAction{})
	m.Register("bench", &bench.BenchmarkAction{})
	m.Register("install", &install.InstallAction{})
}

func TakeAction(action []ActionItem) {
	for _, item := range action {
		log.Infof("take action: %v", item)
		m.Execute(item.Name, item.Topic, item.Data)
	}
}

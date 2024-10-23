package action

import (
	"fmt"
	"sync"
)

// Action 代表一个可执行的动作
type Action interface {
	Execute() error
	GetResult() string
	SetConfig(string) error
}

// ActionConfig 代表动作的配置
type ActionConfig struct {
	Enabled bool           `json:"enabled"`
	Params  map[string]any `json:"params"`
}

// Manager 是动作管理器
type Manager struct {
	actions map[string]Action
	configs map[string]ActionConfig
	mu      sync.RWMutex
}

// NewManager 创建一个新的动作管理器
func NewManager() *Manager {
	return &Manager{
		actions: make(map[string]Action),
		configs: make(map[string]ActionConfig),
	}
}

// Register 注册一个新动作
func (m *Manager) Register(name string, action Action) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.actions[name] = action
}

// Execute 执行指定名称的动作
func (m *Manager) Execute(name string) error {
	m.mu.RLock()
	action, exists := m.actions[name]
	config := m.configs[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("动作 '%s' 未找到", name)
	}

	if !config.Enabled {
		return fmt.Errorf("动作 '%s' 未启用", name)
	}

	return action.Execute()
}

// UpdateConfig 更新动作配置
func (m *Manager) UpdateConfig(name string, config ActionConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[name] = config
}

func TakeAction(action string) {

}

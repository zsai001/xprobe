package task

import (
	"fmt"
	"sync"
	"time"
	"xprobe_agent/config"
	"xprobe_agent/log"
	"xprobe_agent/task/host"
	"xprobe_agent/task/ping"
	"xprobe_agent/task/status"
	"xprobe_agent/util"
)

// Task 代表一个可执行的任务
type Task interface {
	Execute() error
	GetData() []byte
	SetConfig(string) string
}

// TaskConfig 代表任务的配置
type TaskConfig struct {
	Interval time.Duration `yaml:"interval"`
	Enabled  bool          `yaml:"enabled"`
}

// TaskConfigs 存储所有任务的配置
type TaskConfigs map[string]TaskConfig

// Manager 是任务管理器
type Manager struct {
	tasks   map[string]Task
	configs TaskConfigs
	mu      sync.RWMutex
}

// NewManager 创建一个新的任务管理器
func NewManager() *Manager {
	return &Manager{
		tasks:   make(map[string]Task),
		configs: make(TaskConfigs),
	}
}

// Register 注册一个新任务
func (m *Manager) Register(name string, task Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[name] = task
}

// Get 获取一个已注册的任务
func (m *Manager) Get(name string) (Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	task, exists := m.tasks[name]
	return task, exists
}

// Execute 执行指定名称的任务
func (m *Manager) Execute(name string) error {
	task, exists := m.Get(name)
	if !exists {
		return fmt.Errorf("任务 '%s' 未找到", name)
	}
	return task.Execute()
}

// UpdateConfig 更新任务配置
func (m *Manager) UpdateConfig(configs TaskConfigs) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs = configs
}

// GetConfig 获取任务配置
func (m *Manager) GetConfig(name string) (TaskConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	config, exists := m.configs[name]
	return config, exists
}

// RunTasks 运行所有已配置的任务
func (m *Manager) RunTasks() {
	//m.mu.RLock()
	//defer m.mu.RUnlock()
	var err error
	for name, task := range m.tasks {
		err = task.Execute()
		if err != nil {
			log.Errorf("run task %s with error %s", name, err.Error())
		} else {
			log.Infof("run task %s successfully", name)
		}
	}

	allData := make(map[string][]byte)
	for name, task := range m.tasks {
		data := task.GetData()
		if data == nil {
			continue
		}
		allData[name] = data
	}
	util.Report("/api/report", allData)
	log.Info("report with", "api/report", allData)
	time.Sleep(1 * time.Second)
	//for name, config := range m.configs {
	//	if config.Enabled {
	//		m.runTask(name, config.Interval)
	//	}
	//}
	//
	//for name, config := range m.configs {
	//	report := make(map[string][]byte)
	//	if config.Enabled {
	//		task, exists := m.Get(name)
	//		if !exists {
	//			continue
	//		}
	//		data := task.GetData()
	//		if data != nil {
	//			continue
	//		}
	//		report[name] = task.GetData()
	//	}
	//	util.Report("/api/report", report)
	//	time.Sleep(config.Interval)
	//}
}

func (m *Manager) Start() {
	for {
		m.RunTasks()
		time.Sleep(1 * time.Second)
	}
}

// runTask 运行单个任务
func (m *Manager) runTask(name string, interval time.Duration) {
	for {
		task, exists := m.Get(name)
		if !exists {
			log.Errorf("任务 '%s' 不存在\n", name)
			return
		}

		err := task.Execute()
		if err != nil {
			log.Errorf("执行任务 '%s' 失败: %v\n", name, err)
		}

		time.Sleep(interval)
	}
}

// InitTasks 初始化所有任务
func InitTasks(m *Manager, cfg *config.Config) {
	m.Register("host", &host.HostInfoTask{})
	m.Register("status", &status.StatusTask{})
	m.Register("ping", &ping.PingTask{})
}

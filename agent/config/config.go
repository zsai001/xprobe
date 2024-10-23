package config

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"time"
)

var (
	reportInterval = 1 * time.Second
	Host           = "http://127.0.0.1:8080"
	NodeId         = "default_test"
	version        = "1.0.0"
)

// getConfigPath 根据不同操作系统返回配置文件路径
func getConfigPath() string {
	return "./config.yaml"
	//switch runtime.GOOS {
	//case "windows":
	//	// Windows: %ProgramData%\XProbe\config.yaml
	//	programData := os.Getenv("ProgramData")
	//	if programData == "" {
	//		programData = filepath.Join(os.Getenv("SystemDrive")+"\\", "ProgramData")
	//	}
	//	return filepath.Join(programData, "XProbe", "config.yaml")
	//case "darwin":
	//	// macOS: /Library/Application Support/XProbe/config.yaml
	//	return "/Library/Application Support/XProbe/config.yaml"
	//default:
	//	// Linux: /etc/xprobe/config.yaml
	//	return "/etc/xprobe/config.yaml"
	//}
}

// loadConfig 加载配置文件
func LoadConfig(host, node string) (*Config, error) {
	configPath := getConfigPath()

	// 创建默认配置
	cfg := &Config{
		NodeID:  node,
		Host:    host,
		Version: version,
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 如果配置文件不存在，创建默认配置
		err = os.MkdirAll(filepath.Dir(configPath), 0755)
		if err != nil {
			return cfg, fmt.Errorf("创建配置目录失败: %v", err)
		}

		// 保存默认配置
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return cfg, fmt.Errorf("序列化默认配置失败: %v", err)
		}

		err = os.WriteFile(configPath, data, 0644)
		if err != nil {
			return cfg, fmt.Errorf("保存默认配置失败: %v", err)
		}

		return cfg, nil
	}

	// 读取现有配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg, fmt.Errorf("读取配置文件失败: %v", err)
	}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return cfg, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 命令行参数覆盖配置文件
	if flag.Lookup("host") != nil && flag.Lookup("host").Value.String() != "" {
		cfg.Host = flag.Lookup("host").Value.String()
	}
	if flag.Lookup("node") != nil && flag.Lookup("node").Value.String() != "" {
		cfg.NodeID = flag.Lookup("node").Value.String()
	}

	return cfg, nil
}

type Config struct {
	NodeID  string `yaml:"node_id"`
	Host    string `yaml:"host"`
	Version string `yaml:"version"`
}

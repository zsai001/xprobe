package install

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
)

type InstallAction struct {
	PackageName string
	Version     string
	Config      *InstallConfig
}

type InstallConfig struct {
	PackageName string `json:"packageName"`
	Version     string `json:"version"`
	Force       bool   `json:"force"`
}

func (a *InstallAction) Execute(topic string, data interface{}) error {
	switch runtime.GOOS {
	case "linux":
		return a.installOnLinux()
	case "darwin":
		return a.installOnMacOS()
	case "windows":
		return a.installOnWindows()
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

func (a *InstallAction) GetResult() string {
	return fmt.Sprintf("已安装 %s 版本 %s", a.PackageName, a.Version)
}

func (a *InstallAction) SetConfig(cfg string) error {
	a.Config = &InstallConfig{}
	if err := json.Unmarshal([]byte(cfg), a.Config); err != nil {
		return err
	}
	a.PackageName = a.Config.PackageName
	a.Version = a.Config.Version
	return nil
}

func (a *InstallAction) installOnLinux() error {
	cmd := exec.Command("apt-get", "install", "-y", a.PackageName)
	return cmd.Run()
}

func (a *InstallAction) installOnMacOS() error {
	cmd := exec.Command("brew", "install", a.PackageName)
	return cmd.Run()
}

func (a *InstallAction) installOnWindows() error {
	cmd := exec.Command("choco", "install", a.PackageName, "-y")
	return cmd.Run()
}

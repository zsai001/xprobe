package upgrade

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

type UpgradeTask struct {
	ServerURL string
	NodeID    string
	Version   string
	Config    *UpgradeConfig
}

type UpgradeConfig struct {
	NodeID string
}

func (t *UpgradeTask) Execute() error {
	latestVersion, err := t.checkLatestVersion()
	if err != nil {
		return err
	}

	if latestVersion == t.Version {
		fmt.Println("已经是最新版本")
		return nil
	}

	fmt.Printf("发现新版本: %s，正在升级...\n", latestVersion)

	err = t.downloadAndInstall(latestVersion)
	if err != nil {
		return err
	}

	fmt.Println("升级成功，正在重启服务...")
	return t.restartService()
}

func (t *UpgradeTask) checkLatestVersion() (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/version", t.ServerURL))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (t *UpgradeTask) downloadAndInstall(version string) error {
	url := fmt.Sprintf("%s/agent/%s/%s/xprobe_agent", t.ServerURL, runtime.GOOS, runtime.GOARCH)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tempFile, err := os.CreateTemp("", "xprobe_agent_*")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return err
	}

	err = tempFile.Close()
	if err != nil {
		return err
	}

	err = os.Chmod(tempFile.Name(), 0755)
	if err != nil {
		return err
	}

	return os.Rename(tempFile.Name(), os.Args[0])
}

func (t *UpgradeTask) restartService() error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("systemctl", "restart", "xprobe.service").Run()
	case "darwin":
		return exec.Command("launchctl", "stop", "com.xprobe.agent").Run()
	case "windows":
		return exec.Command("powershell", "-Command", "Restart-ScheduledTask -TaskName 'XProbe Agent'").Run()
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

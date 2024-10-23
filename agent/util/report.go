package util

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"xprobe_agent/action"
)

var (
	Host string
)

func SetHost(host string) {
	Host = host
}

type ReportResult struct {
	Action string
}

func Report(path string, data interface{}) error {
	url := fmt.Sprintf("%s/%s", Host, path)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}
	//parse response with action config
	var result ReportResult
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}
	fmt.Println(result)
	if result.Action != "" {
		action.TakeAction(result.Action)
	}
	return nil
}

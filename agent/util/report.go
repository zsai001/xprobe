package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"xprobe_agent/action"
	"xprobe_agent/config"
	"xprobe_agent/v"
)

type ReportResult struct {
	Actions []action.ActionItem `json:"actions"`
}

func Report(path string, data interface{}) error {
	cfg, _ := config.GetConfig()
	url2, err := url.JoinPath(cfg.Host, path)
	if err != nil {
		return err
	}
	// log.Infof("report to %s with %v", url2, data)
	//convert data to json
	send, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// log.Infof("report to %s with %s", url2, string(send))

	//post with header
	header := make(map[string]string)
	header["Content-Type"] = "application/json"
	header["X-Node-Id"] = cfg.NodeID
	header["X-Agent-Version"] = v.Version
	//add header to post
	client := &http.Client{}

	// Create a new request
	req, err := http.NewRequest("POST", url2, bytes.NewBuffer(send))
	if err != nil {
		return err
	}

	// Add headers to the request
	for key, value := range header {
		req.Header.Set(key, value)
	}

	// Send the request
	resp, err := client.Do(req)
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
	if result.Actions != nil {
		action.TakeAction(result.Actions)
	}
	return nil
}

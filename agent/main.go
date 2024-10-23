package main

import (
	"flag"
	"fmt"
	"xprobe_agent/config"
	"xprobe_agent/log"
	"xprobe_agent/task"
)

var (
	Host   string
	NodeId string
)

func main() {
	// 解析命令行参数
	flag.StringVar(&Host, "host", Host, "XProbe server host")
	flag.StringVar(&NodeId, "node", NodeId, "Node ID")
	flag.Parse()
	log.Init()

	// 加载配置
	cfg, err := config.LoadConfig(Host, NodeId)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		return
	}

	// 初始化任务管理器
	taskManager := task.NewManager()
	task.InitTasks(taskManager, cfg)

	// 运行所有已配置的任务
	go taskManager.Start()

	// 保持程序运行
	select {}
}

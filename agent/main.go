package main

import (
	"flag"
	"fmt"
	"time"
	"xprobe_agent/config"
	"xprobe_agent/log"
	"xprobe_agent/task"
)

var (
	reportInterval = 1 * time.Second
	Host           string
	NodeId         string
)

func main() {
	// 解析命令行参数
	flag.DurationVar(&reportInterval, "i", 1*time.Second, "Report interval")
	flag.Parse()

	args := flag.Args()
	if len(args) >= 1 {
		Host = args[0]
	}
	if len(args) >= 2 {
		NodeId = args[1]
	}

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

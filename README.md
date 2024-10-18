
# XProb 服务器探针

XProb 是一个高效、轻量级的多平台服务器探针程序,用于实时监控和管理多个服务器的性能指标。

## 主要特性

- 多平台支持: Linux、macOS、Windows
- 多架构支持: amd64、386、arm、arm64
- 低资源占用,高性能数据采集
- 实时监控: CPU、内存、磁盘、网络等系统指标
- 集中管理: 通过Web界面集中监控多台服务器
- 安全可靠: 加密通信,保护敏感数据
- 易于部署: 支持Docker一键部署
- 可扩展: 支持自定义监控指标和告警规则

## 系统要求

- 服务端:
  - Docker 和 Docker Compose
  - MongoDB (用于数据存储)
- 客户端(被监控服务器):
  - 支持 Linux、macOS 或 Windows 操作系统

## 快速开始

1. 克隆仓库:

   ```
   git clone https://github.com/zsai001/xprob.git
   cd xprob
   ```

2. 启动服务端:

   ```
   sh start.sh
   ```

3. 安装客户端:
   
   在需要监控的服务器上运行以下命令:

   ```
   curl -sSL https://your-xprob-server/install.sh | bash
   ```

4. 访问Web管理界面:

   打开浏览器,访问 `http://your-server-ip:8080`

## 项目结构

```
xprob/
├── main.go # 主程序入口
├── agent/ # 客户端探针代码
├── docker-compose.yml # Docker Compose 配置文件
├── Dockerfile # Docker 构建文件
├── setup.sh # 项目安装脚本
├── web/ # Web API
├── html/ # Web 界面
└── README.md # 项目说明文件
```

欢迎提交问题和拉取请求。对于重大更改,请先开issue讨论您想要改变的内容。

## 许可证

[MIT](https://choosealicense.com/licenses/mit/)
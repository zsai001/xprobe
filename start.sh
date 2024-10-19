#!/bin/bash

# 日志函数
log_error() {
    echo "错误: $1" >&2
}

# 检查命令执行状态
check_status() {
    if [ $? -ne 0 ]; then
        log_error "$1"
        exit 1
    fi
}

# 检查是否已安装Go
check_go() {
    if ! command -v go &>/dev/null; then
        echo "Go未安装,正在安装..."
        # 下载最新版本的Go
        wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
        check_status "下载Go失败"

        # 解压到 /usr/local
        sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
        check_status "解压Go失败"

        # 添加Go到PATH
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        check_status "添加Go到PATH失败"

        source ~/.bashrc
        # 清理下载文件
        rm go1.21.6.linux-amd64.tar.gz
        echo "Go安装完成,请运行 'source ~/.bashrc' 或重新登录以应用更改"
    else
        echo "Go已安装"
    fi
}

# 检查Docker
check_docker() {
    if ! command -v docker &>/dev/null; then
        log_error "Docker未安装,请先安装Docker"
        exit 1
    fi
    echo "Docker已安装"
}

# 检查Docker Compose
check_docker_compose() {
    if ! command -v docker-compose &>/dev/null; then
        log_error "Docker Compose未安装,请先安装Docker Compose"
        exit 1
    fi
    echo "Docker Compose已安装"
}

# 编译agent
compile_agent() {
    echo "正在编译agent..."
    go build -o agent
    check_status "编译agent失败"
    echo "agent编译完成"
}

# 启动项目
start_project() {
    echo "正在启动项目..."
    docker-compose up -d
    check_status "启动项目失败"
    echo "项目启动成功"
}

# 主函数
main() {
    check_docker
    check_docker_compose
    check_go
    compile_agent
    start_project
}

# 执行主函数
main
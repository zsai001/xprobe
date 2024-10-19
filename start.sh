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

# 检查并安装Docker
check_and_install_docker() {
    if ! command -v docker &>/dev/null; then
        echo "Docker未安装,正在安装..."
        curl -fsSL https://get.docker.com -o get-docker.sh
        check_status "下载Docker安装脚本失败"
        sudo sh get-docker.sh
        check_status "Docker安装失败"
        rm get-docker.sh
        
        # 将当前用户添加到docker组
        sudo usermod -aG docker $USER
        check_status "将用户添加到docker组失败"
        
        echo "Docker安装完成,请重新登录以应用组更改"
    else
        echo "Docker已安装"
    fi
}

# 检查并安装Docker Compose
check_and_install_docker_compose() {
    if ! docker compose version &>/dev/null; then
        echo "Docker Compose未安装,正在安装..."
        sudo apt-get update
        sudo apt-get install -y docker-compose-plugin
        check_status "安装Docker Compose失败"
        echo "Docker Compose安装完成"
    else
        echo "Docker Compose已安装"
    fi
}

# 检查并安装Go
check_and_install_go() {
    if ! command -v go &>/dev/null; then
        echo "Go未安装,正在安装..."
        wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
        check_status "下载Go失败"
        sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
        check_status "解压Go失败"
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        check_status "添加Go到PATH失败"
        source ~/.bashrc
        rm go1.21.6.linux-amd64.tar.gz
        echo "Go安装完成,请运行 'source ~/.bashrc' 或重新登录以应用更改"
    else
        echo "Go已安装"
    fi
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
    docker compose up -d
    check_status "启动项目失败"
    echo "项目启动成功"
}

# 主函数
main() {
    check_and_install_docker
    check_and_install_docker_compose
    check_and_install_go
    compile_agent
    start_project
}

# 执行主函数
main
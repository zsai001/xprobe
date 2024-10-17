#!/bin/bash

# 检查是否已安装Docker
check_docker() {
    if ! command -v docker &> /dev/null; then
        echo "Docker未安装,正在安装..."
        curl -fsSL https://get.docker.com -o get-docker.sh
        sudo sh get-docker.sh
        sudo usermod -aG docker $USER
        echo "Docker安装完成,请重新登录以应用更改"
        exit
    else
        echo "Docker已安装"
    fi
}

# 检查是否已安装Docker Compose
check_docker_compose() {
    if ! command -v docker-compose &> /dev/null; then
        echo "Docker Compose未安装,正在安装..."
        sudo curl -L "https://github.com/docker/compose/releases/download/1.29.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
        sudo chmod +x /usr/local/bin/docker-compose
        echo "Docker Compose安装完成"
    else
        echo "Docker Compose已安装"
    fi
}

# 编译agent到html目录
compile_agent() {
    echo "正在编译agent到html目录..."
    cd agent || exit
    
    # 定义要编译的操作系统和架构
    OS_ARCH_PAIRS=(
        "linux/amd64"
        "linux/386"
        "linux/arm"
        "linux/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
        "windows/386"
    )
    
    for os_arch in "${OS_ARCH_PAIRS[@]}"; do
        IFS='/' read -r os arch <<< "$os_arch"
        echo "编译 $os/$arch ..."
        
        output_dir="../html/$os/$arch"
        mkdir -p "$output_dir"
        
        if [ "$os" == "windows" ]; then
            output_name="xprob_agent.exe"
        else
            output_name="xprob_agent"
        fi
        
        GOOS=$os GOARCH=$arch go build -o "$output_dir/$output_name" .
        
        if [ $? -eq 0 ]; then
            echo "$os/$arch 编译成功"
        else
            echo "$os/$arch 编译失败"
        fi
    done
    
    cd ..
    echo "所有版本的agent编译完成"
}


# 启动Docker Compose项目
start_project() {
    echo "正在启动Docker Compose项目..."
    docker-compose up --build -d
    echo "项目已启动"
}

# 主函数
main() {
    check_docker
    check_docker_compose
    compile_agent
    start_project
}

# 运行主函数
main
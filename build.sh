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

        output_dir="../html/agent/$os/$arch"
        mkdir -p "$output_dir"

        if [ "$os" == "windows" ]; then
            output_name="xprobe_agent.exe"
        else
            output_name="xprobe_agent"
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
    echo "正在编译agent..."
    go build -o agent
    check_status "编译agent失败"
    echo "agent编译完成"
}

compile_agent
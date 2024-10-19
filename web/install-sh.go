package web

import (
	"github.com/gin-gonic/gin"
	"net/url"
	"text/template"
)

type ScriptData struct {
	ServerURL string
}

func InstallSh(c *gin.Context) {
	host := c.Request.Host
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := url.URL{
		Scheme: scheme,
		Host:   host,
	}
	data := ScriptData{
		ServerURL: baseURL.String(),
	}
	c.Header("Content-Type", "text/plain")
	bashTemplate.Execute(c.Writer, data)
}

var bashTemplate = template.Must(template.New("bash").Parse(`#!/bin/bash

# XProbe Agent Installation Script
# Supports Linux, macOS, and Windows (via WSL or Git Bash)

XPROBE_SERVER="{{.ServerURL}}"
XPROBE_KEY="$1"

if [ -z "$XPROBE_KEY" ]; then
    echo "错误: 未提供 XProbe 密钥"
    exit 1
fi

# Detect OS and architecture
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="darwin"
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
    OS="windows"
else
    echo "不支持的操作系统"
    exit 1
fi

ARCH=$(uname -m)
case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "不支持的架构: $ARCH"
        exit 1
        ;;
esac

# Download XProbe agent
echo "正在下载 XProbe agent ($OS/$ARCH)..."
curl -fsSL "$XPROBE_SERVER/agent/$OS/$ARCH/xprobe_agent" -o xprobe_agent

# Make agent executable
chmod +x xprobe_agent

# Install agent
if [[ "$OS" == "linux" || "$OS" == "darwin" ]]; then
    sudo mv xprobe_agent /usr/local/bin/xprobe_agent

    # Create systemd service for Linux
    if [[ "$OS" == "linux" ]]; then
        cat << EOF | sudo tee /etc/systemd/system/xprobe.service
[Unit]
Description=XProbe Agent Service
After=network.target

[Service]
ExecStart=/usr/local/bin/xprobe_agent $XPROBE_SERVER $XPROBE_KEY
Restart=always
User=root

[Install]
WantedBy=multi-user.target
EOF

        sudo systemctl daemon-reload
        sudo systemctl enable xprobe.service
        sudo systemctl start xprobe.service

    # Create launchd service for macOS
    elif [[ "$OS" == "darwin" ]]; then
        cat << EOF | sudo tee /Library/LaunchDaemons/com.xprobe.agent.plist
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.xprobe.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/xprobe_agent</string>
        <string>$XPROBE_SERVER</string>
        <string>$XPROBE_KEY</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
EOF

        sudo launchctl load /Library/LaunchDaemons/com.xprobe.agent.plist
    fi

elif [[ "$OS" == "windows" ]]; then
    mkdir -p "$USERPROFILE/XProbe"
    mv xprobe_agent "$USERPROFILE/XProbe/xprobe_agent.exe"

    # Create scheduled task for Windows
    powershell -Command "
        \$action = New-ScheduledTaskAction -Execute '$USERPROFILE\XProbe\xprobe_agent.exe' -Argument '$XPROBE_SERVER $XPROBE_KEY'
        \$trigger = New-ScheduledTaskTrigger -AtStartup
        \$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -RestartInterval (New-TimeSpan -Minutes 1) -RestartCount 3
        Register-ScheduledTask -TaskName 'XProbe Agent' -Action \$action -Trigger \$trigger -Settings \$settings -RunLevel Highest -Force
    "
fi

echo "XProbe agent 安装成功!"
`))

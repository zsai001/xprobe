package web

import (
	"fmt"
	"text/template"

	"github.com/gin-gonic/gin"
)

type ScriptData struct {
	ServerURL string
}

func InstallSh(c *gin.Context) {
	host := c.Request.Host
	data := ScriptData{
		ServerURL: fmt.Sprintf("http://%s", host),
	}
	c.Header("Content-Type", "text/plain")
	bashTemplate.Execute(c.Writer, data)
}

var bashTemplate = template.Must(template.New("bash").Parse(`#!/bin/bash

# XPorb Agent Installation Script
# Supports Linux, macOS, and Windows (via WSL or Git Bash)

XPROB_SERVER="{{.ServerURL}}"
XPROB_KEY="$1"

if [ -z "$XPORB_KEY" ]; then
    echo "Error: XPorb key not provided"
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
    echo "Unsupported operating system"
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
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Download XPorb agent
echo "Downloading XPorb agent for $OS/$ARCH..."
curl -fsSL "$XPORB_SERVER/agent/$OS/$ARCH/xprob_agent" -o xprob_agent

# Make agent executable
chmod +x xprob_agent

# Install agent
if [[ "$OS" == "linux" || "$OS" == "darwin" ]]; then
    sudo mv xprob_agent /usr/local/bin/xprob_agent

    # Create systemd service for Linux
    if [[ "$OS" == "linux" ]]; then
        cat << EOF | sudo tee /etc/systemd/system/xporb.service
[Unit]
Description=XPorb Agent Service
After=network.target

[Service]
ExecStart=/usr/local/bin/xprob_agent $XPORB_KEY
Restart=always
User=root

[Install]
WantedBy=multi-user.target
EOF

        sudo systemctl daemon-reload
        sudo systemctl enable xporb.service
        sudo systemctl start xporb.service

    # Create launchd service for macOS
    elif [[ "$OS" == "darwin" ]]; then
        cat << EOF | sudo tee /Library/LaunchDaemons/com.xporb.agent.plist
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.xprob.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/xprob_agent</string>
        <string>$XPORB_KEY</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
EOF

        sudo launchctl load /Library/LaunchDaemons/com.xporb.agent.plist
    fi

elif [[ "$OS" == "windows" ]]; then
    mkdir -p "$USERPROFILE/XPorb"
    mv xprob_agent "$USERPROFILE/XPorb/xprob_agent.exe"

    # Create scheduled task for Windows
    powershell -Command "
        \$action = New-ScheduledTaskAction -Execute '$USERPROFILE\XPorb\xprob_agent.exe' -Argument '$XPORB_KEY'
        \$trigger = New-ScheduledTaskTrigger -AtStartup
        \$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -RestartInterval (New-TimeSpan -Minutes 1) -RestartCount 3
        Register-ScheduledTask -TaskName 'XPorb Agent' -Action \$action -Trigger \$trigger -Settings \$settings -RunLevel Highest -Force
    "
fi

echo "XPorb agent installed successfully!"
`))

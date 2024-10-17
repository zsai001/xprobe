package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"text/template"
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

# XPorb Client Installation Script
# Supports Linux, macOS, and Windows (via WSL or Git Bash)

XPORB_SERVER="{{.ServerURL}}"
XPORB_KEY="$1"

if [ -z "$XPORB_KEY" ]; then
    echo "Error: XPorb key not provided"
    exit 1
fi

# Detect OS
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="macos"
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
    OS="windows"
else
    echo "Unsupported operating system"
    exit 1
fi

# Download XPorb client
echo "Downloading XPorb client for $OS..."
curl -fsSL "$XPORB_SERVER/xporb-client-$OS" -o xporb-client

# Make client executable
chmod +x xporb-client

# Install client
if [[ "$OS" == "linux" || "$OS" == "macos" ]]; then
    sudo mv xporb-client /usr/local/bin/xporb-client

    # Create systemd service for Linux
    if [[ "$OS" == "linux" ]]; then
        cat << EOF | sudo tee /etc/systemd/system/xporb.service
[Unit]
Description=XPorb Client Service
After=network.target

[Service]
ExecStart=/usr/local/bin/xporb-client $XPORB_KEY
Restart=always
User=root

[Install]
WantedBy=multi-user.target
EOF

        sudo systemctl daemon-reload
        sudo systemctl enable xporb.service
        sudo systemctl start xporb.service

    # Create launchd service for macOS
    elif [[ "$OS" == "macos" ]]; then
        cat << EOF | sudo tee /Library/LaunchDaemons/com.xporb.client.plist
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.xporb.client</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/xporb-client</string>
        <string>$XPORB_KEY</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
EOF

        sudo launchctl load /Library/LaunchDaemons/com.xporb.client.plist
    fi

elif [[ "$OS" == "windows" ]]; then
    mkdir -p "$USERPROFILE/XPorb"
    mv xporb-client "$USERPROFILE/XPorb/xporb-client.exe"

    # Create scheduled task for Windows
    powershell -Command "
        \$action = New-ScheduledTaskAction -Execute '$USERPROFILE\XPorb\xporb-client.exe' -Argument '$XPORB_KEY'
        \$trigger = New-ScheduledTaskTrigger -AtStartup
        \$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -RestartInterval (New-TimeSpan -Minutes 1) -RestartCount 3
        Register-ScheduledTask -TaskName 'XPorb Client' -Action \$action -Trigger \$trigger -Settings \$settings -RunLevel Highest -Force
    "
fi

echo "XPorb client installed successfully!"
`))

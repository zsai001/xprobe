package web

import (
	"net/url"
	"server/util"
	"text/template"

	"github.com/gin-gonic/gin"
)

func InstallPs(c *gin.Context) {
	host := c.Request.Host
	scheme := "http"
	if util.IsHTTPS(c) {
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
	powershellTemplate.Execute(c.Writer, data)
}

var powershellTemplate = template.Must(template.New("powershell").Parse(`# XProbe Agent Windows Installation Script

param(
    [Parameter(Mandatory=$true)]
    [string]$XProbeKey
)

$ErrorActionPreference = "Stop"

$XProbeServer = "{{.ServerURL}}"
$InstallDir = "$env:ProgramFiles\XProbe"
$ExecutablePath = "$InstallDir\xprobe_agent.exe"

# 检测架构
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

# 创建安装目录
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# 下载 XProbe agent
Write-Host "正在下载 XProbe agent..."
Invoke-WebRequest -Uri "$XProbeServer/agent/windows/$arch/xprobe_agent" -OutFile $ExecutablePath

# 创建开机启动任务
$Action = New-ScheduledTaskAction -Execute $ExecutablePath -Argument "$XProbeServer $XProbeKey"
$Trigger = New-ScheduledTaskTrigger -AtStartup
$Settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -RestartInterval (New-TimeSpan -Minutes 1) -RestartCount 3

# 注册计划任务
Register-ScheduledTask -TaskName "XProbe Agent" -Action $Action -Trigger $Trigger -Settings $Settings -User "SYSTEM" -RunLevel Highest -Force

# 立即启动 agent
Start-Process -FilePath $ExecutablePath -ArgumentList "$XProbeServer $XProbeKey"

Write-Host "XProbe agent 安装成功!"
`))

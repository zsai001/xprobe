package web

import (
	"fmt"
	"text/template"

	"github.com/gin-gonic/gin"
)

func InstallPs(c *gin.Context) {
	host := c.Request.Host
	data := ScriptData{
		ServerURL: fmt.Sprintf("http://%s", host),
	}

	c.Header("Content-Type", "text/plain")
	powershellTemplate.Execute(c.Writer, data)
}

var powershellTemplate = template.Must(template.New("powershell").Parse(`# XPorb Agent Windows Installation Script

param(
    [Parameter(Mandatory=$true)]
    [string]$XPorbKey
)

$ErrorActionPreference = "Stop"

$XPorbServer = "{{.ServerURL}}"
$InstallDir = "$env:ProgramFiles\XProb"
$ExecutablePath = "$InstallDir\xprob_agent.exe"

# Detect architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

# Create installation directory
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# Download XPorb agent
Write-Host "Downloading XProb agent..."
Invoke-WebRequest -Uri "$XPorbServer/agent/windows/$arch/xprob_agent" -OutFile $ExecutablePath

# Create a scheduled task to run at startup
$Action = New-ScheduledTaskAction -Execute $ExecutablePath -Argument $XPorbKey
$Trigger = New-ScheduledTaskTrigger -AtStartup
$Settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -RestartInterval (New-TimeSpan -Minutes 1) -RestartCount 3

# Register the scheduled task
Register-ScheduledTask -TaskName "XProb Agent" -Action $Action -Trigger $Trigger -Settings $Settings -User "SYSTEM" -RunLevel Highest -Force

# Start the agent immediately
Start-Process -FilePath $ExecutablePath -ArgumentList $XPorbKey

Write-Host "XProb agent installed successfully!"
`))

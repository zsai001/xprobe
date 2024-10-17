package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"text/template"
)

func InstallPs(c *gin.Context) {
	host := c.Request.Host
	data := ScriptData{
		ServerURL: fmt.Sprintf("http://%s", host),
	}

	c.Header("Content-Type", "text/plain")
	powershellTemplate.Execute(c.Writer, data)
}

var powershellTemplate = template.Must(template.New("powershell").Parse(`# XPorb Client Windows Installation Script

param(
    [Parameter(Mandatory=$true)]
    [string]$XPorbKey
)

$ErrorActionPreference = "Stop"

$XPorbServer = "{{.ServerURL}}"
$InstallDir = "$env:ProgramFiles\XProb"
$ExecutablePath = "$InstallDir\xprob-client.exe"

# Create installation directory
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# Download XPorb client
Write-Host "Downloading XProb client..."
Invoke-WebRequest -Uri "$XProbServer/xprob-client-windows" -OutFile $ExecutablePath

# Create a scheduled task to run at startup
$Action = New-ScheduledTaskAction -Execute $ExecutablePath -Argument $XProbKey
$Trigger = New-ScheduledTaskTrigger -AtStartup
$Settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -RestartInterval (New-TimeSpan -Minutes 1) -RestartCount 3

# Register the scheduled task
Register-ScheduledTask -TaskName "XProb Client" -Action $Action -Trigger $Trigger -Settings $Settings -User "SYSTEM" -RunLevel Highest -Force

# Start the client immediately
Start-Process -FilePath $ExecutablePath -ArgumentList $XProbKey

Write-Host "XProb client installed successfully!"
`))

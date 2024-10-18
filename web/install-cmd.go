package web

import (
	"fmt"
	"text/template"

	"github.com/gin-gonic/gin"
)

func InstallCmd(c *gin.Context) {
	host := c.Request.Host
	data := ScriptData{
		ServerURL: fmt.Sprintf("http://%s", host),
	}
	c.Header("Content-Type", "text/plain")
	cmdTemplate.Execute(c.Writer, data)
}

var cmdTemplate = template.Must(template.New("cmd").Parse(`@echo off
:: XProbe Agent Windows Installation Script

if "%1"=="" (
    echo Error: XProbe key not provided
    exit /b 1
)

set XPROBE_SERVER={{.ServerURL}}
set XPROBE_KEY=%1
set INSTALL_DIR=%ProgramFiles%\XProbe
set EXECUTABLE=%INSTALL_DIR%\xprobe_agent.exe

:: Detect architecture
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    set ARCH=amd64
) else (
    set ARCH=386
)

:: Create installation directory
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

:: Download XProbe agent
echo Downloading XProbe agent...
powershell -Command "& {Invoke-WebRequest -Uri '%XPROBE_SERVER%/agent/windows/%ARCH%/xprobe_agent' -OutFile '%EXECUTABLE%'}"

:: Create a scheduled task to run at startup
schtasks /create /tn "XProbe Agent" /tr "'%EXECUTABLE%' %XPROBE_SERVER% %XPROBE_KEY%" /sc onstart /ru SYSTEM /rl HIGHEST /f

:: Start the agent immediately
start "" "%EXECUTABLE%" %XPROBE_SERVER% %XPROBE_KEY%

echo XProbe agent installed successfully!
`))

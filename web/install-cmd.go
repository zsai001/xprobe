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
:: XPorb Agent Windows Installation Script

if "%1"=="" (
    echo Error: XPorb key not provided
    exit /b 1
)

set XPORB_SERVER={{.ServerURL}}
set XPORB_KEY=%1
set INSTALL_DIR=%ProgramFiles%\XProb
set EXECUTABLE=%INSTALL_DIR%\xprob_agent.exe

:: Detect architecture
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    set ARCH=amd64
) else (
    set ARCH=386
)

:: Create installation directory
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

:: Download XPorb agent
echo Downloading XPorb agent...
powershell -Command "& {Invoke-WebRequest -Uri '%XPORB_SERVER%/agent/windows/%ARCH%/xprob_agent' -OutFile '%EXECUTABLE%'}"

:: Create a scheduled task to run at startup
schtasks /create /tn "XProb Agent" /tr "'%EXECUTABLE%' %XPORB_KEY%" /sc onstart /ru SYSTEM /rl HIGHEST /f

:: Start the agent immediately
start "" "%EXECUTABLE%" %XPORB_KEY%

echo XProb agent installed successfully!
`))

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
:: XProb Client Windows Installation Script

setlocal enabledelayedexpansion

:: Check if XProb key is provided
if "%~1"=="" (
    echo Error: XProb key not provided
    exit /b 1
)

set XPROB_KEY=%~1
set XPROB_SERVER={{.ServerURL}}
set INSTALL_DIR=%ProgramFiles%\XProb
set EXECUTABLE=%INSTALL_DIR%\xprob-client.exe

:: Create installation directory
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

:: Download XProb client using certutil
echo Downloading XProb client...
certutil -urlcache -split -f "%XPROB_SERVER%/xprob-client-windows" "%EXECUTABLE%"

:: Check if download was successful
if not exist "%EXECUTABLE%" (
    echo Failed to download XProb client. Please check your internet connection and try again.
    exit /b 1
)

:: Create a scheduled task to run at startup
echo Creating scheduled task...
schtasks /create /tn "XProb Client" /tr "'%EXECUTABLE%' %XPROB_KEY%" /sc onstart /ru SYSTEM /f

:: Start the client immediately
echo Starting XProb client...
start "" "%EXECUTABLE%" %XPROB_KEY%

echo XProb client installed successfully!
`))

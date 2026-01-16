@echo off
setlocal enabledelayedexpansion

REM Call the PowerShell script
powershell.exe -ExecutionPolicy Bypass -File "%~dp0release.ps1"

exit /b %errorlevel%
@echo off
chcp 65001 >nul

cd /d "%~dp0"

start "health-reminder frontend" cmd /k .\start_frontend_dev.bat

start "health-reminder backend" cmd /k .\start_backend_dev.bat

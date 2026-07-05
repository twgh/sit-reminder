@echo off
chcp 65001 >nul

cd /d "%~dp0"

start "sit-reminder frontend" cmd /k .\start_frontend_dev.bat

start "sit-reminder backend" cmd /k .\start_backend_dev.bat

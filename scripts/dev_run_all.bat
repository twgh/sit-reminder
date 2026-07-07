@echo off
chcp 65001 >nul

cd /d "%~dp0"

start "sit-reminder frontend" cmd /k dev_start_frontend.bat

start "sit-reminder backend" cmd /k dev_start_backend.bat

exit
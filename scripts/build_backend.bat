@echo off
chcp 65001 >nul
cd /d "%~dp0../backend"

set "output=bin/health-reminder.exe"

echo 开始编译【久坐提醒助手】 ...

go build -trimpath -ldflags="-s -w -H windowsgui" -o ../%output%

echo 编译成功! 文件位置: %output%
pause
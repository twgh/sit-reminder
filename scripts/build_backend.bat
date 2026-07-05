@echo off
chcp 65001 >nul
cd /d "%~dp0..\backend"

echo 开始编译【久坐提醒助手】 ...

REM 设置输出文件名
set "output=bin/health-reminder.exe"

REM 获取当前日期, 设置为版本号
for /f "tokens=1,2,3 delims=/ " %%a in ('date /t') do (
    set year=%%a
    set month=%%b
    set day=%%c
)
set datestr=%year%.%month%.%day%

REM 设置编译参数
set LDFLAGS=-X 'github.com/twgh/health-reminder/g.Version=%datestr%' -X 'github.com/twgh/health-reminder/g.DebugState=0'

REM 编译
go build -trimpath -ldflags="%LDFLAGS% -s -w -H windowsgui" -o ../%output%

if %error level% == 0 (
    echo 编译成功! 文件位置: %output%
) else (
    echo 编译失败!
)

pause
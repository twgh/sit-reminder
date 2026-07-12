@echo off
chcp 65001 >nul
cd /d "%~dp0..\backend"

echo 开始编译【久坐提醒助手】 ...

REM 设置输出文件名
set "output=bin/sit-reminder.exe"

REM 获取当前日期, 设置为版本号
for /f "tokens=1,2,3 delims=/ " %%a in ('date /t') do (
    set year=%%a
    set month=%%b
    set day=%%c
)
set "datestr=%year%.%month%.%day%.0"
echo 版本号: %datestr%

REM 设置编译参数
set LDFLAGS=-X 'github.com/twgh/sit-reminder/internal/g.Version=%datestr%' -X 'github.com/twgh/sit-reminder/internal/g.DebugState=0'
echo 编译参数: %LDFLAGS%

REM 编译
go build -trimpath -ldflags="%LDFLAGS% -s -w -H windowsgui" -o ../%output%

if %errorlevel% == 0 (
    echo 编译成功! 文件位置: %output%
) else (
    echo 编译失败!
)

pause
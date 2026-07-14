@echo off
chcp 65001 >nul

set "filename=sit-reminder-dev.exe"

:: 1. 先通过 tasklist 检查进程是否真的存在
tasklist /FI "IMAGENAME eq %filename%" 2>nul | find /I "%filename%" >nul

:: 2. 如果存在（find 命令返回 0），则执行结束操作
if %errorlevel% equ 0 (
    echo 发现 %filename% 进程，正在强制结束...
    taskkill /F /IM sit-reminder-dev.exe /T >nul 2>nul
    echo 结束已有进程成功
) 

echo 开始编译【久坐提醒助手】 ...
cd /d "%~dp0..\backend"

go build -o ../bin/%filename%

if %errorlevel%==0 (
    echo 编译成功! 文件位置: bin\%filename%

    echo 启动 %filename%
    echo.
    cd /d "..\bin"
    %filename%
    exit
) else (
    echo 编译失败!
)

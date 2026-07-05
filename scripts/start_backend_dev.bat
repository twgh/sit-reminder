@echo off
chcp 65001 >nul

echo 开始编译【久坐提醒助手】 ...
cd /d "%~dp0..\backend"

set "filename=health-reminder-dev.exe"
go build -o ../bin/%filename%

if %errorlevel%==0 (
    echo 编译成功! 文件位置: bin\%filename%

    echo 启动 %filename%
    echo.
    cd /d "..\bin"
    %filename%
) else (
    echo 编译失败!
)

pause
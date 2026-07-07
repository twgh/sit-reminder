@echo off
chcp 65001 >nul

echo 启动前端 ...
cd /d "%~dp0..\frontend"

:: 检测 5173 端口是否被占用
netstat -ano | findstr ":5173" | findstr "LISTENING" >nul
if %errorlevel% equ 0 (
    echo [提示] 端口 5173 已被占用，前端服务可能已经在运行中，跳过启动。
) else (
    :: 使用 start 命令启动一个新的独立 CMD 窗口来运行 pnpm dev
    start "Frontend Dev Server" cmd /k "pnpm dev"
)

exit

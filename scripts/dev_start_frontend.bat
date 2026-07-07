@echo off
chcp 65001 >nul

echo 启动前端 ...
REM 进入前端目录
cd /d "%~dp0..\frontend"

REM 检查 node_modules 是否存在
if not exist "node_modules" (
    echo 首次运行，正在安装依赖...
    call pnpm install
    if errorlevel 1 (
        echo 安装依赖失败！
        pause
        exit /b 1
    )
) else (
    echo 依赖已安装
)


:: 检测 5173 端口是否被占用
netstat -ano | findstr ":5173" | findstr "LISTENING" >nul
if %errorlevel% equ 0 (
    echo [提示] 端口 5173 已被占用，前端服务可能已经在运行中，跳过启动。
) else (
    :: 使用 start 命令启动一个新的独立 CMD 窗口来运行 pnpm dev
    start "Frontend Dev Server" cmd /k "pnpm dev"
)

exit

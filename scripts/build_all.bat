@echo off
chcp 65001 >nul

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


echo 开始编译前端 ...
pnpm build && cd /d "..\scripts" && call build_backend.bat

if %errorlevel% == 0 (
) else (
    echo 编译失败!
    pause
)

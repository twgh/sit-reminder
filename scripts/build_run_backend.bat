@echo off
chcp 65001 >nul

REM 先编译
call "%~dp0build_backend.bat"
if %errorlevel% neq 0 (
    pause
    exit /b %errorlevel%
)

REM 编译成功, 运行 exe
cd /d "%~dp0..\bin" && sit-reminder.exe

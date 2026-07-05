@echo off
chcp 65001 >nul
cd /d "%~dp0..\frontend"

echo 开始编译前端 ...
pnpm build && cd /d "..\scripts" && call build_backend.bat

pause
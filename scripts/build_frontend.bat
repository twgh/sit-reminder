@echo off
chcp 65001 >nul
cd /d "%~dp0../frontend"

echo 开始编译前端 ...
pnpm build

pause
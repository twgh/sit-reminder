@echo off
chcp 65001 >nul

echo 启动前端 ...
cd /d "%~dp0../frontend"
pnpm dev

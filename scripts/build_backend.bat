@echo off
chcp 65001 >nul

REM 进入后端目录
cd /d "%~dp0..\backend"

echo 开始编译【久坐提醒助手】 ...

REM 设置输出文件名
set "output=bin/sit-reminder.exe"

REM 获取当前日期, 设置为版本号 (格式: yyyy.m.d.0)
for /f "tokens=1,2,3 delims=/ " %%a in ('date /t') do (
    set /a _y=%%a, _m=%%b, _d=%%c
)
set "datestr=%_y%.%_m%.%_d%.0"
echo 版本号: %datestr%

REM 将版本号同步写入 winres.json (供 go-winres 生成 exe 版本资源)
set "WINRES_VER=%datestr%"
node "%~dp0update_winres_json.js"
if %errorlevel% neq 0 (
    echo 写入 winres.json 版本号失败!
    pause
    exit /b 1
)

REM 校验 go-winres 是否可用
where go-winres >nul 2>nul
if %errorlevel% neq 0 (
    echo 未找到 go-winres, 请先执行: go install github.com/tc-hib/go-winres@latest
    pause
    exit /b 1
)

REM 生成 Windows 资源(图标/清单/版本信息)
go-winres make
if %errorlevel% neq 0 (
    echo go-winres make 失败!
    pause
    exit /b 1
)

REM 设置编译参数
set LDFLAGS=-X 'github.com/twgh/sit-reminder/internal/g.Version=%datestr%' -X 'github.com/twgh/sit-reminder/internal/g.DebugState=0'
echo 编译参数: %LDFLAGS%

REM 编译
go build -trimpath -ldflags="%LDFLAGS% -s -w -H windowsgui" -o ../%output%

if %errorlevel% == 0 (
    echo 编译成功! 文件位置: %output%
) else (
    echo 编译失败!
    pause
)

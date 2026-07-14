## 久坐提醒助手

基于 Go + XCGUI (WebView2) + React + shadcn/ui 的桌面久坐提醒应用。

## 功能

- **时间设置**: 间隔时间输入（默认40分钟），支持步进器(-10/-5/+5/+10)，失焦自动保存到 YAML 配置文件
- **启停控制**: 启动/停止/暂停/继续按钮
- **状态显示**: 当前状态、剩余时间、进度条可视化
- **铃声设置**: 选择铃声文件、测试播放、设置铃声音量
- **提醒通知**: 到时弹出醒目提醒，播放铃声(循环)，可手动停止, 可开启活动倒计时, 活动结束后可启动下一次久坐提醒
- **稍后提醒**: 支持 3/5/10 分钟后再次提醒
- **主题模式**: 支持浅色/深色/跟随系统
- **其它**: 支持开机自启, 关闭窗口时隐藏到托盘, 全局快捷键显示/隐藏窗口

## 技术栈

- **前端**: React 19 + TypeScript + Vite + shadcn/ui (base) + Tailwind CSS v4
- **后端**: Go + XCGUI (炫彩界面库) + WebView2

## 项目结构

```
├── frontend/          # React 前端
│   ├── src/
│   │   ├── App.tsx              # 主界面
│   │   ├── lib/bridge.ts        # Go <-> JS 通信桥接
│   │   └── components/ui/       # shadcn 组件
│   ├── README.md                # 前端开发说明
│   └── package.json
├── backend/           # Go 后端
│   ├── main.go                  # 程序入口
│   ├── go.mod
│   └── winres/                  # 图标, 版本信息, 程序清单
│   └── internal/                # 内部包
│       ├── config/              # 配置包
│       ├── gui/                 # 程序 GUI 界面
│       ├── utils/               # 工具包
│       └── g/                   # 全局变量
├── scripts/           # 脚本目录
│   ├── build_all.bat           # 编译前端和后端正式版
│   ├── build_backend.bat       # 编译后端正式版
│   ├── build_frontend.bat      # 编译前端
│   ├── build_run_backend.bat   # 编译后端正式版并运行
│   ├── dev_run_all.bat         # 同时启动前端开发服务器和后端开发版本
│   ├── dev_start_backend.bat   # 启动后端开发版本
│   └── dev_start_frontend.bat  # 启动前端开发服务器
└── README.md
```

## 安装依赖

```bash
cd backend && go mod tidy
```

## 开发模式

```bash
# 进入脚本目录
cd scripts

# 启动前端和后端
dev_run_all.bat
```

Go 后端代码中 `g.IsDebug() = true` 时会连接 `http://localhost:5173`。

## 生产构建

```bash
cd scripts

# 编译前端和后端
build_all.bat
```

编译后得到单个exe，内嵌了所有前端资源。

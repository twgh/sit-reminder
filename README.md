## 久坐提醒助手

基于 Go + xcgui (WebView2) + React + shadcn/ui 的桌面久坐提醒应用。

## 技术栈

- **前端**: React 19 + TypeScript + Vite + shadcn/ui (base) + Tailwind CSS v4
- **后端**: Go + xcgui (炫彩界面库) + WebView2

## 项目结构

```
├── frontend/          # React 前端
│   ├── src/
│   │   ├── App.tsx              # 主界面
│   │   ├── lib/bridge.ts        # Go <-> JS 通信桥接
│   │   └── components/ui/       # shadcn 组件
│   └── package.json
├── backend/           # Go 后端
│   ├── main.go                  # 程序入口
│   ├── go.mod
│   └── winres/                  # 程序图标, 版本信息
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

## 功能

- **时间设置**: 间隔时间输入（默认40分钟），支持步进器(-10/-5/+5/+10)，失焦自动保存到 YAML 配置文件
- **启停控制**: 启动/停止/暂停/继续按钮
- **状态显示**: 当前状态、剩余时间、进度条可视化
- **铃声设置**: 选择铃声文件、测试播放
- **提醒通知**: 到时弹出醒目提醒，播放铃声(循环)，可手动停止, 可开启活动倒计时
- **稍后提醒**: 支持 3/5/10 分钟后再次提醒

## 架构设计

- 核心倒计时逻辑在 Go 后端（秒级 ticker），每秒通过 WebView Bridge 推送剩余时间到前端
- 前端只负责 UI 渲染，不运行任何计时器
- 铃声播放使用 Go 端 `wutil.AudioPlayer` (MCI)

## 开发模式

```bash
# 进入脚本目录
cd scripts

# 1. 启动前端开发服务器
dev_start_frontend.bat

# 2. 启动 Go 后端（连接 Vite 开发服务器，支持 HMR）
dev_start_backend.bat

# 或者两个都启动
dev_run_all.bat
```

Go 后端代码中 `g.IsDebug() = true` 时会连接 `http://localhost:5173`。

## 生产构建

```bash
cd scripts

# 1. 构建前端
build_frontend.bat

# 2. 编译 Go 后端
build_backend.bat

# 或者前端和后端一起编译
build_all.bat
```

编译后得到单个exe，内嵌了所有前端资源。

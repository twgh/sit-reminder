package gui

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/twgh/sit-reminder/internal/config"
	"github.com/twgh/sit-reminder/internal/g"
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/edge"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/wapi/wutil"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xcc"
)

type TimerState int

const (
	StateIdle TimerState = iota
	StateRunning
	StatePaused
	StateFinished
)

type TimerType int

const (
	TimerRegular  TimerType = iota // 常规提醒
	TimerSnooze                    // 稍后提醒
	TimerActivity                  // 活动倒计时
)

type MainWindow struct {
	edg    *edge.Edge
	w      *window.Window
	wv     *edge.WebView
	ap     *wutil.AudioPlayer // 提醒铃声
	ap2    *wutil.AudioPlayer // 活动结束铃声
	config *config.AppConfig

	mu        sync.Mutex
	timer     *time.Ticker
	done      chan struct{}
	pauseCh   chan struct{}
	resumeCh  chan struct{}
	state     TimerState
	timerType TimerType
	remaining int
	total     int
}

func NewMainWindow(edg *edge.Edge) *MainWindow {
	m := &MainWindow{
		edg:      edg,
		ap:       wutil.NewAudioPlayer(),
		ap2:      wutil.NewAudioPlayer(),
		config:   config.LoadConfig(),
		done:     make(chan struct{}),
		pauseCh:  make(chan struct{}),
		resumeCh: make(chan struct{}),
		state:    StateIdle,
	}

	var err error
	m.w, m.wv, err = m.edg.NewWebViewWithWindow(
		edge.WithXmlWindowTitle("久坐提醒助手"),
		edge.WithXmlWindowClassName(g.AppName),
		edge.WithXmlWindowSize(480, 528),
		edge.WithFillParent(true),
		edge.WithAppDrag(true),
		edge.WithDebug(g.IsDebug()),
		edge.WithDefaultContextMenus(g.IsDebug()),
		edge.WithBrowserAcceleratorKeys(g.IsDebug()),
		edge.WithStatusBar(false),
		edge.WithZoomControl(false),
		edge.WithAutoFocus(true),
		edge.WithDefaultBackgroundColor(edge.NewColor(0, 0, 0, 0)),
	)
	if err != nil {
		wapi.MessageBoxW(0, "创建 WebView 失败: "+err.Error(), "错误", wapi.MB_OK|wapi.MB_IconError)
		os.Exit(1)
	}

	// 从资源中加载程序图标
	hIconApp := wapi.LoadImageW(wapi.GetModuleHandleW(""), common.StrPtr("APPICON"), wapi.IMAGE_ICON, 0, 0, wapi.LR_SHARED|wapi.LR_DEFAULTSIZE)
	// 设置任务栏预览窗口左上角的图标
	m.w.SetSmallIcon(hIconApp)

	// 禁止拖拽边框改变窗口大小
	m.w.EnableDragBorder(false)

	if !g.IsDebug() {
		m.setupEmbedFS()
	} else {
		m.setupDevServer()
	}

	// 最小化时挂起 WebView 以节省内存
	m.saveMemory()

	m.regWebViewEvents()
	m.bindFunctions()
	m.wv.Navigate(m.getHost() + "/index.html")
	return m
}

// regWebViewEvents 注册 WebView 事件
func (m *MainWindow) regWebViewEvents() {
	firstLoad := true
	m.wv.Event_NavigationCompleted(func(sender *edge.ICoreWebView2, args *edge.ICoreWebView2NavigationCompletedEventArgs) uintptr {
		uri := sender.MustGetSource()
		fmt.Println("导航完成:", uri)
		if firstLoad && uri == m.getHost()+"/index.html" {
			firstLoad = false
			m.w.Show(true)
			m.pushConfig()
		}
		return 0
	})
}

// pushConfig 推送配置到 WebView
func (m *MainWindow) pushConfig() {
	jsonBytes, _ := json.Marshal(m.config)
	m.wv.Eval(fmt.Sprintf("window.__configLoaded && __configLoaded(%s)", string(jsonBytes)))
}

// bindFunctions 绑定函数
func (m *MainWindow) bindFunctions() {
	// ===== 窗口控制 =====
	m.wv.Bind("api.minimize", func() {
		m.w.ShowWindow(xcc.SW_MINIMIZE)
	})
	m.wv.Bind("api.close", func() {
		m.w.CloseWindow()
	})

	// ===== 定时器控制 =====
	m.wv.Bind("api.startTimer", func() {
		m.startTimer(TimerRegular)
	})
	m.wv.Bind("api.stopTimer", func() {
		m.stopTimer()
	})
	m.wv.Bind("api.pauseTimer", func() {
		m.pauseTimer()
	})
	m.wv.Bind("api.resumeTimer", func() {
		m.resumeTimer()
	})

	// 设置间隔（使用 float64 接收 JS number，避免 JSON 类型转换问题）
	m.wv.Bind("api.setInterval", func(minutes float64) {
		m.mu.Lock()
		m.config.IntervalMinutes = int(minutes)
		if m.state == StateIdle {
			m.total = int(minutes) * 60
			m.remaining = m.total
		}
		m.mu.Unlock()
	})

	// 稍后提醒（使用 float64）
	m.wv.Bind("api.snooze", func(minutes float64) {
		m.snooze(int(minutes))
	})

	// ===== 活动倒计时 =====
	m.wv.Bind("api.startActivityTimer", func() {
		m.startActivityTimer(5)
	})
	m.wv.Bind("api.startActivityTimerWithMinutes", func(minutes float64) {
		m.startActivityTimer(int(minutes))
	})

	// ===== 提醒铃声 =====
	m.wv.Bind("api.selectRingtone", func() string {
		return m.selectRingtone()
	})
	m.wv.Bind("api.testRingtone", func() {
		m.testRingtone()
	})
	m.wv.Bind("api.stopRingtone", func() {
		m.stopRingtone()
		m.stopActivityRingtone()
	})

	// ===== 活动结束铃声 =====
	m.wv.Bind("api.selectActivityRingtone", func() string {
		return m.selectActivityRingtone()
	})
	m.wv.Bind("api.testActivityRingtone", func() {
		m.testActivityRingtone()
	})

	// ===== 配置 =====
	m.wv.Bind("api.getConfig", func() *config.AppConfig {
		return m.config
	})
	m.wv.Bind("api.saveConfig", func() {
		if err := config.SaveConfig(m.config); err != nil {
			log.Println("保存配置失败:", err)
		}
		fmt.Println("配置已保存到", config.ConfigPath())
	})
}

// Run 程序入口
func Run() {
	checkWebView2()
	edg := createEdge()

	app.InitOrExit()
	a := app.New(true)
	a.EnableAutoDPI(true).EnableDPI(true)

	NewMainWindow(edg)

	a.Run()
	a.Exit()
}

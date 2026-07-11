package gui

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/twgh/sit-reminder/internal/config"
	"github.com/twgh/sit-reminder/internal/g"
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/edge"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/wapi/wutil"
	"github.com/twgh/xcgui/widget"
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
	TimerSedentary TimerType = iota // 久坐提醒
	TimerSnooze                     // 稍后提醒
	TimerActivity                   // 活动倒计时
)

type MainWindow struct {
	edg    *edge.Edge
	w      *window.Window
	wv     *edge.WebView
	ap     *wutil.AudioPlayer // 久坐提醒铃声
	ap2    *wutil.AudioPlayer // 活动结束铃声
	config *config.AppConfig
	tray   *window.TrayIcon // 托盘图标

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
		edge.WithXmlWindowSize(480, 536),
		edge.WithFillParent(true),
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

	// 禁止拖拽边框改变窗口大小
	m.w.EnableDragBorder(false)

	// 设置为透明窗口
	m.w.SetTransparentType(xcc.Window_Transparent_Shaped)

	// 从资源中加载程序图标
	hIconApp := wapi.LoadImageW(wapi.GetModuleHandleW(""), common.StrPtr("APPICON"), wapi.IMAGE_ICON, 0, 0, wapi.LR_SHARED|wapi.LR_DEFAULTSIZE)
	// 设置任务栏预览窗口左上角的图标
	m.w.SetSmallIcon(hIconApp)

	// 创建托盘图标
	m.tray = m.w.CreateTrayIcon(hIconApp, "久坐提醒助手")
	// 显示托盘图标
	m.tray.Show()

	// 注册炫彩事件
	m.regXcEvents()

	if !g.IsDebug() {
		m.setupEmbedFS()
	} else {
		m.setupDevServer()
	}

	// 节省 WebView 内存
	m.saveMemory()

	m.regWebViewEvents()
	m.bindFunctions()
	m.wv.Navigate(m.getHost() + "/index.html")
	return m
}

// regXcEvents 注册炫彩事件
func (m *MainWindow) regXcEvents() {
	// 窗口关闭事件
	m.w.AddEvent_Close(func(hWindow int, pbHandled *bool) int {
		*pbHandled = true // 拦截
		// 给 body 加 class 禁用关闭按钮的 hover 样式（CSS 已在 index.css 预定义）
		m.wv.EvalAsync(`document.body.classList.add('close-hover-disabled')`, func(errorCode syscall.Errno, result string) uintptr {
			m.wv.Show(false) // 隐藏 WebView
			m.w.Show(false)  // 隐藏窗口
			return 0
		})
		return 0
	})

	// 托盘图标事件
	m.w.AddEvent_TrayIcon(func(wParam, lParam uintptr, pbHandled *bool) int {
		if int32(wParam) != m.tray.Id { // 不是自定义的托盘图标唯一标识符.
			return 0
		}
		switch xcc.WM_(lParam) {
		case xcc.WM_LBUTTONDOWN: // 鼠标左键按下
			m.activateWindow()
		case xcc.WM_RBUTTONDOWN: // 鼠标右键按下
			// 创建菜单
			menu := widget.NewMenu()
			// 一级菜单
			menu.AddItem(100, "设置", 0, xcc.Menu_Item_Flag_Normal)
			menu.AddItem(99999, "退出", 0, xcc.Menu_Item_Flag_Normal)

			// 获取鼠标光标的屏幕坐标
			var pt wapi.POINT
			wapi.GetCursorPos(&pt)
			// 弹出菜单
			menu.Popup(m.w.GetHWND(), pt.X, pt.Y, 0, xcc.Menu_Popup_Position_Left_Top)
		}
		return 0
	})

	// 菜单被选择事件
	m.w.AddEvent_Menu_Select(func(hWindow int, nID int32, pbHandled *bool) int {
		switch nID {
		case 100: // 设置
			m.activateWindow()
			m.wv.Eval("window.__showSettings && window.__showSettings()")

		case 99999: // 退出
			m.w.DestroyWindow() // 销毁窗口
			app.PostQuitMessage(0)
		}
		return 0
	})
}

// regWebViewEvents 注册 WebView 事件
func (m *MainWindow) regWebViewEvents() {
	firstLoad := true
	// 导航完成事件
	m.wv.Event_NavigationCompleted(func(sender *edge.ICoreWebView2, args *edge.ICoreWebView2NavigationCompletedEventArgs) uintptr {
		uri := sender.MustGetSource()
		fmt.Println("导航完成:", uri)
		if uri == m.getHost()+"/index.html" {
			if firstLoad {
				firstLoad = false
				m.w.Show()
			}

			m.pushConfig() // 推送配置到 WebView
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

	// 设置窗口位置（供 JS WindowDrag 调用）
	m.wv.Bind("wnd.setPos", func(x, y int32) {
		m.w.SetPosition(m.w.DpiConv(x), m.w.DpiConv(y))
	})

	// ===== 定时器控制 =====
	m.wv.Bind("api.startTimer", func() {
		m.startTimer(TimerSedentary)
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
		m.startActivityTimer(m.config.ActivityMinutes)
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
	m.wv.Bind("api.setActivityMinutes", func(minutes float64) {
		m.mu.Lock()
		m.config.ActivityMinutes = int(minutes)
		m.mu.Unlock()
	})
	m.wv.Bind("api.setVolume", func(volume float64) {
		m.mu.Lock()
		m.config.Volume = int(volume)
		m.mu.Unlock()
	})
}

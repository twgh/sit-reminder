package gui

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/twgh/sit-reminder/internal/config"
	"github.com/twgh/sit-reminder/internal/g"
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/ease"
	"github.com/twgh/xcgui/edge"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/wapi/wutil"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
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

	origWidth  int32    // 窗口原始宽度
	origHeight int32    // 窗口原始高度
	lastPos    xc.POINT // 窗口上一次的位置

	hideOnStart bool // -hide 命令行参数: 启动后不显示窗口
}

func NewMainWindow(edg *edge.Edge, hideOnStart bool) *MainWindow {
	m := &MainWindow{
		edg:         edg,
		ap:          wutil.NewAudioPlayer(),
		ap2:         wutil.NewAudioPlayer(),
		config:      config.LoadConfig(),
		hideOnStart: hideOnStart,
		done:        make(chan struct{}),
		pauseCh:     make(chan struct{}),
		resumeCh:    make(chan struct{}),
		state:       StateIdle,
		origWidth:   480,
		origHeight:  536,
		lastPos:     xc.POINT{X: 0, Y: 0},
	}

	var err error
	m.w, m.wv, err = m.edg.NewWebViewWithWindow(
		edge.WithXmlWindowTitle("久坐提醒助手"),
		edge.WithXmlWindowClassName(g.AppName),
		edge.WithXmlWindowSize(m.origWidth, m.origHeight),
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

	// 根据配置设置窗口置顶
	m.applyAlwaysOnTop(m.config.AlwaysOnTop)

	// 从资源中加载程序图标
	hMod := wapi.GetModuleHandleW("")
	hIconApp := wapi.LoadImageW(hMod, common.StrPtr("APPICON"), wapi.IMAGE_ICON, 0, 0, wapi.LR_SHARED|wapi.LR_DEFAULTSIZE)

	// 设置任务栏预览窗口左上角的图标, 使用24x24尺寸, 也会影响任务管理器里的图标
	hIcon24 := wapi.LoadImageW(hMod, common.StrPtr("APPICON"), wapi.IMAGE_ICON, 24, 24, wapi.LR_SHARED)
	m.w.SetSmallIcon(hIcon24)

	// 设置大图标, 会影响任务栏图标, Alt+Tab 窗口图标
	m.w.SetBigIcon(hIconApp)

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
		*pbHandled = true // 拦截窗口关闭
		// 给 body 加 class 禁用关闭按钮的 hover 样式（CSS 已在 index.css 预定义）
		m.wv.EvalAsync(`document.body.classList.add('close-hover-disabled')`, func(errorCode syscall.Errno, result string) uintptr {
			m.animateToTray() // 缩放+位移动画后隐藏到托盘
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
				if m.hideOnStart { // 开机自启时不显示窗口
					m.wv.Show(false) // 顺带隐藏 WebView, 这会使其进入效率模式
				} else {
					m.w.Show()
				}
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
		m.maybeHideAfterStart()
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
		m.maybeHideAfterStart()
	})

	// ===== 活动倒计时 =====
	m.wv.Bind("api.startActivityTimer", func() {
		m.startActivityTimer(m.config.ActivityMinutes)
		m.maybeHideAfterStart()
	})
	m.wv.Bind("api.startActivityTimerWithMinutes", func(minutes float64) {
		m.startActivityTimer(int(minutes))
		m.maybeHideAfterStart()
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
	m.wv.Bind("api.setAutoHide", func(enabled bool) {
		m.mu.Lock()
		m.config.AutoHide = enabled
		m.mu.Unlock()
	})
	m.wv.Bind("api.setAlwaysOnTop", func(enabled bool) {
		m.mu.Lock()
		m.config.AlwaysOnTop = enabled
		m.mu.Unlock()
		m.applyAlwaysOnTop(enabled)
	})
	m.wv.Bind("api.setThemeMode", func(mode string) {
		m.mu.Lock()
		m.config.ThemeMode = mode
		m.mu.Unlock()
	})
	m.wv.Bind("api.setAutoStart", func(enabled bool) string {
		m.mu.Lock()
		m.config.AutoStart = enabled
		m.mu.Unlock()
		if err := m.setAutoStart(enabled); err != nil {
			// 回滚配置
			m.mu.Lock()
			m.config.AutoStart = false
			m.mu.Unlock()
			if enabled {
				return "设置开机自启失败：" + err.Error()
			}
			return "取消开机自启失败：" + err.Error()
		}
		return ""
	})

	// ===== 系统信息 =====
	m.wv.Bind("api.getVersion", func() string {
		return g.Version
	})
}

// activateWindow 激活窗口到前台
func (m *MainWindow) activateWindow() {
	m.wv.Show() // 显示 WebView
	// 恢复窗口原始大小和位置（动画过程中可能被缩放过）
	m.w.SetRect(&xc.RECT{Left: m.lastPos.X, Top: m.lastPos.Y, Right: m.lastPos.X + m.origWidth, Bottom: m.lastPos.Y + m.origHeight})
	m.w.ShowWindow(xcc.SW_SHOWNORMAL)
	if !m.config.AlwaysOnTop {
		m.w.SetTop().SetTop(false)
	}
	// 恢复关闭按钮的 hover 样式: 移除 body 上的禁用 class
	m.wv.Eval(`document.body.classList.remove('close-hover-disabled')`)
}

// hideWindow 隐藏窗口和 WebView
//   - 加一个 WebView 的隐藏是因为这样能让它在后台自动变成效率模式
func (m *MainWindow) hideWindow() {
	m.wv.Show(false)
	m.w.Show(false)
}

// maybeHideAfterStart 如果配置了自动隐藏，则在启动计时后隐藏窗口
func (m *MainWindow) maybeHideAfterStart() {
	m.mu.Lock()
	autoHide := m.config.AutoHide
	m.mu.Unlock()
	if autoHide {
		xc.UI(func() {
			m.animateToTray()
		})
	}
}

// applyAlwaysOnTop 设置窗口是否置顶
func (m *MainWindow) applyAlwaysOnTop(enabled bool) {
	m.w.SetTop(enabled)
}

// setAutoStart 设置/取消开机自启（通过注册表 HKCU\...\Run）
func (m *MainWindow) setAutoStart(enabled bool) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	if enabled {
		cmd := exec.Command("reg", "add",
			`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
			"/v", g.AppName,
			"/t", "REG_SZ",
			"/d", exePath+" -hide",
			"/f")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("写入注册表失败: %w", err)
		}
	} else {
		cmd := exec.Command("reg", "delete",
			`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
			"/v", g.AppName,
			"/f")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("删除注册表失败: %w", err)
		}
	}
	return nil
}

// animateToTray 缩放+位移动画：窗口等比例缩小并向屏幕右下角托盘区域平移，结束后隐藏窗口。
func (m *MainWindow) animateToTray() {
	// 获取窗口大小和位置
	rc := m.w.GetRectEx()
	m.lastPos.X = rc.Left
	m.lastPos.Y = rc.Top
	winW := rc.Right - rc.Left
	winH := rc.Bottom - rc.Top
	// 窗口中心坐标
	winCX := (rc.Left + rc.Right) / 2
	winCY := (rc.Top + rc.Bottom) / 2

	dpi := m.w.GetDPI()
	// 屏幕右下角（托盘位置）, 这个获取的屏幕大小是物理坐标, 比如2560*1600,
	// 要转换成逻辑坐标, 比如在系统 150% 缩放下计算出来是 1707*1067
	targetCX := wapi.MulDiv(wutil.GetScreenWidth(), 96, dpi)
	targetCY := wapi.MulDiv(wutil.GetScreenHeight(), 96, dpi)
	// 缩放到 80% 屏幕宽度位置
	targetCX = int32(0.8 * float32(targetCX))

	// 缓动动画，每步 10ms
	const steps = 15
	for t := 0; t < steps; t++ {
		v := ease.Quad(float32(t)/float32(steps), xcc.Ease_Type_InOut)

		// 缩放：1.0 → 0.01（避免缩到 0 导致窗口消失闪烁）
		scale := float32(1.0 - v*0.99)

		// 新窗口大小
		newW := int32(float32(winW) * scale)
		newH := int32(float32(winH) * scale)

		// 新窗口中心：从原位置插值到屏幕右下角
		newCX := int32(float32(winCX) + v*float32(targetCX-winCX))
		newCY := int32(float32(winCY) + v*float32(targetCY-winCY))

		// 逆推左上角坐标
		newLeft := newCX - newW/2
		newTop := newCY - newH/2

		rect := xc.RECT{
			Left:   newLeft,
			Top:    newTop,
			Right:  newLeft + newW,
			Bottom: newTop + newH,
		}
		m.w.SetRect(&rect).Redraw(true)
		time.Sleep(time.Millisecond * 10)
	}

	// 动画结束后隐藏窗口
	m.hideWindow()
}

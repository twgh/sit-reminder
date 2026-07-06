package gui

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"gopkg.in/yaml.v3"
)

//go:embed dist/**
var embedAssets embed.FS

const hostName = "app.health"

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

// configPath 返回配置文件路径
func (m *MainWindow) configPath() string {
	confgDir := filepath.Join(os.Getenv("APPDATA"), g.AppName)
	return filepath.Join(confgDir, "config.yaml")
}

// loadConfig 加载配置文件
func (m *MainWindow) loadConfig() {
	path := m.configPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := os.ReadFile(path)
	if err != nil {
		m.config = config.NewAppConfig()
		return
	}
	cfg := config.NewAppConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		m.config = config.NewAppConfig()
		return
	}
	if cfg.IntervalMinutes < 1 {
		cfg.IntervalMinutes = 1
	}
	m.config = cfg
}

// saveConfig 保存配置文件
func (m *MainWindow) saveConfig() error {
	path := m.configPath()
	data, err := yaml.Marshal(m.config)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func NewMainWindow(edg *edge.Edge) *MainWindow {
	m := &MainWindow{
		edg:      edg,
		ap:       wutil.NewAudioPlayer(),
		ap2:      wutil.NewAudioPlayer(),
		config:   config.NewAppConfig(),
		done:     make(chan struct{}),
		pauseCh:  make(chan struct{}),
		resumeCh: make(chan struct{}),
		state:    StateIdle,
	}

	m.loadConfig()

	var err error
	m.w, m.wv, err = m.edg.NewWebViewWithWindow(
		edge.WithXmlWindowTitle("久坐提醒助手"),
		edge.WithXmlWindowClassName(g.AppName),
		edge.WithXmlWindowSize(480, 880),
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
	hAppIcon := wapi.LoadImageW(wapi.GetModuleHandleW(""), common.StrPtr("APPICON"), wapi.IMAGE_ICON, 0, 0, wapi.LR_SHARED|wapi.LR_DEFAULTSIZE)
	// 设置任务栏预览窗口左上角的图标
	m.w.SetSmallIcon(hAppIcon)

	// 禁止拖拽边框改变窗口大小
	m.w.EnableDragBorder(false)

	if !g.IsDebug() {
		m.setupEmbedFS()
	} else {
		m.setupDevServer()
	}

	m.regWebViewEvents()
	m.bindFunctions()
	m.wv.Navigate(m.getHost() + "/index.html")
	return m
}

// getHost 返回主机名
func (m *MainWindow) getHost() string {
	if g.IsDebug() {
		return "http://localhost:5173"
	}
	return edge.JoinUrlHeader(hostName)
}

// setupDevServer 输出 Vite 开发服务器
func (m *MainWindow) setupDevServer() {
	fmt.Println("开发模式: 连接 Vite 开发服务器 http://localhost:5173")
}

// setupEmbedFS 使用嵌入文件系统
func (m *MainWindow) setupEmbedFS() {
	fmt.Println("正式模式: 使用嵌入文件系统")
	err := edge.SetVirtualHostNameToEmbedFSMapping(hostName, embedAssets)
	if err != nil {
		wapi.MessageBoxW(0, "SetVirtualHostNameToEmbedFSMapping 失败: "+err.Error(), "错误", wapi.MB_OK|wapi.MB_IconError)
		os.Exit(5)
	}
	err = m.wv.EnableVirtualHostNameToEmbedFSMapping(true)
	if err != nil {
		wapi.MessageBoxW(0, "EnableVirtualHostNameToEmbedFSMapping 失败: "+err.Error(), "错误", wapi.MB_OK|wapi.MB_IconError)
		os.Exit(6)
	}
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
		m.startActivityTimer()
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
		if err := m.saveConfig(); err != nil {
			log.Println("保存配置失败:", err)
		}
		fmt.Println("配置已保存到", m.configPath())
	})
}

// ==================== 定时器 ====================

// startTimer 启动普通计时（暂停后恢复也用此方法）
func (m *MainWindow) startTimer(typ TimerType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == StateRunning {
		return
	}

	m.timerType = typ

	if m.state != StatePaused {
		m.total = m.config.IntervalMinutes * 60
		m.remaining = m.total
	}

	m.state = StateRunning
	m.done = make(chan struct{})

	go m.runTimerLoop()

	// 合并 __timerTick 和 __statusChanged 到一个 Eval，防止 React 中间态渲染
	script := fmt.Sprintf("(function(){window.__timerTick&&__timerTick(%d,%d);window.__statusChanged&&__statusChanged('running')})()", m.remaining, m.total)
	m.wv.Eval(script)
}

// startTimerWithTotal 以指定秒数启动计时（用于 snooze / activity）
func (m *MainWindow) startTimerWithTotal(typ TimerType, totalSec int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == StateRunning {
		return
	}

	m.timerType = typ
	m.total = totalSec
	m.remaining = totalSec
	m.state = StateRunning
	m.done = make(chan struct{})

	go m.runTimerLoop()

	script := fmt.Sprintf("(function(){window.__timerTick&&__timerTick(%d,%d);window.__statusChanged&&__statusChanged('running')})()", m.remaining, m.total)
	m.wv.Eval(script)
}

func (m *MainWindow) stopTimer() {
	m.mu.Lock()
	wasFinished := m.state == StateFinished
	m.mu.Unlock()

	if wasFinished {
		m.stopRingtone()
		m.stopActivityRingtone()
	}

	m.mu.Lock()
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}
	select {
	case m.done <- struct{}{}:
	default:
	}
	m.state = StateIdle
	m.timerType = TimerRegular
	m.total = m.config.IntervalMinutes * 60
	m.remaining = m.total
	m.mu.Unlock()

	xc.UI(func() {
		m.evalStatusChanged("idle")
		// 通知前端重置进度显示
		m.wv.Eval(fmt.Sprintf("window.__timerTick && __timerTick(%d, %d)", m.remaining, m.total))
	})
}

func (m *MainWindow) pauseTimer() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateRunning {
		return
	}
	m.state = StatePaused
	select {
	case m.pauseCh <- struct{}{}:
	default:
	}
	m.evalStatusChanged("paused")
}

func (m *MainWindow) resumeTimer() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StatePaused {
		return
	}
	m.state = StateRunning
	select {
	case m.resumeCh <- struct{}{}:
	default:
	}
	m.evalStatusChanged("running")
}

func (m *MainWindow) snooze(minutes int) {
	m.stopRingtone()
	m.stopActivityRingtone()

	m.mu.Lock()
	m.state = StateIdle
	if m.timer != nil {
		m.timer.Stop()
	}
	m.mu.Unlock()

	// 直接使用 startTimerWithTotal，不再依赖 snoozeMode 标志
	m.startTimerWithTotal(TimerSnooze, minutes*60)
}

func (m *MainWindow) startActivityTimer() {
	m.stopRingtone()

	m.mu.Lock()
	m.state = StateIdle
	if m.timer != nil {
		m.timer.Stop()
	}
	m.mu.Unlock()

	// 通知前端进入活动倒计时
	m.wv.Eval("window.__activityStarted && __activityStarted()")

	// 使用 startTimerWithTotal 启动5分钟活动倒计时
	m.startTimerWithTotal(TimerActivity, 5*60)
}

func (m *MainWindow) runTimerLoop() {
	m.mu.Lock()
	m.timer = time.NewTicker(1 * time.Second)
	done := m.done
	pauseCh := m.pauseCh
	resumeCh := m.resumeCh
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		if m.timer != nil {
			m.timer.Stop()
			m.timer = nil
		}
		m.mu.Unlock()
	}()

	for {
		select {
		case <-done:
			return

		case <-pauseCh:
			select {
			case <-resumeCh:
				continue
			case <-done:
				return
			}

		case <-m.timer.C:
			m.mu.Lock()
			if m.state != StateRunning {
				m.mu.Unlock()
				return
			}
			m.remaining--
			rem := m.remaining
			tot := m.total
			m.mu.Unlock()

			if rem <= 0 {
				m.mu.Lock()
				m.state = StateFinished
				if m.timer != nil {
					m.timer.Stop()
				}
				tt := m.timerType
				m.mu.Unlock()

				// 根据计时类型决定行为
				typeStr := "regular"
				switch tt {
				case TimerActivity:
					typeStr = "activity"
				case TimerSnooze:
					typeStr = "snooze"
				}

				xc.UI(func() {
					m.wv.Eval(fmt.Sprintf("window.__timerFinished && __timerFinished('%s')", typeStr))
					m.evalStatusChanged("finished")
					if tt == TimerActivity {
						m.playActivityRingtone()
					} else {
						m.playRingtone()
					}
				})
				return
			}

			xc.UI(func() {
				m.wv.Eval(fmt.Sprintf("window.__timerTick && __timerTick(%d, %d)", rem, tot))
			})
		}
	}
}

func (m *MainWindow) evalStatusChanged(status string) {
	m.wv.Eval(fmt.Sprintf("window.__statusChanged && __statusChanged('%s')", status))
}

// ==================== 提醒铃声 ====================

func (m *MainWindow) selectRingtone() string {
	path := wutil.OpenFile(0, []string{
		"音频文件(*.mp3;*.wma)", "*.mp3;*.wma",
	}, "%USERPROFILE%\\Music")
	if path == "" {
		return m.config.RingtonePath
	}
	m.config.RingtonePath = path
	return path
}

func (m *MainWindow) testRingtone() {
	path := m.config.RingtonePath
	if path == "" {
		log.Println("未选择提醒铃声文件")
		return
	}
	m.closeAudio()
	if err := m.ap.Open(path); err != nil {
		log.Printf("打开铃声文件失败: %v", err)
		return
	}
	_ = m.ap.Play(wutil.PlayOptions{Volume: new(800), Wait: false, Repeat: false, SeekToStart: true})
}

func (m *MainWindow) playRingtone() {
	path := m.config.RingtonePath
	if path == "" {
		log.Println("未选择提醒铃声，无法播放")
		return
	}
	m.closeAudio()
	if err := m.ap.Open(path); err != nil {
		log.Printf("打开铃声失败: %v", err)
		return
	}
	_ = m.ap.Play(wutil.PlayOptions{Volume: new(1000), Wait: false, Repeat: true, SeekToStart: true})
}

func (m *MainWindow) stopRingtone() {
	m.closeAudio()
}

func (m *MainWindow) closeAudio() {
	if m.ap.Alias == "" {
		return
	}
	if !m.ap.IsStopped() {
		_ = m.ap.Stop()
	}
	_ = m.ap.Close()
	m.ap = wutil.NewAudioPlayer()
}

// ==================== 活动结束铃声 ====================

func (m *MainWindow) selectActivityRingtone() string {
	path := wutil.OpenFile(0, []string{
		"音频文件(*.mp3;*.wav;*.aac;*.ogg;*.flac)", "*.mp3;*.wav;*.aac;*.ogg;*.flac",
		"All Files(*.*)", "*.*",
	}, "%USERPROFILE%\\Music")
	if path == "" {
		return m.config.ActivityRingtonePath
	}
	m.config.ActivityRingtonePath = path
	return path
}

func (m *MainWindow) testActivityRingtone() {
	path := m.config.ActivityRingtonePath
	if path == "" {
		log.Println("未选择活动结束铃声文件")
		return
	}
	m.closeActivityAudio()
	if err := m.ap2.Open(path); err != nil {
		log.Printf("打开活动铃声失败: %v", err)
		return
	}
	_ = m.ap2.Play(wutil.PlayOptions{Volume: new(800), Wait: false, Repeat: false, SeekToStart: true})
}

func (m *MainWindow) playActivityRingtone() {
	path := m.config.ActivityRingtonePath
	if path == "" {
		// 没有设置活动铃声则用提醒铃声
		m.playRingtone()
		return
	}
	m.closeActivityAudio()
	if err := m.ap2.Open(path); err != nil {
		log.Printf("打开活动铃声失败: %v", err)
		m.playRingtone()
		return
	}
	_ = m.ap2.Play(wutil.PlayOptions{Volume: new(1000), Wait: false, Repeat: true, SeekToStart: true})
}

func (m *MainWindow) stopActivityRingtone() {
	m.closeActivityAudio()
}

func (m *MainWindow) closeActivityAudio() {
	if m.ap2.Alias == "" {
		return
	}
	if !m.ap2.IsStopped() {
		_ = m.ap2.Stop()
	}
	_ = m.ap2.Close()
	m.ap2 = wutil.NewAudioPlayer()
}

func createEdge() *edge.Edge {
	edg, err := edge.New(edge.Option{
		UserDataFolder: filepath.Join(os.Getenv("APPDATA"), g.AppName),
		EnvOptions: &edge.EnvOptions{
			DisableTrackingPrevention: true,
			ScrollBarStyle:            edge.COREWEBVIEW2_SCROLLBAR_STYLE_FLUENT_OVERLAY,
		},
	})
	if err != nil {
		wapi.MessageBoxW(0, "创建 WebView 环境失败: "+err.Error(), "错误", wapi.MB_OK|wapi.MB_IconError)
		os.Exit(1)
	}
	return edg
}

func checkWebView2() {
	fmt.Println("本库使用的 WebView2 运行时版本号:", edge.GetVersion())
	localVersion, err := edge.GetAvailableBrowserVersion()
	if err != nil {
		wapi.MessageBoxW(0, "获取 WebView2 运行时版本号失败: "+err.Error(), "提示", wapi.MB_IconError)
		os.Exit(1)
	}
	if localVersion == "" {
		wapi.MessageBoxW(0, "请安装 WebView2 运行时后再打开程序!\n下载完后请使用管理员权限运行安装包!", "提示", wapi.MB_IconWarning|wapi.MB_OK)
		edge.DownloadWebView2()
		os.Exit(2)
	}
	fmt.Println("本机安装的 WebView2 运行时版本号:", localVersion)

	if ret, _ := edge.CompareBrowserVersions(localVersion, edge.GetVersion()); ret == -1 {
		log.Println("本机 WebView2 运行时版本低于本库使用的版本!")
	}
}

// ==================== 程序入口 ====================

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

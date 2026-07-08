package gui

import (
	"fmt"
	"time"

	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

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
	m.evalTimerTypeChanged(typ)
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
	m.evalTimerTypeChanged(typ)
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
	m.timerType = TimerSedentary
	m.total = m.config.IntervalMinutes * 60
	m.remaining = m.total
	m.mu.Unlock()

	xc.UI(func() {
		m.evalStatusChanged("idle")
		m.evalTimerTypeChanged(TimerSedentary)
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

// startActivityTimer 启动活动倒计时
func (m *MainWindow) startActivityTimer(minutes int) {
	m.stopRingtone()

	m.mu.Lock()
	m.state = StateIdle
	if m.timer != nil {
		m.timer.Stop()
	}
	m.mu.Unlock()

	// 通知前端进入活动倒计时
	m.wv.Eval(fmt.Sprintf("window.__activityStarted && __activityStarted(%d)", minutes))

	// 使用 startTimerWithTotal 启动活动倒计时
	m.startTimerWithTotal(TimerActivity, minutes*60)
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
				typeStr := "sedentary"
				switch tt {
				case TimerActivity:
					typeStr = "activity"
				case TimerSnooze:
					typeStr = "snooze"
				}

				xc.UI(func() {
					// 提醒时激活窗口
					m.activateWindow()

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

// timerTypeStr 将 TimerType 转换为前端可识别的字符串
func timerTypeStr(t TimerType) string {
	switch t {
	case TimerActivity:
		return "activity"
	case TimerSnooze:
		return "snooze"
	default:
		return "sedentary"
	}
}

// evalTimerTypeChanged 通知前端当前倒计时类型已改变
func (m *MainWindow) evalTimerTypeChanged(t TimerType) {
	m.wv.Eval(fmt.Sprintf("window.__timerTypeChanged && __timerTypeChanged('%s')", timerTypeStr(t)))
}

// activateWindow 激活窗口到前台, 如果 WebView 是在挂起状态, 会自动恢复
func (m *MainWindow) activateWindow() {
	m.w.SendMessage(wapi.WM_SIZE, wapi.SIZE_RESTORED, 0)
	m.w.ShowWindow(xcc.SW_RESTORE)
	m.wv.Show() // 显示 WebView
	// 恢复关闭按钮的 hover 样式: 移除 body 上的禁用 class
	m.wv.Eval(`document.body.classList.remove('close-hover-disabled')`)
}

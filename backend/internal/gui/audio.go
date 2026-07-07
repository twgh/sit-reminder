package gui

import (
	"log"

	"github.com/twgh/xcgui/wapi/wutil"
)

// ==================== 提醒铃声 ====================

func (m *MainWindow) selectRingtone() string {
	path := wutil.OpenFile(m.w.Handle, []string{
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
	_ = m.ap.Play(wutil.PlayOptions{Volume: new(m.config.Volume), Wait: false, Repeat: false, SeekToStart: true})
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
	_ = m.ap.Play(wutil.PlayOptions{Volume: new(m.config.Volume), Wait: false, Repeat: true, SeekToStart: true})
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
	path := wutil.OpenFile(m.w.Handle, []string{
		"音频文件(*.mp3;*.wma)", "*.mp3;*.wma",
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
	_ = m.ap2.Play(wutil.PlayOptions{Volume: new(m.config.Volume), Wait: false, Repeat: false, SeekToStart: true})
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
	_ = m.ap2.Play(wutil.PlayOptions{Volume: new(m.config.Volume), Wait: false, Repeat: true, SeekToStart: true})
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

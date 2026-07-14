package config

import (
	"os"
	"path/filepath"

	"github.com/twgh/sit-reminder/internal/g"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	IntervalMinutes      int    `yaml:"interval_minutes" json:"intervalMinutes"`            // 久坐提醒间隔分钟数
	RingtonePath         string `yaml:"ringtone_path" json:"ringtonePath"`                  // 久坐提醒铃声路径
	ActivityRingtonePath string `yaml:"activity_ringtone_path" json:"activityRingtonePath"` // 活动结束铃声路径
	ActivityMinutes      int    `yaml:"activity_minutes" json:"activityMinutes"`            // 活动倒计时分钟数
	Volume               int    `yaml:"volume" json:"volume"`                               // 播放音量, 0-1000
	AutoHide             bool   `yaml:"auto_hide" json:"autoHide"`                          // 开始提醒倒计时后隐藏窗口
	AlwaysOnTop          bool   `yaml:"always_on_top" json:"alwaysOnTop"`                   // 窗口总在最前
	ThemeMode            string `yaml:"theme_mode" json:"themeMode"`                        // 主题模式: "浅色" | "深色" | "跟随系统"
	AutoStart            bool   `yaml:"auto_start" json:"autoStart"`                        // 开机自启
	Hotkey               string `yaml:"hotkey" json:"hotkey"`                               // 全局呼出热键, 默认 "Ctrl+Shift+R"
}

func NewAppConfig() *AppConfig {
	return &AppConfig{
		IntervalMinutes:      40,
		RingtonePath:         "",
		ActivityRingtonePath: "",
		ActivityMinutes:      5,
		Volume:               1000,
		AlwaysOnTop:          true,
		ThemeMode:            "跟随系统",
		Hotkey:               "Ctrl+Shift+R",
	}
}

// ConfigPath 返回配置文件路径
func ConfigPath() string {
	configDir := filepath.Join(os.Getenv("APPDATA"), g.AppName)
	return filepath.Join(configDir, "config.yaml")
}

// LoadConfig 加载配置文件
func LoadConfig() *AppConfig {
	path := ConfigPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := os.ReadFile(path)
	if err != nil {
		return NewAppConfig()
	}
	cfg := NewAppConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return NewAppConfig()
	}
	if cfg.IntervalMinutes < 1 {
		cfg.IntervalMinutes = 1
	}
	if cfg.ActivityMinutes < 1 {
		cfg.ActivityMinutes = 1
	}
	if cfg.Volume < 0 {
		cfg.Volume = 0
	}
	if cfg.Volume > 1000 {
		cfg.Volume = 1000
	}
	return cfg
}

// SaveConfig 保存配置文件
func SaveConfig(cfg *AppConfig) error {
	path := ConfigPath()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

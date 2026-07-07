package config

import (
	"os"
	"path/filepath"

	"github.com/twgh/sit-reminder/internal/g"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	IntervalMinutes      int    `yaml:"interval_minutes" json:"intervalMinutes"`
	RingtonePath         string `yaml:"ringtone_path" json:"ringtonePath"`
	ActivityRingtonePath string `yaml:"activity_ringtone_path" json:"activityRingtonePath"`
}

func NewAppConfig() *AppConfig {
	return &AppConfig{
		IntervalMinutes:      40,
		RingtonePath:         "",
		ActivityRingtonePath: "",
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

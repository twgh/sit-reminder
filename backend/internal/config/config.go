package config

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

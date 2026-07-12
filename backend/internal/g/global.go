package g

// DebugState 调试状态, 1 为调试状态, 0 为生产状态
var DebugState = "1"

// Version 版本号
var Version = "2026.7.5.0"

// AppName 应用名称, 英文, 用于配置目录创建, 窗口类名
var AppName = "sit-reminder"

// IsDebug 是否是调试模式
func IsDebug() bool {
	return DebugState == "1"
}

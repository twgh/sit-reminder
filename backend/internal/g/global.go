package g

// DebugState 调试状态, 1 为调试状态, 0 为生产状态
var DebugState = "1"

var Version = "2026.7.5"

func IsDebug() bool {
	return DebugState == "1"
}

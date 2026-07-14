package utils

import (
	"fmt"
	"strings"

	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/xcc"
)

// ParseHotkey 解析热键字符串, 如 "Ctrl+Shift+R", 返回 (modifiers, vkCode).
func ParseHotkey(s string) (uint32, uint32, error) {
	var modifiers uint32
	var key string
	parts := strings.Split(s, "+")
	for _, part := range parts {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "ctrl":
			modifiers |= wapi.Mod_Control
		case "shift":
			modifiers |= wapi.Mod_Shift
		case "alt":
			modifiers |= wapi.Mod_Alt
		case "win":
			modifiers |= wapi.Mod_Win
		default:
			key = strings.TrimSpace(part)
		}
	}
	if len(key) == 0 {
		return 0, 0, fmt.Errorf("无效的热键: %s (缺少按键)", s)
	}
	// 单字母: A-Z => VK_A-VK_Z
	if len(key) == 1 {
		ch := key[0]
		if ch >= 'a' && ch <= 'z' {
			return modifiers, uint32(ch - 'a' + 'A'), nil // VK_A=0x41
		}
		if ch >= 'A' && ch <= 'Z' {
			return modifiers, uint32(ch), nil
		}
		if ch >= '0' && ch <= '9' {
			return modifiers, uint32(ch), nil // VK_0=0x30
		}
	}
	// F1-F12
	if len(key) >= 2 && (key[0] == 'F' || key[0] == 'f') {
		fn := strings.TrimPrefix(key, "F")
		fn = strings.TrimPrefix(fn, "f")
		switch fn {
		case "1":
			return modifiers, uint32(xcc.VK_F1), nil
		case "2":
			return modifiers, uint32(xcc.VK_F2), nil
		case "3":
			return modifiers, uint32(xcc.VK_F3), nil
		case "4":
			return modifiers, uint32(xcc.VK_F4), nil
		case "5":
			return modifiers, uint32(xcc.VK_F5), nil
		case "6":
			return modifiers, uint32(xcc.VK_F6), nil
		case "7":
			return modifiers, uint32(xcc.VK_F7), nil
		case "8":
			return modifiers, uint32(xcc.VK_F8), nil
		case "9":
			return modifiers, uint32(xcc.VK_F9), nil
		case "10":
			return modifiers, uint32(xcc.VK_F10), nil
		case "11":
			return modifiers, uint32(xcc.VK_F11), nil
		case "12":
			return modifiers, uint32(xcc.VK_F12), nil
		}
	}
	return 0, 0, fmt.Errorf("无效的热键按键: %s", key)
}

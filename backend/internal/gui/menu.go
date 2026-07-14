package gui

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// 菜单项ID
const (
	menuItemSetting = iota + 1

	menuItemExit = 99999
)

// 显示托盘菜单
//
// darkMode: 是否为深色模式, 默认为 false.
func (m *MainWindow) showTrayMenu(darkMode ...bool) int {
	mode := false
	if len(darkMode) > 0 {
		mode = darkMode[0]
	}
	// 创建菜单. Choose 类似于三元运算符, 根据模式选择函数
	menu := common.Choose(mode, NewMenuExDark, NewMenuEx)(m.w.Handle)

	// 菜单项选择事件
	menu.AddEvent_Menu_Select(m.w.Handle, func(hWindow int, nID int32, pbHandled *bool) int {
		switch nID {
		case menuItemSetting: // 设置
			m.activateWindow()
			m.wv.Eval("window.__showSettings && window.__showSettings()")

		case menuItemExit: // 退出
			m.w.DestroyWindow() // 销毁窗口
			app.PostQuitMessage(0)
		}
		return 0
	}, false) // 这里填 false, 不然每次显示菜单都会加一个回调函数

	// 一级菜单
	menu.AddItem(menuItemSetting, "设置", 0, xcc.Menu_Item_Flag_Normal)

	menu.AddSeparator() // 分隔栏

	menu.AddItem(menuItemExit, "退出", 0, xcc.Menu_Item_Flag_Normal)

	// 获取鼠标光标的位置
	var pt wapi.POINT
	wapi.GetCursorPos(&pt)
	// 弹出菜单
	menu.Popup(xc.XWnd_GetHWND(m.w.Handle), pt.X, pt.Y, 0, xcc.Menu_Popup_Position_Left_Top)
	return 0
}

// MenuEx 美化菜单.
type MenuEx struct {
	*widget.Menu
	style *Style
}

// NewMenuEx 创建一个使用默认样式的自绘菜单实例.
//   - 等价于 NewMenuExWithStyle(NewStyle()).
//
// hWindowOrhEle: 窗口或元素句柄.
func NewMenuEx(hWindowOrhEle int) *MenuEx {
	return NewMenuExWithStyle(hWindowOrhEle, NewStyle())
}

// NewMenuExDark 创建一个深色风格的自绘菜单实例.
//
// hWindowOrhEle: 窗口或元素句柄.
func NewMenuExDark(hWindowOrhEle int) *MenuEx {
	return NewMenuExWithStyle(hWindowOrhEle, DarkStyle())
}

// NewMenuExWithStyle 使用自定义样式创建一个自绘菜单实例.
//
// hWindowOrhEle: 窗口或元素句柄.
func NewMenuExWithStyle(hWindowOrhEle int, style *Style) *MenuEx {
	m := widget.NewMenu()
	if style == nil {
		style = NewStyle()
	}
	if style.ItemHeight > 0 {
		m.SetItemHeight(style.ItemHeight)
	}
	m.EnableDrawBackground(true).EnableDrawItem(true)

	mx := &MenuEx{
		Menu:  m,
		style: style,
	}
	mx.bindMenuEvent(hWindowOrhEle)
	return mx
}

// GetStyle 返回当前菜单的样式, 可在添加菜单项前后修改.
func (mx *MenuEx) GetStyle() *Style {
	return mx.style
}

// SetStyle 替换整个样式表.
//   - 注意: ItemHeight 不会被回写, 因为该值已在构造时同步给原生 Menu.
func (mx *MenuEx) SetStyle(style *Style) *MenuEx {
	if style == nil {
		style = NewStyle()
	}
	mx.style = style
	return mx
}

// AddItemSvg 是 AddItemIcon 的便捷封装, 内部用 xc.XImage_LoadSvgString() 把 SVG 字符串转成图片句柄.
//
// nID: 项ID.
//
// text: 文本内容.
//
// nParentID: 父项ID.
//
// svg: SVG 字符串.
//
// nFlags: 标识, Menu_Item_Flag_.
func (mx *MenuEx) AddItemSvg(nID int32, text string, nParentID int32, svg string, nFlags xcc.Menu_Item_Flag_) *MenuEx {
	hIcon := xc.XImage_LoadSvgString(svg)
	mx.AddItemIcon(nID, text, nParentID, hIcon, nFlags)
	return mx
}

// AddSeparator 在指定父项下添加一个分隔栏. nParentID 为空时添加到根.
//
// nParentID: 父项ID, 不填默认为 0.
func (mx *MenuEx) AddSeparator(nParentID ...int32) *MenuEx {
	pid := int32(0)
	if len(nParentID) > 0 {
		pid = nParentID[0]
	}
	mx.Menu.AddItem(-1, "", pid, xcc.Menu_Item_Flag_Separator)
	return mx
}

// bindMenuEvent 将菜单绘制事件绑定到指定窗口或元素上.
//
// hWindowOrhEle: 窗口或元素句柄.
func (mx *MenuEx) bindMenuEvent(hWindowOrhEle int) *MenuEx {
	// 背景绘制
	mx.Menu.AddEvent_Menu_Draw_Background(hWindowOrhEle, func(hWindowOrhEle, hDraw int, pInfo *xc.Menu_DrawBackground_, pbHandled *bool) int {
		*pbHandled = true
		var rc xc.RECT
		xc.XWnd_GetClientRect(pInfo.HWindow, &rc)
		// 绘制菜单背景
		xc.XDraw_SetBrushColor(hDraw, mx.style.BackgroundColor)
		xc.XDraw_FillRect(hDraw, &rc)
		// 绘制菜单边框
		xc.XDraw_SetBrushColor(hDraw, mx.style.BorderColor)
		xc.XDraw_DrawRect(hDraw, &rc)
		return 0
	}, false)
	// 菜单项绘制
	mx.Menu.AddEvent_Menu_DrawItem(hWindowOrhEle, func(hWindowOrhEle, hDraw int, pInfo *xc.Menu_DrawItem_, pbHandled *bool) int {
		*pbHandled = true
		mx.drawItem(hDraw, pInfo)
		return 0
	}, false)
	return mx
}

// drawItem 菜单项自绘逻辑.
func (mx *MenuEx) drawItem(hDraw int, pInfo *xc.Menu_DrawItem_) {
	// 分割栏
	if pInfo.NState&xcc.MenuItem_State_Flag_Separator > 0 {
		xc.XDraw_SetBrushColor(hDraw, mx.style.SeparatorColor)
		xc.XDraw_DrawLine(hDraw, pInfo.RcItem.Left+3, pInfo.RcItem.Top+1, pInfo.RcItem.Right-3, pInfo.RcItem.Top+1)
		return
	}
	// 鼠标停留时菜单项的背景
	if pInfo.NState&xcc.MenuItem_State_Flag_Stay > 0 {
		rc := xc.RECT{
			Left:   pInfo.RcItem.Left + 1,
			Top:    pInfo.RcItem.Top,
			Right:  pInfo.RcItem.Right - 1,
			Bottom: pInfo.RcItem.Bottom - 1,
		}
		xc.XDraw_SetBrushColor(hDraw, mx.style.HoverColor)
		xc.XDraw_FillRect(hDraw, &rc)
	}
	// 三角形(指向子菜单)
	if pInfo.NState&xcc.MenuItem_State_Flag_Popup > 0 {
		var pt [3]xc.POINT
		pt[0].X = pInfo.RcItem.Right - 12
		pt[0].Y = pInfo.RcItem.Top + 10
		pt[1].X = pInfo.RcItem.Right - 12
		pt[1].Y = pInfo.RcItem.Top + 20
		pt[2].X = pInfo.RcItem.Right - 7
		pt[2].Y = pInfo.RcItem.Top + 15
		xc.XDraw_SetBrushColor(hDraw, mx.style.SubMenuArrowColor)
		xc.XDraw_FillPolygon(hDraw, pt[:])
	}
	// 文本
	leftWidth := xc.XMenu_GetLeftWidth(pInfo.HMenu)
	rc := pInfo.RcItem
	rc.Left = leftWidth + 5
	if pInfo.NState&xcc.MenuItem_State_Flag_Disable > 0 {
		xc.XDraw_SetBrushColor(hDraw, mx.style.TextDisabledColor)
	} else {
		xc.XDraw_SetBrushColor(hDraw, mx.style.TextColor)
	}
	text := common.UintPtrToString(pInfo.PText)
	xc.XDraw_SetTextAlign(hDraw, xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
	xc.XDraw_DrawText(hDraw, text, &rc)
	// 图标
	if pInfo.HIcon > 0 {
		iconWidth := xc.XImage_GetHeight(pInfo.HIcon)
		iconHeight := xc.XImage_GetWidth(pInfo.HIcon)
		height := pInfo.RcItem.Bottom - pInfo.RcItem.Top
		if height >= 2 && iconWidth >= 2 && iconHeight >= 2 {
			top := (height - iconHeight) / 2
			left := (leftWidth - iconWidth) / 2
			if top < 0 {
				top = 0
			}
			if left < 0 {
				left = 0
			}
			xc.XDraw_Image(hDraw, pInfo.HIcon, left, pInfo.RcItem.Top+top)
		}
	}
}

// Style 自绘菜单的样式配置.
//
// 通过 NewStyle() 创建一个带默认值的样式实例, 然后按需修改其中的字段即可.
//
// 颜色字段默认值: 文字偏深灰, 背景纯白, 边框淡灰.
type Style struct {
	// 整菜单项高度(像素). 0 表示使用菜单内部默认值.
	ItemHeight int32

	// 整菜单项文本显示区域宽度(像素), 不含左侧图标区. 0 表示不设置.
	// 如果你需要统一多个菜单项的宽度, 可在添加项后使用 menu.SetItemWidth(id, w) 单独设置.
	ItemWidth int32

	// 背景填充色.
	BackgroundColor uint32

	// 边框颜色.
	BorderColor uint32

	// 鼠标停留时菜单项的背景填充色.
	HoverColor uint32

	// 分割栏颜色.
	SeparatorColor uint32

	// 菜单项文本颜色(未禁用时).
	TextColor uint32

	// 菜单项文本颜色(禁用时).
	TextDisabledColor uint32

	// 指向子菜单的三角形的颜色.
	SubMenuArrowColor uint32
}

// NewStyle 返回一个使用默认颜色/样式的 Style.
func NewStyle() *Style {
	return &Style{
		ItemHeight:        30,
		ItemWidth:         0,
		BackgroundColor:   xc.RGBA(255, 255, 255, 255),
		BorderColor:       xc.RGBA(218, 220, 224, 255),
		HoverColor:        xc.RGBA(230, 230, 230, 255),
		SeparatorColor:    xc.RGBA(218, 220, 224, 255),
		TextColor:         xc.RGBA(77, 77, 77, 255),
		TextDisabledColor: xc.RGBA(160, 160, 160, 255),
		SubMenuArrowColor: xc.RGBA(130, 130, 130, 255),
	}
}

// DarkStyle 返回一个深色风格的样式.
func DarkStyle() *Style {
	return &Style{
		ItemHeight:        30,
		BackgroundColor:   xc.RGBA(43, 43, 43, 255),
		BorderColor:       xc.RGBA(70, 70, 70, 255),
		HoverColor:        xc.RGBA(70, 70, 70, 255),
		SeparatorColor:    xc.RGBA(70, 70, 70, 255),
		TextColor:         xc.RGBA(220, 220, 220, 255),
		TextDisabledColor: xc.RGBA(120, 120, 120, 255),
		SubMenuArrowColor: xc.RGBA(200, 200, 200, 255),
	}
}

// SetItemHeight 设置菜单项高度.
func (s *Style) SetItemHeight(h int32) *Style {
	s.ItemHeight = h
	return s
}

// SetItemWidth 设置菜单项文本区域宽度, 0 表示不设置.
func (s *Style) SetItemWidth(w int32) *Style {
	s.ItemWidth = w
	return s
}

// SetBackgroundColor 设置背景填充色.
func (s *Style) SetBackgroundColor(c uint32) *Style {
	s.BackgroundColor = c
	return s
}

// SetBorderColor 设置边框颜色.
func (s *Style) SetBorderColor(c uint32) *Style {
	s.BorderColor = c
	return s
}

// SetHoverColor 设置鼠标停留时的背景色.
func (s *Style) SetHoverColor(c uint32) *Style {
	s.HoverColor = c
	return s
}

// SetSeparatorColor 设置分割栏颜色.
func (s *Style) SetSeparatorColor(c uint32) *Style {
	s.SeparatorColor = c
	return s
}

// SetTextColor 设置菜单项文本颜色(未禁用时).
func (s *Style) SetTextColor(c uint32) *Style {
	s.TextColor = c
	return s
}

// SetTextDisabledColor 设置菜单项禁用时的文本颜色.
func (s *Style) SetTextDisabledColor(c uint32) *Style {
	s.TextDisabledColor = c
	return s
}

// SetSubMenuArrowColor 设置指向子菜单的三角形的颜色.
func (s *Style) SetSubMenuArrowColor(c uint32) *Style {
	s.SubMenuArrowColor = c
	return s
}

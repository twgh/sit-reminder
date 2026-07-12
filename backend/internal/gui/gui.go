package gui

import "github.com/twgh/xcgui/app"

// Run 程序入口
func Run(hide bool) {
	checkWebView2()
	edg := createEdge()

	app.InitOrExit()
	a := app.New(true)
	a.EnableAutoDPI(true).EnableDPI(true)

	NewMainWindow(edg, hide)

	a.Run()
	a.Exit()
}

package main

import (
	"flag"

	"github.com/twgh/sit-reminder/internal/gui"
)

func main() {
	hide := flag.Bool("hide", false, "启动后不显示窗口（配合开机自启使用）")
	flag.Parse()
	gui.Run(*hide)
}

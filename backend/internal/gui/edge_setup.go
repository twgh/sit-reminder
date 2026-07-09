package gui

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/twgh/sit-reminder/internal/g"
	"github.com/twgh/xcgui/edge"
	"github.com/twgh/xcgui/wapi"
)

//go:embed dist/**
var embedAssets embed.FS

const hostName = "app.health"

// getHost 返回主机名
func (m *MainWindow) getHost() string {
	if g.IsDebug() {
		return "http://localhost:5173"
	}
	return edge.JoinUrlHeader(hostName)
}

// setupDevServer 输出 Vite 开发服务器
func (m *MainWindow) setupDevServer() {
	fmt.Println("开发模式: 连接 Vite 开发服务器 http://localhost:5173")
}

// setupEmbedFS 使用嵌入文件系统
func (m *MainWindow) setupEmbedFS() {
	fmt.Println("正式模式: 使用嵌入文件系统")
	err := edge.SetVirtualHostNameToEmbedFSMapping(hostName, embedAssets)
	if err != nil {
		wapi.MessageBoxW(0, "SetVirtualHostNameToEmbedFSMapping 失败: "+err.Error(), "错误", wapi.MB_OK|wapi.MB_IconError)
		os.Exit(5)
	}
	err = m.wv.EnableVirtualHostNameToEmbedFSMapping(true)
	if err != nil {
		wapi.MessageBoxW(0, "EnableVirtualHostNameToEmbedFSMapping 失败: "+err.Error(), "错误", wapi.MB_OK|wapi.MB_IconError)
		os.Exit(6)
	}
}

// saveMemory 节省内存
func (m *MainWindow) saveMemory() {
	// 设置内存使用目标级别，能节省近一半占用内存
	m.wv.WebView2_19 = m.wv.CoreWebView.MustGetICoreWebView2_19()
	if m.wv.WebView2_19 != nil {
		m.wv.WebView2_19.SetMemoryUsageTargetLevel(edge.COREWEBVIEW2_MEMORY_USAGE_TARGET_LEVEL_LOW)
	}
}

func createEdge() *edge.Edge {
	edg, err := edge.New(edge.Option{
		UserDataFolder: filepath.Join(os.Getenv("APPDATA"), g.AppName),
		EnvOptions: &edge.EnvOptions{
			DisableTrackingPrevention: true,
			ScrollBarStyle:            edge.COREWEBVIEW2_SCROLLBAR_STYLE_FLUENT_OVERLAY,
		},
	})
	if err != nil {
		wapi.MessageBoxW(0, "创建 WebView 环境失败: "+err.Error(), "错误", wapi.MB_OK|wapi.MB_IconError)
		os.Exit(1)
	}
	return edg
}

func checkWebView2() {
	fmt.Println("本库使用的 WebView2 运行时版本号:", edge.GetVersion())
	localVersion, err := edge.GetAvailableBrowserVersion()
	if err != nil {
		wapi.MessageBoxW(0, "获取 WebView2 运行时版本号失败: "+err.Error(), "提示", wapi.MB_IconError)
		os.Exit(1)
	}
	if localVersion == "" {
		wapi.MessageBoxW(0, "请安装 WebView2 运行时后再打开程序!\n下载完后请使用管理员权限运行安装包!", "提示", wapi.MB_IconWarning|wapi.MB_OK)
		edge.DownloadWebView2()
		os.Exit(2)
	}
	fmt.Println("本机安装的 WebView2 运行时版本号:", localVersion)

	if ret, _ := edge.CompareBrowserVersions(localVersion, edge.GetVersion()); ret == -1 {
		log.Println("本机 WebView2 运行时版本低于本库使用的版本!")
	}
}

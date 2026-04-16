package main

import (
	"embed"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	// 解析命令行参数
	// 右键菜单: NekoArc.exe --pack "C:\path"
	// 右键菜单: NekoArc.exe --extract "C:\path\file.nya"
	// 右键菜单: NekoArc.exe --repair "C:\path\file.nya"
	// 双击文件: NekoArc.exe "C:\path\file.nya"
	for i, arg := range os.Args[1:] {
		switch arg {
		case "--pack":
			if i+2 < len(os.Args) {
				app.startupAction = "pack"
				app.startupFile = os.Args[i+2]
			}
		case "--extract":
			if i+2 < len(os.Args) {
				app.startupAction = "extract"
				app.startupFile = os.Args[i+2]
			}
		case "--repair":
			if i+2 < len(os.Args) {
				app.startupAction = "repair"
				app.startupFile = os.Args[i+2]
			}
		default:
			if !strings.HasPrefix(arg, "-") && app.startupFile == "" {
				app.startupFile = arg
				app.startupAction = "open"
			}
		}
	}

	err := wails.Run(&options.App{
		Title:     "NekoArc",
		Width:     900,
		Height:    650,
		MinWidth:  700,
		MinHeight: 500,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 3, G: 7, B: 18, A: 1},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}

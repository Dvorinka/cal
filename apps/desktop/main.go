//go:build !headless

package main

import (
	"context"
	_ "embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/appicon.png
var appIcon []byte

func main() {
	// Window opens instantly on a setup screen; the DB/API attach behind it.
	b := newBackend()

	var appCtx context.Context
	err := wails.Run(&options.App{
		Title:  "Cal",
		Width:  1280,
		Height: 820,
		AssetServer: &assetserver.Options{
			Handler: b,
		},
		BackgroundColour: &options.RGBA{R: 21, G: 21, B: 19, A: 255},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "cal-desktop-6d3f8a2e",
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				if appCtx != nil {
					wruntime.WindowUnminimise(appCtx)
					wruntime.Show(appCtx)
				}
			},
		},
		OnStartup: func(ctx context.Context) {
			appCtx = ctx
		},
		OnShutdown: func(_ context.Context) {
			b.Close()
		},
		// ProgramName sets the GTK program name → WM_CLASS "cal", which is what
		// cal.desktop's StartupWMClass matches against for the launcher icon.
		// Note: a non-nil Linux block stops wails forcing WebviewGpuPolicyNever.
		Linux: &linux.Options{
			Icon:             appIcon,
			ProgramName:      "cal",
			WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

//go:build !headless

package main

import (
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	handler, closeDB, err := NewHandler()
	if err != nil {
		log.Fatalf("cal desktop: %v", err)
	}
	defer closeDB()

	err = wails.Run(&options.App{
		Title:  "Cal",
		Width:  1280,
		Height: 820,
		AssetServer: &assetserver.Options{
			Handler: handler,
		},
		BackgroundColour: &options.RGBA{R: 21, G: 21, B: 19, A: 255},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "cal-desktop-6d3f8a2e",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

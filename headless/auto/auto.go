//go:build darwin || windows || linux

// Package auto, imported for its effect, makes an application's RunCanvas serve
// the headless line protocol instead of opening a window whenever the process
// is started with VIVA_HEADLESS set. Import it from a file guarded by a build
// tag so release builds never contain it:
//
//	//go:build vivahead
//
//	package main
//
//	import _ "github.com/vivaui/viva/headless/auto"
//
// and run the tagged build with VIVA_HEADLESS=1.
package auto

import (
	"os"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/headless"
)

func init() {
	if os.Getenv("VIVA_HEADLESS") != "" {
		app.UseRunner(headless.Run)
		app.UseApplicationRunner(headless.RunApplication)
	}
}

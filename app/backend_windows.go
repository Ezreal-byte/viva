package app

import (
	"github.com/vivaui/viva/internal/platform"
	"github.com/vivaui/viva/internal/platform/windows"
)

func runNative(config platform.Config) error { return windows.Run(config) }
func runNativeApplication(start platform.ApplicationStart) error {
	return windows.RunApplication(start)
}

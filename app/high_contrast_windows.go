//go:build windows

package app

import windowsplatform "github.com/vivaui/viva/internal/platform/windows"

func nativeHighContrast() bool { return windowsplatform.HighContrast() }

//go:build windows

package main

import (
	"github.com/vivaui/viva/internal/platform/windows"
	"github.com/vivaui/viva/text"
	"github.com/vivaui/viva/ui"
)

const platformName = "windows"

func nativeMeasurer() (text.MeasureFunc, func(), error) { return windows.NewTextMeasurer() }

func nativeMono(size float32, face ui.MonoFace) (float32, float32, error) {
	return windows.MeasureMono(size, face)
}

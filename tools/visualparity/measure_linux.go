//go:build linux

package main

import (
	"github.com/vivaui/viva/internal/platform/linux"
	"github.com/vivaui/viva/text"
	"github.com/vivaui/viva/ui"
)

const platformName = "linux"

func nativeMeasurer() (text.MeasureFunc, func(), error) { return linux.NewTextMeasurer() }

func nativeMono(size float32, face ui.MonoFace) (float32, float32, error) {
	return linux.MeasureMono(size, face)
}

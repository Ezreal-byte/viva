package gfx

import "testing"

func TestTextScaleKeepsExplicitAndDefaultTypographyInStep(t *testing.T) {
	theme := LightTheme()
	scaled := theme.WithTextScale(1.5)
	if scaled.Metrics.FontSizeM != theme.Metrics.FontSizeM*1.5 || scaled.Metrics.EffectiveTextScale() != 1.5 {
		t.Fatal("body and explicit scale diverged")
	}
	reset := scaled.WithTextScale(1)
	if reset.Metrics != theme.WithTextScale(1).Metrics {
		t.Fatal("text scaling compounded")
	}
	if scaled.Metrics.ControlHeight != theme.Metrics.ControlHeight {
		t.Fatal("text scaling changed controls")
	}
}

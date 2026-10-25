package app

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

func TestPanelCornerRadiusMatchesTheDarwinPanel(t *testing.T) {
	source, err := os.ReadFile("../internal/platform/darwin/native.m")
	if os.IsNotExist(err) {
		t.Skip("the Darwin source is not beside this test binary")
	}
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`#define VIVA_PANEL_CORNER_RADIUS ([0-9.]+)`).FindSubmatch(source)
	if match == nil {
		t.Fatal("VIVA_PANEL_CORNER_RADIUS is not defined in native.m")
	}
	native, err := strconv.ParseFloat(string(match[1]), 32)
	if err != nil || float32(native) != panelCornerRadius {
		t.Fatalf("native.m rounds panels to %s, app says %d", match[1], panelCornerRadius)
	}
}

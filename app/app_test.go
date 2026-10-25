package app

import (
	"testing"
)

func TestConfigNormalizeDefaults(t *testing.T) {
	config, err := (Config{}).normalize()
	if err != nil {
		t.Fatal(err)
	}
	if config.Title != defaultTitle || config.Width != defaultWidth || config.Height != defaultHeight {
		t.Fatalf("unexpected defaults: %+v", config)
	}
	if config.ClearColor.Alpha != 1 {
		t.Fatalf("default color is not opaque: %+v", config.ClearColor)
	}
	if config.WindowMode != WindowTitled {
		t.Fatalf("default window mode = %d", config.WindowMode)
	}
}

func TestWindowModeValuesStayStable(t *testing.T) {
	if WindowTitled != 0 || WindowFrameless != 1 || WindowCustomTitlebar != 2 {
		t.Fatal("window mode constants must stay stable for the native ABI")
	}
}

func TestConfigNormalizeRejectsInvalidValues(t *testing.T) {
	tests := []Config{
		{Width: -1},
		{Height: -1},
		{ClearColor: Color{Red: 2, Alpha: 1}},
	}
	for _, config := range tests {
		if _, err := config.normalize(); err == nil {
			t.Fatalf("normalize accepted invalid config: %+v", config)
		}
	}
}

func TestSystemIntegrationValidation(t *testing.T) {
	if err := SetClipboardText("bad\x00text"); err == nil {
		t.Fatal("clipboard accepted NUL")
	}
	if err := OpenURL("file:///tmp/example"); err == nil {
		t.Fatal("OpenURL accepted file scheme")
	}
	if _, err := OpenFiles(OpenFileOptions{Filters: []FileFilter{{Name: "Text"}}}); err == nil {
		t.Fatal("OpenFiles accepted incomplete filter")
	}
	if err := SendNotification(Notification{}); err == nil {
		t.Fatal("SendNotification accepted empty notification")
	}
}

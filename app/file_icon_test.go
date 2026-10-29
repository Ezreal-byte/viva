package app

import (
	"strings"
	"testing"
)

func TestFileIconRejectsRequestsItCannotAnswer(t *testing.T) {
	for _, test := range []struct {
		name, path string
		pixels     int
		want       string
	}{
		{"relative path", "Safari.app", 32, "not absolute"},
		{"NUL in path", "/Applications/Safari\x00.app", 32, "NUL"},
		{"zero size", "/Applications/Safari.app", 0, "outside"},
		{"negative size", "/Applications/Safari.app", -1, "outside"},
		{"huge size", "/Applications/Safari.app", MaxFileIconPixels + 1, "outside"},
	} {
		if _, err := FileIcon(test.path, test.pixels); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%s: err = %v, want one containing %q", test.name, err, test.want)
		}
	}
}

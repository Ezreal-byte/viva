//go:build windows

package windows

import (
	"reflect"
	"syscall"
	"testing"
)

func TestMultiStringSplitsAtZerosAndStopsAtTheEmptyOne(t *testing.T) {
	utf16 := func(values ...string) []uint16 {
		var buffer []uint16
		for _, value := range values {
			encoded, _ := syscall.UTF16FromString(value)
			buffer = append(buffer, encoded...)
		}
		return append(buffer, 0)
	}
	if got, want := splitMultiString(utf16("zh-CN", "en-US")), []string{"zh-CN", "en-US"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got := splitMultiString(nil); got != nil {
		t.Fatalf("an empty buffer has no names: %v", got)
	}
}

func TestPreferredLanguagesAnswersOnThisMachine(t *testing.T) {
	for _, name := range PreferredLanguages() {
		if name == "" {
			t.Fatal("an empty language name")
		}
	}
}

func TestDarkAppearanceAnswersOnThisMachine(t *testing.T) {
	dark, known := DarkAppearance()
	if dark && !known {
		t.Fatal("a machine that does not say cannot be dark")
	}
	t.Logf("registry: dark=%v known=%v", dark, known)
}

//go:build darwin || windows || linux

package singleinstance

import (
	"os"
	"testing"
	"time"
)

func TestSingleInstanceHandsLaterLaunchesToTheFirst(t *testing.T) {
	dir, err := os.MkdirTemp("", "gl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	t.Setenv("XDG_RUNTIME_DIR", dir)
	t.Setenv("TMPDIR", dir)
	t.Setenv("TEMP", dir)
	t.Setenv("TMP", dir)
	got := make(chan []string, 1)
	claim, primary, err := SingleInstance("dev.viva.test.launch", nil, func(args []string) { got <- args })
	if err != nil || !primary {
		t.Fatalf("primary %v err %v", primary, err)
	}
	defer claim.Close()
	other, primary, err := SingleInstance("dev.viva.test.launch", []string{"ssh://h/"}, func([]string) {})
	if err != nil || primary || other != nil {
		t.Fatalf("second: primary %v err %v", primary, err)
	}
	select {
	case args := <-got:
		if len(args) != 1 || args[0] != "ssh://h/" {
			t.Fatalf("args %q", args)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("not delivered")
	}
	var none *Instance
	none.Close()
	if _, _, err := SingleInstance("x", nil, nil); err == nil {
		t.Fatal("a nil handler was accepted")
	}
}

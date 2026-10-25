package autostart

import "testing"

func TestRunValueRoundTrip(t *testing.T) {
	entry := Entry{ID: "dev.viva.autostart-test", Executable: `C:\Program Files\Viva Test\test.exe`, Args: []string{"--background", "a b"}}
	t.Cleanup(func() { Set(entry, false) })
	if on, err := Enabled(entry.ID); on || err != nil {
		t.Fatalf("before: %v %v", on, err)
	}
	if err := Set(entry, true); err != nil {
		t.Fatal(err)
	}
	if on, err := Enabled(entry.ID); !on || err != nil {
		t.Fatalf("after enabling: %v %v", on, err)
	}
	if got := commandLine(entry); got != `"C:\Program Files\Viva Test\test.exe" --background "a b"` {
		t.Fatalf("command line %s", got)
	}
	if err := Set(entry, false); err != nil {
		t.Fatal(err)
	}
	if on, _ := Enabled(entry.ID); on {
		t.Fatal("still enabled")
	}
	if err := Set(entry, false); err != nil {
		t.Fatalf("disabling twice: %v", err)
	}
}

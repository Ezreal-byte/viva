package autostart

import "testing"

func TestInvalidEntriesAreRefused(t *testing.T) {
	for _, entry := range []Entry{
		{ID: "", Executable: "/bin/x"},
		{ID: "a/b", Executable: "/bin/x"},
		{ID: "ok", Executable: ""},
		{ID: "ok", Executable: "/bin/x", Args: []string{"a\nb"}},
	} {
		if err := Set(entry, true); err == nil {
			t.Errorf("%+v was accepted", entry)
		}
	}
	if _, err := Enabled("bad/id"); err == nil {
		t.Error("a bad id was queried")
	}
}

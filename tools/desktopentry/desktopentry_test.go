package desktopentry

import (
	"strings"
	"testing"
)

func TestEscape(t *testing.T) {
	for _, test := range []struct{ in, want string }{
		{"plain", "plain"},
		{`a\b`, `a\\b`},
		{"two\nlines", `two\nlines`},
		{"tab\tcr\r", `tab\tcr\r`},
		{" lead", `\slead`},
		{"in side", "in side"},
		{"a;b", "a;b"},
	} {
		if got := Escape(test.in); got != test.want {
			t.Errorf("Escape(%q) = %q, want %q", test.in, got, test.want)
		}
	}
}

func TestEscapeList(t *testing.T) {
	for _, test := range []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{"Utility"}, "Utility;"},
		{[]string{"a", "b"}, "a;b;"},
		{[]string{"semi;colon", `back\slash`}, `semi\;colon;back\\slash;`},
	} {
		if got := EscapeList(test.in); got != test.want {
			t.Errorf("EscapeList(%q) = %q, want %q", test.in, got, test.want)
		}
	}
}

func TestQuoteExec(t *testing.T) {
	for _, test := range []struct{ in, want string }{
		{"/usr/bin/app", "/usr/bin/app"},
		{"app", "app"},
		{"/home/a b/bin/app", `"/home/a b/bin/app"`},
		{`/x/$y`, `"/x/\$y"`},
		{`/x/"y`, `"/x/\"y"`},
	} {
		if got := QuoteExec(test.in); got != test.want {
			t.Errorf("QuoteExec(%q) = %q, want %q", test.in, got, test.want)
		}
	}
	// Through the file the backslash is doubled again, as the spec says.
	data, err := Entry{Name: "N", Exec: QuoteExec(`/x/$y z`)}.Bytes()
	if err != nil || !strings.Contains(string(data), `Exec="/x/\\$y z"`+"\n") {
		t.Errorf("%v\n%s", err, data)
	}
}

func TestEntryBytes(t *testing.T) {
	data, err := Entry{
		Name: "Fixture Notes", Comment: "Takes notes;\nsecond line", Exec: "fixture-notes", Icon: "fixture-notes",
		Categories: []string{"Utility"}, Keywords: []string{"notes", "a;b"}, StartupWMClass: "fixture-notes",
	}.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	want := "[Desktop Entry]\nType=Application\nVersion=1.0\nName=Fixture Notes\nComment=Takes notes;\\nsecond line\n" +
		"Exec=fixture-notes\nIcon=fixture-notes\nTerminal=false\nCategories=Utility;\nKeywords=notes;a\\;b;\nStartupWMClass=fixture-notes\n"
	if string(data) != want {
		t.Errorf("got\n%s\nwant\n%s", data, want)
	}

	minimal, err := Entry{Name: "N", Exec: "n"}.Bytes()
	if err != nil || string(minimal) != "[Desktop Entry]\nType=Application\nVersion=1.0\nName=N\nExec=n\nTerminal=false\n" {
		t.Errorf("minimal entry: %v\n%s", err, minimal)
	}
}

func TestEntryRejects(t *testing.T) {
	for _, entry := range []Entry{{Exec: "x"}, {Name: "x"}, {Name: "a\x01b", Exec: "x"}, {Name: "x", Exec: "x", Keywords: []string{"\x1b"}}} {
		if _, err := entry.Bytes(); err == nil {
			t.Errorf("%+v was accepted", entry)
		}
	}
}

func TestURLSchemesAddTheLinkFieldAndMimeTypes(t *testing.T) {
	data, err := Entry{Name: "N", Exec: "n", URLSchemes: []string{"ssh", "telnet"}}.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	want := "[Desktop Entry]\nType=Application\nVersion=1.0\nName=N\nExec=n %u\nTerminal=false\nMimeType=x-scheme-handler/ssh;x-scheme-handler/telnet;\n"
	if string(data) != want {
		t.Errorf("got\n%s\nwant\n%s", data, want)
	}
}

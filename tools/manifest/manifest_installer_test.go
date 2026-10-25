package manifest

import (
	"strings"
	"testing"
)

func TestWindowsTableIsRead(t *testing.T) {
	manifest, err := Parse("viva.toml", "/p", "[windows]\npublisher = \"Example Ltd\"\nscope = \"machine\"\nurl = \"https://example.com/notes\"\n")
	if err != nil {
		t.Fatal(err)
	}
	want := Windows{Publisher: "Example Ltd", Scope: "machine", URL: "https://example.com/notes"}
	if manifest.Windows != want {
		t.Fatalf("windows = %+v, want %+v", manifest.Windows, want)
	}
	if manifest.Windows.InstallScope() != ScopeMachine {
		t.Errorf("scope = %q", manifest.Windows.InstallScope())
	}
}

func TestWindowsScopeDefaultsToTheUser(t *testing.T) {
	manifest, err := Parse("viva.toml", "/p", "[app]\nname = \"Notes\"\n")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Windows.Scope != "" || manifest.Windows.InstallScope() != ScopeUser {
		t.Fatalf("scope = %q, resolved %q", manifest.Windows.Scope, manifest.Windows.InstallScope())
	}
	user, err := Parse("viva.toml", "/p", "[windows]\nscope = \"user\"\nurl = \"http://example.com\"\n")
	if err != nil || user.Windows.InstallScope() != ScopeUser {
		t.Fatalf("%v %+v", err, user)
	}
}

func TestWindowsErrors(t *testing.T) {
	for _, test := range []struct{ name, source, want string }{
		{"unknown key", "[windows]\nfoo = \"x\"\n", "viva.toml:2: [windows] foo: is not a known key; [windows] has: publisher, scope, url"},
		{"misspelled key", "[windows]\nscop = \"user\"\n", "viva.toml:2: [windows] scop: is not a known key; [windows] has: publisher, scope, url"},
		{"scope", "[windows]\nscope = \"all\"\n", `viva.toml: [windows] scope: "all" must be "user" (the default) or "machine"`},
		{"scope case", "[windows]\nscope = \"User\"\n", "viva.toml: [windows] scope:"},
		{"url scheme", "[windows]\nurl = \"ftp://example.com\"\n", "viva.toml: [windows] url:"},
		{"url without a scheme", "[windows]\nurl = \"example.com\"\n", "viva.toml: [windows] url:"},
		{"url with a quote", "[windows]\nurl = \"https://example.com/\\\"x\"\n", "viva.toml: [windows] url:"},
		{"url with a space", "[windows]\nurl = \"https://example.com/a b\"\n", "viva.toml: [windows] url:"},
		{"publisher quote", "[windows]\npublisher = \"Ex \\\"Ltd\\\"\"\n", "viva.toml: [windows] publisher:"},
		{"publisher newline", "[windows]\npublisher = \"a\\nb\"\n", "viva.toml: [windows] publisher:"},
		{"wrong type", "[windows]\nscope = 1\n", "viva.toml:2: [windows] scope: must be a string"},
		{"table as value", "windows = \"x\"\n", "viva.toml:1: windows: must be a table ([windows])"},
	} {
		_, err := Parse("viva.toml", "/p", test.source)
		if err == nil {
			t.Errorf("%s: accepted", test.name)
			continue
		}
		if !strings.HasPrefix(err.Error(), test.want) {
			t.Errorf("%s:\n got %v\nwant %s", test.name, err, test.want)
		}
	}
}

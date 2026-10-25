package manifest

import (
	"strings"
	"testing"
)

func TestLinuxTableIsRead(t *testing.T) {
	loaded, err := Parse("viva.toml", "/p", `[app]
name = "N"
id = "dev.example.n"

[linux]
category = "Development"
comment = "Edits things"
maintainer = "Ann Lee <ann@example.com>"
homepage = "https://example.com/n"
depends = ["libfoo1 (>= 1.2)", "bar | baz"]
keywords = ["edit", "notes"]
`)
	if err != nil {
		t.Fatal(err)
	}
	linux := loaded.Linux
	if linux.Category != "Development" || linux.Comment != "Edits things" || len(linux.Depends) != 2 || len(linux.Keywords) != 2 {
		t.Errorf("linux = %+v", linux)
	}
	if got := (Linux{}).LinuxCategory(); got != "Utility" {
		t.Errorf("default category = %q", got)
	}
	if got := linux.LinuxCategory(); got != "Development" {
		t.Errorf("category = %q", got)
	}
}

func TestLinuxTableIsValidated(t *testing.T) {
	for _, test := range []struct{ name, body, want string }{
		{"category", `category = "Games"`, "[linux] category"},
		{"category case", `category = "utility"`, "[linux] category"},
		{"comment newline", `comment = "a\nb"`, "[linux] comment"},
		{"comment control", `comment = "a\u0001b"`, "[linux] comment"},
		{"maintainer bare", `maintainer = "Ann"`, "[linux] maintainer"},
		{"maintainer no address", `maintainer = "Ann <ann>"`, "[linux] maintainer"},
		{"maintainer no name", `maintainer = "<ann@example.com>"`, "[linux] maintainer"},
		{"homepage scheme", `homepage = "ftp://example.com"`, "[linux] homepage"},
		{"homepage bare", `homepage = "example.com"`, "[linux] homepage"},
		{"homepage space", `homepage = "https://exa mple.com"`, "[linux] homepage"},
		{"depends semicolon", `depends = ["a; rm -rf"]`, "[linux] depends"},
		{"depends comma", `depends = ["a, b"]`, "[linux] depends"},
		{"depends empty", `depends = [""]`, "[linux] depends"},
		{"depends padded", `depends = [" a"]`, "[linux] depends"},
		{"keyword semicolon", `keywords = ["a;b"]`, "[linux] keywords"},
		{"keyword empty", `keywords = [""]`, "[linux] keywords"},
	} {
		_, err := Parse("viva.toml", "/p", "[app]\nname = \"N\"\nid = \"dev.example.n\"\n[linux]\n"+test.body+"\n")
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%s: %v, want it to mention %q", test.name, err, test.want)
		}
	}
	for _, category := range LinuxCategories {
		if _, err := Parse("viva.toml", "/p", "[linux]\ncategory = \""+category+"\"\n"); err != nil {
			t.Errorf("category %s: %v", category, err)
		}
	}
	for _, dependency := range []string{"libc6", "libfoo1 (>= 1.2~rc1)", "a | b", "x:amd64 (<< 2)", "libstdc++6"} {
		if _, err := Parse("viva.toml", "/p", "[linux]\ndepends = [\""+dependency+"\"]\n"); err != nil {
			t.Errorf("dependency %q: %v", dependency, err)
		}
	}
}

func TestLinuxUnknownKeyListsTheKeys(t *testing.T) {
	_, err := Parse("viva.toml", "/p", "[linux]\ncategroy = \"Utility\"\n")
	if err == nil {
		t.Fatal("a misspelt key was accepted")
	}
	for _, want := range []string{"viva.toml:2", "[linux] categroy", "category, comment, depends, homepage, keywords, maintainer"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%v lacks %q", err, want)
		}
	}
}

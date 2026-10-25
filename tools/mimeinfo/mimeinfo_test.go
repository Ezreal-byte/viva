package mimeinfo

import (
	"strings"
	"testing"
)

func TestPackage(t *testing.T) {
	got, err := Package([]Type{{MIME: "text/markdown", Name: "Markdown & more", Extensions: []string{"md", "markdown"}}})
	if err != nil {
		t.Fatal(err)
	}
	want := `<?xml version="1.0" encoding="UTF-8"?>
<mime-info xmlns="http://www.freedesktop.org/standards/shared-mime-info">
	<mime-type type="text/markdown">
		<comment>Markdown &amp; more</comment>
		<glob pattern="*.md"/>
		<glob pattern="*.markdown"/>
	</mime-type>
</mime-info>
`
	if string(got) != want {
		t.Fatalf("got:\n%s", got)
	}
}

func TestPackageRejects(t *testing.T) {
	for _, types := range [][]Type{
		nil,
		{{MIME: "text/x", Extensions: nil}},
		{{MIME: "nothing", Extensions: []string{"a"}}},
		{{MIME: "text/x", Name: "a\nb", Extensions: []string{"a"}}},
	} {
		if _, err := Package(types); err == nil || strings.TrimSpace(err.Error()) == "" {
			t.Errorf("%+v: no error", types)
		}
	}
}

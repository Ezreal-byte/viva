package app

import (
	"reflect"
	"testing"
)

func TestPosixLocaleNamesBecomeBCP47Tags(t *testing.T) {
	got := normalizeLanguages([]string{"zh_CN.UTF-8", " en_US@euro ", "C", "POSIX", "", "zh-CN", "pt_BR.ISO-8859-1", "EN-us"})
	want := []string{"zh-CN", "en-US", "pt-BR"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeLanguages = %v, want %v", got, want)
	}
	if got := normalizeLanguages(nil); got != nil {
		t.Fatalf("nothing reported must stay nil, got %v", got)
	}
}

func TestPlatformTagsPassThrough(t *testing.T) {
	got := normalizeLanguages([]string{"zh-Hans-CN", "en-US", "ja-JP"})
	if !reflect.DeepEqual(got, []string{"zh-Hans-CN", "en-US", "ja-JP"}) {
		t.Fatalf("got %v", got)
	}
}

func TestEnvironmentLocalePrefersLanguageThenLibcOrder(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(name string) string { return values[name] }
	}
	for _, test := range []struct {
		name   string
		values map[string]string
		want   []string
	}{
		{"LANGUAGE list first", map[string]string{"LANGUAGE": "zh_CN:en", "LANG": "en_US.UTF-8"}, []string{"zh-CN", "en", "en-US"}},
		{"LC_ALL beats LANG", map[string]string{"LC_ALL": "de_DE.UTF-8", "LANG": "en_US.UTF-8"}, []string{"de-DE"}},
		{"LC_MESSAGES beats LANG", map[string]string{"LC_MESSAGES": "fr_FR", "LANG": "en_US.UTF-8"}, []string{"fr-FR"}},
		{"only LANG", map[string]string{"LANG": "ja_JP.UTF-8"}, []string{"ja-JP"}},
		{"the C locale says nothing", map[string]string{"LANG": "C"}, nil},
		{"nothing set", map[string]string{}, nil},
	} {
		if got := normalizeLanguages(languagesFromEnvironment(env(test.values))); !reflect.DeepEqual(got, test.want) {
			t.Errorf("%s: %v, want %v", test.name, got, test.want)
		}
	}
}

package linux

import "testing"

func TestLatin1KeepsWhatFitsAndMarksTheRest(t *testing.T) {
	for _, test := range []struct{ in, want string }{
		{"Viva", "Viva"},
		{"café", "caf\xe9"},
		{"终端", "??"},
		{"a\U0001F600b", "a?b"},
		{"", ""},
	} {
		if got := string(latin1(test.in)); got != test.want {
			t.Fatalf("latin1(%q) = %q, want %q", test.in, got, test.want)
		}
	}
}

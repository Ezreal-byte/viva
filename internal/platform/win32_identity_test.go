package platform

import "testing"

func TestAppUserModelIDIsTheClassTheApplicationNamed(t *testing.T) {
	for _, test := range []struct{ class, want string }{
		{"Viva.Showcase", "Viva.Showcase"},
		{"Company Product", "Company.Product"},
		{"  ", ""},
		{"", ""},
	} {
		if got := AppUserModelID(test.class); got != test.want {
			t.Errorf("AppUserModelID(%q) = %q, want %q", test.class, got, test.want)
		}
	}
	long := make([]rune, 300)
	for index := range long {
		long[index] = 'a'
	}
	if got := AppUserModelID(string(long)); len([]rune(got)) != 128 {
		t.Errorf("a long name is %d characters", len([]rune(got)))
	}
}

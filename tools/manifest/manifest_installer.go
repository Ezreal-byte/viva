package manifest

import (
	"fmt"
	"net/url"
	"strings"
)

// Windows installation scopes.
const (
	// ScopeUser installs under the user's profile and needs no administrator
	// rights. It is what an empty [windows] scope means.
	ScopeUser = "user"
	// ScopeMachine installs under Program Files for every user.
	ScopeMachine = "machine"
)

// InstallScope is the scope the installer uses: [windows] scope, or the user's.
func (windows Windows) InstallScope() string {
	if windows.Scope == "" {
		return ScopeUser
	}
	return windows.Scope
}

// validateWindows checks the [windows] table. Empty values are left as written;
// InstallScope and the packager resolve the defaults where they are used.
func (manifest *Manifest) validateWindows() error {
	fail := func(key, format string, arguments ...any) error {
		return fmt.Errorf("[windows] %s: %s", key, fmt.Sprintf(format, arguments...))
	}
	windows := manifest.Windows
	switch windows.Scope {
	case "", ScopeUser, ScopeMachine:
	default:
		return fail("scope", "%q must be %q (the default) or %q", windows.Scope, ScopeUser, ScopeMachine)
	}
	if windows.URL != "" {
		parsed, err := url.Parse(windows.URL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return fail("url", "%q must be an http:// or https:// address", windows.URL)
		}
		if hasControlOrQuote(windows.URL) || strings.ContainsAny(windows.URL, " \\") {
			return fail("url", "%q cannot contain spaces, quotes, backslashes or control characters", windows.URL)
		}
	}
	if hasControlOrQuote(windows.Publisher) {
		return fail("publisher", "%q cannot contain quotes or control characters", windows.Publisher)
	}
	return nil
}

// hasControlOrQuote reports a control character or a quotation mark.
func hasControlOrQuote(text string) bool {
	return strings.IndexFunc(text, func(r rune) bool { return r < ' ' || r == 0x7f || r == '"' || r == '`' }) >= 0
}

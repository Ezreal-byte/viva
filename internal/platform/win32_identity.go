package platform

import "strings"

// AppUserModelID is the name Windows knows the application by, to group its
// windows under one taskbar button and to pin it: the window class the
// application named (Config.Class), with no spaces and at most 128 characters.
// Without a class it is empty, and Windows groups by the executable's path as it
// does for any program, which keeps a shortcut made by an installer that knows
// nothing of this name grouped with the windows.
func AppUserModelID(class string) string {
	name := strings.Join(strings.Fields(class), ".")
	if runes := []rune(name); len(runes) > 128 {
		name = string(runes[:128])
	}
	return name
}

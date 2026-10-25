package ui

import (
	"strings"
	"sync/atomic"
)

// MaxMonoFamilies is how many named fixed-pitch families one process can use,
// besides the two built-in faces.
const MaxMonoFamilies = 250

// maxMonoFamilyName bounds a family name; real ones are a few words long.
const maxMonoFamilyName = 96

// monoFamilies is the registry: entry i is the name of MonoFace(2+i). It is
// replaced as a whole, never edited, so a reader on the render thread sees
// either the old table or the new one.
var monoFamilies atomic.Pointer[[]string]

// MonoFamily returns the face that draws fixed-pitch text in the installed font
// family called name ("JetBrains Mono", "Menlo"). The same name always gives the
// same face. A name that is empty, too long, holds a control character or
// would be one past the limit gives MonoSystem. Whether the family is installed is
// the platform's to answer: a name it does not know draws in its fallback
// fixed-pitch font, so an application should offer only names it has listed.
func MonoFamily(name string) MonoFace {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxMonoFamilyName {
		return MonoSystem
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return MonoSystem
		}
	}
	for {
		current := monoFamilies.Load()
		var families []string
		if current != nil {
			families = *current
		}
		for index, existing := range families {
			if existing == name {
				return MonoFace(2 + index)
			}
		}
		if len(families) >= MaxMonoFamilies {
			return MonoSystem
		}
		next := append(append(make([]string, 0, len(families)+1), families...), name)
		if monoFamilies.CompareAndSwap(current, &next) {
			return MonoFace(2 + len(families))
		}
	}
}

// MonoFamilyName is the family a face from MonoFamily stands for, or "" for the
// built-in faces and for a number no MonoFamily call returned.
func MonoFamilyName(face MonoFace) string {
	if face < 2 {
		return ""
	}
	families := monoFamilies.Load()
	if families == nil || int(face-2) >= len(*families) {
		return ""
	}
	return (*families)[face-2]
}

// MonoFamilyCount is how many named families have been registered, so a platform
// can tell cheaply that it has new ones to learn.
func MonoFamilyCount() int {
	if families := monoFamilies.Load(); families != nil {
		return len(*families)
	}
	return 0
}

// validMonoFace reports whether face is one the canvas can draw: a built-in
// one or a registered family.
func validMonoFace(face MonoFace) bool {
	return face <= MonoClassic || int(face-2) < MonoFamilyCount()
}

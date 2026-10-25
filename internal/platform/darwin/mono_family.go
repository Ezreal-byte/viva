//go:build darwin

package darwin

/*
#include <stdlib.h>
#include "native.h"
*/
import "C"

import (
	"strings"
	"sync"
	"unsafe"

	"github.com/vivaui/viva/ui"
)

var (
	monoFamilyLock sync.Mutex
	monoFamilySent int
)

// syncMonoFamilies tells the native side the names of the font families the
// application registered with ui.MonoFamily since the last call. Faces are
// numbered in registration order, so only the new ones are sent.
func syncMonoFamilies() {
	count := ui.MonoFamilyCount()
	monoFamilyLock.Lock()
	defer monoFamilyLock.Unlock()
	for monoFamilySent < count {
		name := C.CString(ui.MonoFamilyName(ui.MonoFace(2 + monoFamilySent)))
		C.viva_register_mono_family(C.int(2+monoFamilySent), name)
		C.free(unsafe.Pointer(name))
		monoFamilySent++
	}
}

// MonospaceFamilies lists the installed fixed-pitch font families, sorted.
func MonospaceFamilies() []string {
	list := C.viva_list_mono_families()
	if list == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(list))
	value := C.GoString(list)
	if value == "" {
		return nil
	}
	return strings.Split(value, "\n")
}

//go:build !baseline && !integer && !windows

package main

import (
	"github.com/ebitengine/purego"
	"runtime"
)

func load() (func(), func(), error) {
	name := "libc.so.6"
	if runtime.GOOS == "darwin" {
		name = "/usr/lib/libSystem.B.dylib"
	}
	library, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return nil, nil, err
	}
	var getpid func() int32
	purego.RegisterLibFunc(&getpid, library, "getpid")
	call := func() { getpid() }
	return call, func() { purego.Dlclose(library) }, nil
}

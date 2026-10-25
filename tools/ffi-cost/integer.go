//go:build integer && !windows

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
	address, err := purego.Dlsym(library, "getpid")
	if err != nil {
		purego.Dlclose(library)
		return nil, nil, err
	}
	return func() { purego.SyscallN(address) }, func() { purego.Dlclose(library) }, nil
}

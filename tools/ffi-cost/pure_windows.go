//go:build !baseline && !integer

package main

import (
	"github.com/ebitengine/purego"
	"syscall"
)

func load() (func(), func(), error) {
	procedure := syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentProcessId")
	if err := procedure.Find(); err != nil {
		return nil, nil, err
	}
	var getpid func() uint32
	purego.RegisterFunc(&getpid, procedure.Addr())
	call := func() { getpid() }
	return call, func() {}, nil
}

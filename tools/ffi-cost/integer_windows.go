//go:build integer

package main

import "syscall"

func load() (func(), func(), error) {
	procedure := syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentProcessId")
	if err := procedure.Find(); err != nil {
		return nil, nil, err
	}
	return func() { procedure.Call() }, func() {}, nil
}

//go:build baseline

package main

import "syscall"

func load() (func(), func(), error) { return func() { syscall.Getpid() }, func() {}, nil }

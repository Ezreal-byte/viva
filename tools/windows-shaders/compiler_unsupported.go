//go:build !windows

package main

import "fmt"

func compileShader([]byte, string, string) ([]byte, error) {
	return nil, fmt.Errorf("run this development generator on Windows with d3dcompiler_47.dll")
}

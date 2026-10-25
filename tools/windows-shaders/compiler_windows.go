package main

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

func blobCall(blob uintptr, slot int) uintptr {
	object := *(*unsafe.Pointer)(unsafe.Pointer(&blob))
	table := *(*unsafe.Pointer)(object)
	method := *(*uintptr)(unsafe.Add(table, slot*int(unsafe.Sizeof(uintptr(0)))))
	result, _, _ := syscall.SyscallN(method, blob)
	return result
}

func compileShader(source []byte, entry, profile string) ([]byte, error) {
	var directory [32768]uint16
	length, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemDirectoryW").Call(uintptr(unsafe.Pointer(&directory[0])), uintptr(len(directory)))
	if length == 0 || length >= uintptr(len(directory)) {
		return nil, fmt.Errorf("cannot locate system directory")
	}
	compiler := syscall.NewLazyDLL(syscall.UTF16ToString(directory[:length]) + `\d3dcompiler_47.dll`)
	compile := compiler.NewProc("D3DCompile")
	if err := compile.Find(); err != nil {
		return nil, err
	}
	entryName, err := syscall.BytePtrFromString(entry)
	if err != nil {
		return nil, err
	}
	profileName, err := syscall.BytePtrFromString(profile)
	if err != nil {
		return nil, err
	}
	sourceName, _ := syscall.BytePtrFromString("scene.hlsl")
	var code, diagnostics uintptr
	status, _, _ := compile.Call(uintptr(unsafe.Pointer(&source[0])), uintptr(len(source)), uintptr(unsafe.Pointer(sourceName)), 0, 0, uintptr(unsafe.Pointer(entryName)), uintptr(unsafe.Pointer(profileName)), uintptr(compilerFlags), 0, uintptr(unsafe.Pointer(&code)), uintptr(unsafe.Pointer(&diagnostics)))
	if code != 0 {
		defer blobCall(code, 2)
	}
	if diagnostics != 0 {
		defer blobCall(diagnostics, 2)
	}
	if int32(status) < 0 {
		if diagnostics == 0 {
			return nil, fmt.Errorf("D3DCompile HRESULT %#x", uint32(status))
		}
		data, length := blobCall(diagnostics, 3), blobCall(diagnostics, 4)
		return nil, fmt.Errorf("%s", unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&data))), int(length)))
	}
	if code == 0 {
		return nil, fmt.Errorf("D3DCompile returned no bytecode")
	}
	data, length := blobCall(code, 3), blobCall(code, 4)
	result := append([]byte(nil), unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&data))), int(length))...)
	runtime.KeepAlive(source)
	return result, nil
}

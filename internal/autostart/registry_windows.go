package autostart

import (
	"errors"
	"strings"
	"syscall"
	"unsafe"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func openRun(access uint32) (syscall.Handle, error) {
	name, _ := syscall.UTF16PtrFromString(runKey)
	var key syscall.Handle
	var disposition uint32
	status, _, _ := createKey.Call(uintptr(syscall.HKEY_CURRENT_USER), uintptr(unsafe.Pointer(name)), 0, 0, 0,
		uintptr(access), 0, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&disposition)))
	if status != 0 {
		return 0, syscall.Errno(status)
	}
	return key, nil
}

// commandLine quotes the program and arguments the way CommandLineToArgvW reads
// them back.
func commandLine(entry Entry) string {
	words := make([]string, 0, 1+len(entry.Args))
	for _, word := range append([]string{entry.Executable}, entry.Args...) {
		words = append(words, syscall.EscapeArg(word))
	}
	return strings.Join(words, " ")
}

func enable(entry Entry) error {
	key, err := openRun(syscall.KEY_SET_VALUE)
	if err != nil {
		return err
	}
	defer syscall.RegCloseKey(key)
	name, err := syscall.UTF16PtrFromString(entry.ID)
	if err != nil {
		return err
	}
	value, err := syscall.UTF16FromString(commandLine(entry))
	if err != nil {
		return err
	}
	status, _, _ := setValue.Call(uintptr(key), uintptr(unsafe.Pointer(name)), 0, syscall.REG_SZ,
		uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)*2))
	if status != 0 {
		return syscall.Errno(status)
	}
	return nil
}

func disable(id string) error {
	key, err := openRun(syscall.KEY_SET_VALUE)
	if err != nil {
		return err
	}
	defer syscall.RegCloseKey(key)
	name, _ := syscall.UTF16PtrFromString(id)
	err = regDeleteValue(key, name)
	if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return nil
	}
	return err
}

func enabled(id string) (bool, error) {
	key, err := openRun(syscall.KEY_QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer syscall.RegCloseKey(key)
	name, _ := syscall.UTF16PtrFromString(id)
	var kind, size uint32
	err = syscall.RegQueryValueEx(key, name, nil, &kind, nil, &size)
	if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return false, nil
	}
	return err == nil, err
}

var advapi = syscall.NewLazyDLL("advapi32.dll")
var deleteValue = advapi.NewProc("RegDeleteValueW")
var createKey = advapi.NewProc("RegCreateKeyExW")
var setValue = advapi.NewProc("RegSetValueExW")

func regDeleteValue(key syscall.Handle, name *uint16) error {
	status, _, _ := deleteValue.Call(uintptr(key), uintptr(unsafe.Pointer(name)))
	if status != 0 {
		return syscall.Errno(status)
	}
	return nil
}

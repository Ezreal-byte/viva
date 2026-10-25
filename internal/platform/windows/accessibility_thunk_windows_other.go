//go:build windows && !amd64

package windows

func uiaFromPointThunkAddress() uintptr { return 0 }

func uiaRangeValueThunkAddress() uintptr { return 0 }

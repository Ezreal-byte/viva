//go:build !darwin && !linux && !windows

package autostart

func enable(Entry) error           { return ErrUnsupported }
func disable(string) error         { return ErrUnsupported }
func enabled(string) (bool, error) { return false, ErrUnsupported }

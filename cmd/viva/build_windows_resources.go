package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vivaui/viva/tools/winpkg"
	"github.com/vivaui/viva/tools/winres"
)

// addWindowsResources gives a program built for Windows its own icon and version
// information, which Explorer, the taskbar, Alt+Tab and Task Manager read from
// the .exe: a Go program carries none. They are added to the linked file, so the
// source directory is left alone. A program that already has resources, for
// example from a .syso of its own, keeps them.
func (plan buildPlan) addWindowsResources(output string, warnings io.Writer) error {
	if plan.target != "windows" {
		return nil
	}
	var ico []byte
	if icon := plan.manifest.App.Icon; icon != "" {
		switch strings.ToLower(filepath.Ext(icon)) {
		case ".png", ".ico":
			var err error
			if ico, _, err = winpkg.LoadIcon(plan.manifest.Path(icon)); err != nil {
				// A bad icon costs the program its icon, not the build.
				fmt.Fprintf(warnings, "viva: the program gets no icon: %v\n", err)
				ico = nil
			}
		}
	}
	program, err := os.ReadFile(output)
	if err != nil {
		return err
	}
	patched, err := winres.Patch(program, ico, winres.Info{
		Name: plan.manifest.App.Name, Description: plan.manifest.App.Name, Version: plan.manifest.App.Version,
		Publisher: plan.manifest.Windows.Publisher, File: plan.executable + ".exe",
	})
	if errors.Is(err, winres.ErrHasResources) {
		return nil
	}
	if err != nil {
		return err
	}
	info, err := os.Stat(output)
	if err != nil {
		return err
	}
	return os.WriteFile(output, patched, info.Mode().Perm())
}

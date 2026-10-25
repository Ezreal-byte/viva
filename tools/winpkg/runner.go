package winpkg

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// CommonLocations are the directories searched for makensis after PATH and
// NSIS_HOME. It is a variable so a test can keep a machine's own NSIS out of
// the way.
var CommonLocations = []string{
	"/opt/homebrew/bin", "/usr/local/bin", "/usr/bin",
	`C:\Program Files (x86)\NSIS`, `C:\Program Files\NSIS`,
}

// InstallHint says how to get NSIS.
const InstallHint = "install NSIS: on macOS 'brew install nsis', on Debian or Ubuntu 'apt install nsis', " +
	"on Windows from https://nsis.sourceforge.io (or set NSIS_HOME to its directory)"

// ErrNoNSIS is wrapped by the error Find returns when makensis is missing.
var ErrNoNSIS = errors.New("makensis was not found")

// sourceDateEpoch is 1980-01-01, the time zip files use for "no time".
const sourceDateEpoch = "315532800"

// ErrMakensisCrash is wrapped by the error Build returns when makensis dies of
// the macOS Unicode bug rather than of anything in the script.
var ErrMakensisCrash = errors.New("winpkg: makensis crashed")

// Find locates makensis: PATH first, then NSIS_HOME (itself or its Bin
// directory), then the usual install locations.
func Find() (string, error) { return find(os.Getenv("NSIS_HOME"), CommonLocations) }

func find(home string, locations []string) (string, error) {
	if path, err := exec.LookPath("makensis"); err == nil {
		return path, nil
	}
	names := []string{"makensis"}
	if runtime.GOOS == "windows" {
		names = []string{"makensis.exe"}
	}
	var directories []string
	if home != "" {
		directories = append(directories, home, filepath.Join(home, "Bin"))
	}
	directories = append(directories, locations...)
	for _, directory := range directories {
		for _, name := range names {
			candidate := filepath.Join(directory, name)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("%w: a Windows installer is built with NSIS; %s", ErrNoNSIS, InstallHint)
}

// Result is what a successful run reports.
type Result struct {
	// Warnings are makensis's warning lines, without the prefix.
	Warnings []string
}

// Build writes the script into a scratch directory and compiles it with
// makensis, producing script.Output. makensis runs with UTF-8 input, without
// changing directory (every path in the script is absolute) and at a verbosity
// that shows warnings.
func Build(script Script) (Result, error) {
	makensis, err := Find()
	if err != nil {
		return Result{}, err
	}
	return build(makensis, script)
}

func build(makensis string, script Script) (Result, error) {
	source, err := script.Generate()
	if err != nil {
		return Result{}, err
	}
	scratch, err := os.MkdirTemp("", "viva-nsis-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(scratch)
	file := filepath.Join(scratch, "installer.nsi")
	if err := os.WriteFile(file, source, 0o600); err != nil {
		return Result{}, err
	}
	command := exec.Command(makensis, "-INPUTCHARSET", "UTF8", "-NOCD", "-V2", file)
	command.Dir = scratch
	// NSIS stamps the installer with the build time unless it is told one; a
	// fixed time makes the same inputs give the same installer.
	command.Env = append(os.Environ(), "SOURCE_DATE_EPOCH="+sourceDateEpoch)
	var captured bytes.Buffer
	command.Stdout, command.Stderr = &captured, &captured
	if err := command.Run(); err != nil {
		if strings.Contains(captured.String(), "bad_alloc") {
			return Result{}, fmt.Errorf("%w: %s crashed with std::bad_alloc while generating the language tables; "+
				"this is NSIS bug #1333, which stops makensis builds on macOS from compiling any Unicode installer; "+
				"build the installer on Linux or Windows (viva package --target windows works there) or with an NSIS built from a fixed source", ErrMakensisCrash, makensis)
		}
		return Result{}, fmt.Errorf("winpkg: makensis failed: %w\n%s", err, strings.TrimSpace(captured.String()))
	}
	return Result{Warnings: warnings(captured.String())}, nil
}

// warnings picks the lines makensis labels as warnings. Its own output has the
// form "warning: message (file:line)"; a continuation line is part of the
// previous one and is skipped.
func warnings(output string) []string {
	var found []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "warning: "); ok {
			found = append(found, rest)
		}
	}
	return found
}

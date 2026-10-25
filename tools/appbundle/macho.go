package appbundle

import (
	"debug/macho"
	"fmt"
)

const (
	lcVersionMinMacOSX = 0x24
	lcBuildVersion     = 0x32
	platformMacOS      = 1
)

// RequiredMacOS reports the oldest macOS a Mach-O program says it runs on, as
// "major.minor" (or "major.minor.patch" when the patch is not zero). It reads
// the deployment target the linker recorded, which is what the system checks
// before it starts the program; Info.plist's LSMinimumSystemVersion is only a
// statement, and a bundle that claims an older system than its program needs
// fails to launch there.
func RequiredMacOS(path string) (string, error) {
	file, err := macho.Open(path)
	if err != nil {
		return "", fmt.Errorf("appbundle: reading %s: %w", path, err)
	}
	defer file.Close()
	order := file.ByteOrder
	for _, load := range file.Loads {
		raw, ok := load.(macho.LoadBytes)
		if !ok || len(raw) < 16 {
			continue
		}
		switch order.Uint32(raw[0:4]) {
		case lcBuildVersion:
			if order.Uint32(raw[8:12]) == platformMacOS {
				return formatPackedVersion(order.Uint32(raw[12:16])), nil
			}
		case lcVersionMinMacOSX:
			return formatPackedVersion(order.Uint32(raw[8:12])), nil
		}
	}
	return "", fmt.Errorf("appbundle: %s records no macOS deployment target", path)
}

// formatPackedVersion decodes the linker's xxxx.yy.zz layout.
func formatPackedVersion(packed uint32) string {
	major, minor, patch := packed>>16, packed>>8&0xff, packed&0xff
	if patch != 0 {
		return fmt.Sprintf("%d.%d.%d", major, minor, patch)
	}
	return fmt.Sprintf("%d.%d", major, minor)
}

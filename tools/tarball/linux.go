package tarball

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/vivaui/viva/tools/desktopentry"
	"github.com/vivaui/viva/tools/mimeinfo"
)

// Linux describes a release archive for Linux: a directory holding the
// program, its menu entry, its icon and a script that installs them.
type Linux struct {
	// Executable is the program's file name; it also names the menu entry and
	// the icon.
	Executable string
	// Name is the display name in the menu.
	Name    string
	Version string
	// Arch is amd64 or arm64.
	Arch     string
	Comment  string
	Category string
	Keywords []string
	// URLSchemes are the link schemes the application opens.
	URLSchemes []string
	// FileTypes are the document types the application opens.
	FileTypes []mimeinfo.Type
	// Program is the executable's bytes.
	Program []byte
	// Icon is a square PNG, the largest size available (256 px when the source
	// allows); empty for an application without an icon.
	Icon       []byte
	IconPixels int
}

var executablePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)

// Directory is the name of the archive's single top-level directory, which is
// also the archive's name without .tar.gz.
func (spec Linux) Directory() string {
	return fmt.Sprintf("%s-%s-linux-%s", spec.Executable, spec.Version, spec.Arch)
}

func (spec Linux) check() error {
	switch {
	case !executablePattern.MatchString(spec.Executable):
		return fmt.Errorf("tarball: %q is not a plain executable name", spec.Executable)
	case spec.Name == "":
		return errors.New("tarball: the application needs a display name")
	case spec.Version == "" || strings.ContainsAny(spec.Version, "/ \t\n"):
		return fmt.Errorf("tarball: %q is not a usable version", spec.Version)
	case spec.Arch != "amd64" && spec.Arch != "arm64":
		return fmt.Errorf("tarball: architecture %q must be amd64 or arm64", spec.Arch)
	case len(spec.Program) == 0:
		return errors.New("tarball: the program is empty")
	case (len(spec.Icon) == 0) != (spec.IconPixels == 0):
		return errors.New("tarball: an icon needs its size and the other way round")
	}
	return nil
}

// DesktopEntry is the menu entry inside the archive: it starts the program by
// name, which install.sh rewrites to the installed path.
func (spec Linux) DesktopEntry() ([]byte, error) {
	entry := desktopentry.Entry{
		Name: spec.Name, Comment: spec.Comment, Exec: spec.Executable,
		Categories: []string{spec.Category}, Keywords: spec.Keywords, StartupWMClass: spec.Executable,
		URLSchemes: spec.URLSchemes, MimeTypes: mimeinfo.Types(spec.FileTypes),
	}
	if len(spec.Icon) > 0 {
		entry.Icon = spec.Executable
	}
	return entry.Bytes()
}

// Write writes the .tar.gz.
func (spec Linux) Write(output io.Writer) error {
	if err := spec.check(); err != nil {
		return err
	}
	desktop, err := spec.DesktopEntry()
	if err != nil {
		return err
	}
	files := []File{
		{Path: spec.Executable, Data: spec.Program, Mode: 0o755},
		{Path: spec.Executable + ".desktop", Data: desktop, Mode: 0o644},
		{Path: "install.sh", Data: []byte(spec.script()), Mode: 0o755},
		{Path: "README.txt", Data: []byte(spec.readme()), Mode: 0o644},
	}
	if len(spec.Icon) > 0 {
		files = append(files, File{Path: spec.Executable + ".png", Data: spec.Icon, Mode: 0o644})
	}
	if len(spec.FileTypes) > 0 {
		types, err := mimeinfo.Package(spec.FileTypes)
		if err != nil {
			return err
		}
		files = append(files, File{Path: spec.Executable + ".mime.xml", Data: types, Mode: 0o644})
	}
	var archive bytes.Buffer
	if err := WriteTar(&archive, files, spec.Directory(), SourceDate()); err != nil {
		return err
	}
	compressed, err := Gzip(archive.Bytes())
	if err != nil {
		return err
	}
	_, err = output.Write(compressed)
	return err
}

func (spec Linux) readme() string {
	return strings.NewReplacer("@NAME@", spec.Name, "@VERSION@", spec.Version, "@ARCH@", spec.Arch, "@APP@", spec.Executable).Replace(readmeText)
}

const readmeText = `@NAME@ @VERSION@ for Linux (@ARCH@)

Install for the current user (no administrator rights):

    ./install.sh

This puts the program in ~/.local/bin, the menu entry in
~/.local/share/applications and the icon in ~/.local/share/icons. Run it as
root, or give --prefix, to install elsewhere (root defaults to /usr/local):

    sudo ./install.sh
    ./install.sh --prefix /opt/@APP@

Remove it again with the same prefix:

    ./install.sh uninstall

Or skip installing and run ./@APP@ from this directory.

The program needs a Vulkan driver (libvulkan.so.1), libxcb and libcairo on the
system, as every Viva application does. This archive is not signed.
`

// script is install.sh: POSIX sh, so it runs under dash as well as bash. Only
// the validated executable name and the icon size are put into it.
func (spec Linux) script() string {
	size := ""
	if spec.IconPixels > 0 {
		size = fmt.Sprint(spec.IconPixels)
	}
	link := ""
	if field := desktopentry.ExecField(len(spec.FileTypes) > 0, len(spec.URLSchemes) > 0); field != "" {
		link = " " + field
	}
	types := ""
	if len(spec.FileTypes) > 0 {
		types = "yes"
	}
	return strings.NewReplacer("@APP@", spec.Executable, "@SIZE@", size, "@LINK@", link, "@TYPES@", types).Replace(installScript)
}

// installScript is the template of install.sh; @APP@ is the executable,
// @SIZE@ the icon's pixel size, or empty without an icon, and @TYPES@ is "yes"
// when the archive carries document types.
const installScript = `#!/bin/sh
# Installs or removes @APP@. Run ./install.sh --help for the options.
set -eu

app=@APP@
icon_size=@SIZE@
file_types=@TYPES@
here=$(cd "$(dirname "$0")" && pwd)

usage() {
	cat <<'EOF'
usage: ./install.sh [install|uninstall] [--prefix DIR]

  install      copy the program, menu entry and icon under the prefix (default)
  uninstall    remove them again
  --prefix DIR where to install: DIR/bin, DIR/share/applications, DIR/share/icons.
               Default is ~/.local, or /usr/local when run as root.
EOF
}

fail() {
	echo "install.sh: $*" >&2
	exit 1
}

action=install
prefix=
while [ $# -gt 0 ]; do
	case $1 in
	install | uninstall) action=$1 ;;
	--prefix)
		[ $# -ge 2 ] || fail "--prefix needs a directory"
		prefix=$2
		shift
		;;
	--prefix=*) prefix=${1#--prefix=} ;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		usage >&2
		exit 2
		;;
	esac
	shift
done

if [ -z "$prefix" ]; then
	if [ "$(id -u)" = 0 ]; then
		prefix=/usr/local
	else
		[ -n "${HOME:-}" ] || fail "HOME is not set; give --prefix"
		prefix=$HOME/.local
	fi
fi
case $prefix in
/*) ;;
*) fail "the prefix must be an absolute path" ;;
esac
# The path goes into the menu entry's Exec line, where spaces and shell
# characters would need quoting; refuse them instead of getting that wrong.
case $prefix in
*[!A-Za-z0-9_./+-]*) fail "the prefix may only contain letters, digits and _ . / + -" ;;
esac
prefix=${prefix%/}

bin=$prefix/bin
applications=$prefix/share/applications
icon_directory=$prefix/share/icons/hicolor/${icon_size}x${icon_size}/apps
mime_root=$prefix/share/mime

refresh() {
	if command -v update-desktop-database >/dev/null 2>&1; then
		update-desktop-database -q "$applications" 2>/dev/null || true
	fi
	if [ -n "$file_types" ] && command -v update-mime-database >/dev/null 2>&1; then
		update-mime-database "$mime_root" 2>/dev/null || true
	fi
	if [ -n "$icon_size" ] && command -v gtk-update-icon-cache >/dev/null 2>&1 && [ -d "$prefix/share/icons/hicolor" ]; then
		gtk-update-icon-cache -q -t -f "$prefix/share/icons/hicolor" 2>/dev/null || true
	fi
}

if [ "$action" = uninstall ]; then
	rm -f "$bin/$app" "$applications/$app.desktop"
	if [ -n "$file_types" ]; then
		rm -f "$mime_root/packages/$app.xml"
		rmdir "$mime_root/packages" 2>/dev/null || true
	fi
	if [ -n "$icon_size" ]; then
		rm -f "$icon_directory/$app.png"
		# Take away the size directories the install made, if nothing else is in them.
		rmdir "$icon_directory" "${icon_directory%/apps}" 2>/dev/null || true
	fi
	refresh
	echo "removed $app from $prefix"
	exit 0
fi

[ -f "$here/$app" ] || fail "$here/$app is missing; run this script from the unpacked archive"

mkdir -p "$bin" "$applications"
# Copy beside the target and rename, so replacing a running program works.
cp "$here/$app" "$bin/.$app.new"
chmod 755 "$bin/.$app.new"
mv -f "$bin/.$app.new" "$bin/$app"

if [ -n "$icon_size" ]; then
	mkdir -p "$icon_directory"
	cp "$here/$app.png" "$icon_directory/$app.png"
	chmod 644 "$icon_directory/$app.png"
	sed -e "s|^Exec=.*|Exec=$bin/$app@LINK@|" -e "s|^Icon=.*|Icon=$icon_directory/$app.png|" "$here/$app.desktop" >"$applications/.$app.desktop.new"
else
	sed -e "s|^Exec=.*|Exec=$bin/$app@LINK@|" "$here/$app.desktop" >"$applications/.$app.desktop.new"
fi
chmod 644 "$applications/.$app.desktop.new"
mv -f "$applications/.$app.desktop.new" "$applications/$app.desktop"
if [ -n "$file_types" ]; then
	mkdir -p "$mime_root/packages"
	cp "$here/$app.mime.xml" "$mime_root/packages/$app.xml"
	chmod 644 "$mime_root/packages/$app.xml"
fi
refresh

echo "installed $app in $prefix"
case ":${PATH:-}:" in
*":$bin:"*) ;;
*) echo "note: $bin is not in PATH; start it from the menu or run $bin/$app" ;;
esac
`

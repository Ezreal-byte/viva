#!/bin/bash
# Runs the compiled fidelity test of internal/platform/linux (go test -c, then this script on
# the Linux test machine) under several Xft resource sets on a private X display, to see
# that Viva's text follows the font settings of the desktop as a real GtkLabel does.
#
#   Xvnc :17 -SecurityTypes None -geometry 1024x768 -rfbport 5917 -localhost &
#   linux_variants.sh /path/to/linux.test
#
# The test compares labels with GtkLabels drawn on the same display, so the settings are
# applied to both.
test=$1
export HOME=${TMPDIR:-/tmp}/fidelity-home SHELL=/bin/bash DISPLAY=${DISPLAY:-:17}
mkdir -p $HOME
if [ -f /tmp/viva-vulkan/lvp-local.json ]; then export VK_ICD_FILENAMES=/tmp/viva-vulkan/lvp-local.json LD_LIBRARY_PATH=/tmp/viva-vulkan/root/usr/lib/x86_64-linux-gnu; fi
run() {
  printf 'Xft.antialias: %s\nXft.hinting: %s\nXft.hintstyle: %s\nXft.rgba: %s\n' "$1" "$2" "$3" "$4" | xrdb -load
  echo "== antialias=$1 hinting=$2 hintstyle=$3 rgba=$4"
  timeout 100 "$test" -test.count=1 -test.run TestLabelsAgainstAGtkLabel -test.v 2>&1 | grep -E "lines:|words:|FAIL" | cut -c1-300
}
run 1 1 hintnone none
run 1 1 hintmedium none
run 1 0 hintfull none
run 0 1 hintfull none
run 1 1 hintslight bgr
run 1 1 hintfull vrgb
run 1 1 hintslight rgb

# Addresses of your own Linux and Windows test environments. Source this file.
# Export VIVA_LINUX_ENV and VIVA_WINDOWS_ENV, or put those exports in
# tools/testenv/local.sh, which is ignored by git.
here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
if [ -f "$here/local.sh" ]; then . "$here/local.sh"; fi
VIVA_LINUX_ENV=${VIVA_LINUX_ENV:-}
VIVA_WINDOWS_ENV=${VIVA_WINDOWS_ENV:-}
export VIVA_LINUX_ENV VIVA_WINDOWS_ENV
# Keep a developer proxy out of the way when the environments are on the LAN.
export no_proxy="${no_proxy:-},localhost,127.0.0.1"

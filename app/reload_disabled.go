//go:build !vivadev && (darwin || windows || linux)

package app

import "github.com/vivaui/viva/ui"

// runReloadWorkerIfEnabled is the release implementation: no worker protocol
// package is imported, so a normal build neither starts a worker nor carries
// its server and codec.
func runReloadWorkerIfEnabled(Config, *ui.Canvas) (bool, error) { return false, nil }

//go:build darwin || windows || linux

package app

import (
	"errors"

	"github.com/vivaui/viva/internal/platform"
)

// secretError turns what a platform's secret store reports into the sentinels
// callers test for.
func secretError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, platform.ErrSecretNotFound):
		return ErrSecretNotFound
	case errors.Is(err, platform.ErrSecretStoreUnavailable):
		return ErrSecretStoreUnavailable
	case errors.Is(err, platform.ErrSecretDenied):
		return ErrPermissionDenied
	}
	return err
}

//go:build darwin

package darwin

import (
	"testing"

	"github.com/vivaui/viva/internal/textenginetest"
)

func TestTextEnginePassesTheContract(t *testing.T) {
	textenginetest.Run(t, SharedTextEngine())
}

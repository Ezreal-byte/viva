//go:build darwin

package darwin

import (
	"os"
	"testing"

	"github.com/vivaui/viva/internal/platform/darwin/mainpump"
)

// The tests reach functions that hop to the main thread; serve that queue.
func TestMain(m *testing.M) { os.Exit(mainpump.Run(m)) }

package gallery

import (
	"os"
	"testing"

	"github.com/vivaui/viva/leaks"
)

// A test that opens a driver, window or native resource must give it back: the run
// fails if any counter is above zero when the tests end.
func TestMain(m *testing.M) { os.Exit(leaks.Main(m)) }

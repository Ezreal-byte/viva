package instance

import "os"

// A Windows temporary directory belongs to the user already.
func ownedByUser(os.FileInfo) bool { return true }

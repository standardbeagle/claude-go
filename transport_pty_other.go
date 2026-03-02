//go:build !unix

package claude

import (
	"fmt"
	"os"
)

// openPTYPair is a stub for non-Unix platforms. PTY stdout is not supported;
// the caller falls back to standard pipes.
func openPTYPair() (master, slave *os.File, err error) {
	return nil, nil, fmt.Errorf("PTY stdout not supported on this platform")
}

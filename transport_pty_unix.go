//go:build unix

package claude

import (
	"os"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// openPTYPair creates a PTY master/slave pair with raw mode on the slave.
// Raw mode prevents the line discipline from converting \n to \r\n in output,
// keeping JSON lines clean for parsing. The caller must close the slave after
// the child process has started (it inherits the fd via cmd.Stdout).
func openPTYPair() (master, slave *os.File, err error) {
	ptmx, pts, err := pty.Open()
	if err != nil {
		return nil, nil, err
	}

	// Set raw mode on slave to disable output processing (OPOST/ONLCR).
	// We don't need to restore since the slave is passed to the child and closed.
	if _, err := term.MakeRaw(int(pts.Fd())); err != nil {
		ptmx.Close()
		pts.Close()
		return nil, nil, err
	}

	return ptmx, pts, nil
}

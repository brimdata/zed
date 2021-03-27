package terminal

import (
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

func IsTerminalFile(f *os.File) bool {
	return terminal.IsTerminal(int(f.Fd()))
}

// Width returns the width in columns of the terminal that is the standard
// input.  Width returns 80 if the standard input is not a terminal or its size
// cannot be determined.
func Width() int {
	width, _, err := terminal.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return 80
	}
	return width
}

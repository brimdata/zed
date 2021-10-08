package color

import "golang.org/x/sys/windows"

// init enables processing of ANSI (aka VT100) escape sequences by Windows
// consoles attached to standard output or standard error.
func init() {
	for _, handle := range [...]windows.Handle{windows.Stdout, windows.Stderr} {
		var mode uint32
		if err := windows.GetConsoleMode(handle, &mode); err == nil {
			windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
		}
	}
}

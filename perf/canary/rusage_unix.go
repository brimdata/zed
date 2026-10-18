//go:build unix

package main

import (
	"os"
	"syscall"
)

// maxRSS returns the peak resident set size of an exited process in bytes.
func maxRSS(ps *os.ProcessState) int64 {
	if ru, ok := ps.SysUsage().(*syscall.Rusage); ok {
		return int64(ru.Maxrss) * maxrssUnit
	}
	return 0
}

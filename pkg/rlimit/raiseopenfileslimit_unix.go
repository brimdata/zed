// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package rlimit

import (
	"fmt"
	"syscall"
)

func raiseOpenFilesLimit() (int, error) {
	rlimit, err := maxRlimit()
	if err != nil {
		return 0, err
	}
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		return 0, fmt.Errorf("setrlimit: %w", err)
	}
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		return 0, fmt.Errorf("getrlimit: %w", err)
	}
	return int(rlimit.Cur), nil
}

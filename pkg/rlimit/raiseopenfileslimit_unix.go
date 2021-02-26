// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package rlimit

import (
	"fmt"
	"syscall"
)

func raiseOpenFilesLimit() (uint64, error) {
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
	return uint64(rlimit.Cur), nil
}

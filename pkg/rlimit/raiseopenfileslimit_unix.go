// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package rlimit

import (
	"fmt"
	"syscall"
)

func raiseOpenFilesLimit() (uint64, error) {
	var rlimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		return 0, fmt.Errorf("getrlimit: %w", err)
	}
	if err := kernMaxFiles(&rlimit); err != nil {
		return 0, err
	}
	rlimit.Cur = rlimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		return 0, fmt.Errorf("setrlimit: %w", err)
	}
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		return 0, fmt.Errorf("getrlimit: %w", err)
	}
	return uint64(rlimit.Cur), nil
}

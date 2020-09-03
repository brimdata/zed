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
	var err error
	rlimit.Cur, err = kernMaxFiles(rlimit.Max)
	if err != nil {
		return 0, err
	}
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		return 0, fmt.Errorf("setrlimit: %w", err)
	}
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		return 0, fmt.Errorf("getrlimit: %w", err)
	}
	return rlimit.Cur, nil
}

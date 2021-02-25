// +build !darwin,!plan9,!windows

package rlimit

import (
	"fmt"
	"syscall"
)

func maxRlimit() (syscall.Rlimit, error) {
	var rlimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		return rlimit, fmt.Errorf("getrlimit: %w", err)
	}
	rlimit.Cur = rlimit.Max
	return rlimit, nil
}

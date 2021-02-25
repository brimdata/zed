package rlimit

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

func maxRlimit() (syscall.Rlimit, error) {
	var rlimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		return rlimit, fmt.Errorf("getrlimit: %w", err)
	}
	var err error
	rlimit.Cur, err = kernMaxFiles(rlimit.Max)
	return rlimit, err
}

func kernMaxFiles(max uint64) (uint64, error) {
	kernMax, err := unix.SysctlUint32("kern.maxfilesperproc")
	if err != nil {
		return 0, fmt.Errorf("systcl: %w", err)
	}
	if uint64(kernMax) < max {
		return uint64(kernMax), nil
	}
	return max, nil
}

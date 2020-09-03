package rlimit

import (
	"fmt"

	"golang.org/x/sys/unix"
)

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

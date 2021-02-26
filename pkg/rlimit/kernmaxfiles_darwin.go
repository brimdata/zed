package rlimit

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

func kernMaxFiles(rlimit *syscall.Rlimit) error {
	kernMax, err := unix.SysctlUint32("kern.maxfilesperproc")
	if err != nil {
		return fmt.Errorf("systcl: %w", err)
	}
	if uint64(kernMax) < rlimit.Max {
		rlimit.Max = uint64(kernMax)
	}
	return nil
}

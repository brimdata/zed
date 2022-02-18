//go:build !darwin && !js && !plan9 && !windows
// +build !darwin,!js,!plan9,!windows

package rlimit

import "syscall"

func kernMaxFiles(rlimit *syscall.Rlimit) error {
	return nil
}

//go:build plan9 || windows
// +build plan9 windows

package rlimit

func raiseOpenFilesLimit() (uint64, error) {
	return 0, nil
}

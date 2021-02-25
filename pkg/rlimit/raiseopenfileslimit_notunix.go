// +build plan9 windows

package rlimit

func raiseOpenFilesLimit() (int, error) {
	return 0, nil
}

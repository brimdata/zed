// Package rlimit provides a single function, RaiseOpenFilesLimit.
package rlimit

// RaiseOpenFilesLimit on Unix raises the current process's soft limit on the
// number of open files to its hard limit and returns the limit.
// On other operating systems, RaiseOpenFilesLimit does nothing.
func RaiseOpenFilesLimit() (int, error) {
	return raiseOpenFilesLimit()
}

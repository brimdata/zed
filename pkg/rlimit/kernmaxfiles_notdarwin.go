// +build !darwin

package rlimit

func kernMaxFiles(max uint64) (uint64, error) {
	return max, nil
}

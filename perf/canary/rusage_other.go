//go:build !unix

package main

import "os"

// maxRSS isn't available on this platform, so peak memory isn't compared.
func maxRSS(*os.ProcessState) int64 {
	return 0
}

// +build !windows

package winexec

import "fmt"

func winexec(_ []string) error {
	return fmt.Errorf("winexec only for windows platforms")
}

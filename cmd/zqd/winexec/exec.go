// +build !windows

package winexec

import "errors"

func winexec(_ []string) error {
	return errors.New("winexec is only available on Windows")
}

package cli

import "strings"

// CommaStringsFlag is a [flag.Value] representing a comma-separated list of
// strings.
type CommaStringsFlag []string

func (c *CommaStringsFlag) String() string {
	if *c == nil {
		return ""
	}
	return strings.Join(*c, ",")
}

func (c *CommaStringsFlag) Set(value string) error {
	if value == "" {
		*c = nil
	} else {
		*c = strings.Split(value, ",")
	}
	return nil
}

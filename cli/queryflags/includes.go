package queryflags

import (
	"os"
	"strings"
)

type Includes []string

func (i Includes) String() string {
	return strings.Join(i, ",")
}

func (i *Includes) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i Includes) Read() ([]string, error) {
	var srcs []string
	for _, path := range i {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		srcs = append(srcs, string(b))
	}
	return srcs, nil
}

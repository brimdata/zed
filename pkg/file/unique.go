package file

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func UniquePath(parent, name string) (string, error) {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	matches, err := filepath.Glob(filepath.Join(parent, base+"*"+ext))
	if err != nil {
		return "", err
	}
	if len(matches) > 0 {
		slice := sort.StringSlice(matches)
		slice.Sort()

		for i := 1; true; i++ {
			name = fmt.Sprintf("%s_%02d%s", base, i, ext)
			if n := slice.Search(base); n == slice.Len() {
				break
			}
		}
	}
	return name, nil
}

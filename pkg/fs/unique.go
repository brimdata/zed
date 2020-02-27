package fs

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// UniquePath determines a unique path given the desired name and parent
// directory. For instance if the desired path is /usr/mypath but such a path
// already exists, the path /usr/mypath_01 is returned (and so on).
// File extensions are respected and the unique number is appended before any
// extensions (e.g. mypath.txt -> mypath_01.txt).
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

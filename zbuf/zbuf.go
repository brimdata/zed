package zbuf

import (
	"github.com/brimdata/zed/zio"
)

func ReadAll(r zio.Reader) (arr Array, err error) {
	if err := zio.Copy(&arr, r); err != nil {
		return nil, err
	}
	return
}

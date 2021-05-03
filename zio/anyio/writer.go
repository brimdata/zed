package anyio

import (
	"github.com/brimdata/zed/zio/textio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zio/zstio"
)

type WriterOpts struct {
	Format string
	UTF8   bool
	Text   textio.WriterOpts
	Zng    zngio.WriterOpts
	ZSON   zsonio.WriterOpts
	Zst    zstio.WriterOpts
}

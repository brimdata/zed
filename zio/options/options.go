package options

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/brimsec/zq/zio/ndjsonio"
	"github.com/brimsec/zq/zio/textio"
	"github.com/brimsec/zq/zio/zngio"
)

type Reader struct {
	Format string
	Zng    zngio.ReaderOpts
	JSON   ndjsonio.ReaderOpts
	AwsCfg *aws.Config
}

type Writer struct {
	Format string
	UTF8   bool
	Text   textio.WriterOpts
	Zng    zngio.WriterOpts
}

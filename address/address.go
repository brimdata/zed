package address

import (
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/scanner"
	"github.com/segmentio/ksuid"
)

type SpanInfo interface {
	GetSpan() nano.Span
	GetChunks() []Chunk
	// SourceOpener provides scanning access to a single source.
	// It works for locals files or S3 (not for accessing zqd/worker).
	// It may return a nil ScannerCloser, in the
	// case that it represents a logically empty source.
	SourceOpener() (scanner.ScannerCloser, error)
}

type Chunk interface {
	GetId() ksuid.KSUID
	GetFirst() nano.Ts
	GetLast() nano.Ts
	GetDataFileKind() string
	GetRecordCount() int
}

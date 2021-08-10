package anyio

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTemp writes the data to a temporary file, and returns its path.
func writeTemp(t *testing.T, data []byte) string {
	f, err := os.CreateTemp("", "")
	require.NoError(t, err)
	_, err = f.Write(data)
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)
	return f.Name()
}

func trim(s string) string {
	return strings.TrimSpace(s) + "\n"
}

func TestMultiFileScanner(t *testing.T) {
	input := []string{
		`
{v:10(int32),ts:1970-01-01T00:00:01Z}(=0)
{v:20,ts:1970-01-01T00:00:02Z}(0)
`,
		`
{v:15(int32),ts:1970-01-01T00:00:04Z}(=0)
{v:25,ts:1970-01-01T00:00:05Z}(0)
`,
	}

	const exp = `
{v:10(int32),ts:1970-01-01T00:00:01Z}(=0)
{v:20,ts:1970-01-01T00:00:02Z}(0)
{v:15,ts:1970-01-01T00:00:04Z}(0)
{v:25,ts:1970-01-01T00:00:05Z}(0)
`

	f1 := writeTemp(t, []byte(input[0]))
	defer os.Remove(f1)
	f2 := writeTemp(t, []byte(input[1]))
	defer os.Remove(f2)

	mfr := MultiFileReader(zson.NewContext(), storage.NewLocalEngine(), []string{f1, f2}, ReaderOpts{})
	sn, err := zbuf.NewScanner(context.Background(), mfr, nil)
	require.NoError(t, err)
	_, ok := sn.(*multiFileScanner)
	assert.True(t, ok)

	var sb strings.Builder
	err = zbuf.CopyPuller(zsonio.NewWriter(zio.NopCloser(&sb), zsonio.WriterOpts{}), sn)
	require.NoError(t, err)
	require.Equal(t, trim(exp), trim(sb.String()))

	expStats := zbuf.ScannerStats{
		BytesRead:      30,
		BytesMatched:   30,
		RecordsRead:    4,
		RecordsMatched: 4,
	}
	require.Equal(t, expStats, sn.Stats())
}

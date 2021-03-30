package detector

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTemp writes the data to a temporary file, and returns its path.
func writeTemp(t *testing.T, data []byte) string {
	f, err := ioutil.TempFile("", "")
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
#0:record[v:int32,ts:time]
0:[10;1;]
0:[20;2;]
`,
		`
#0:record[v:int32,ts:time]
0:[15;4;]
0:[25;5;]
`,
	}

	const exp = `
#0:record[v:int32,ts:time]
0:[10;1;]
0:[20;2;]
0:[15;4;]
0:[25;5;]
`

	f1 := writeTemp(t, []byte(input[0]))
	defer os.Remove(f1)
	f2 := writeTemp(t, []byte(input[1]))
	defer os.Remove(f2)

	mfr := MultiFileReader(resolver.NewContext(), []string{f1, f2}, zio.ReaderOpts{})
	sn, err := zbuf.NewScanner(context.Background(), mfr, nil, nano.MaxSpan)
	require.NoError(t, err)
	_, ok := sn.(*multiFileScanner)
	assert.True(t, ok)

	var sb strings.Builder
	err = zbuf.CopyPuller(tzngio.NewWriter(zio.NopCloser(&sb)), sn)
	require.NoError(t, err)
	require.Equal(t, trim(exp), trim(sb.String()))

	expStats := zbuf.ScannerStats{
		BytesRead:      30,
		BytesMatched:   30,
		RecordsRead:    4,
		RecordsMatched: 4,
	}
	require.Equal(t, expStats, *sn.Stats())
}

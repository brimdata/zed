package archive

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/require"
)

func createArchiveSpace(t *testing.T, datapath string, srcfile string, co *CreateOptions) {
	ark, err := CreateOrOpenArchive(datapath, co, nil)
	require.NoError(t, err)

	zctx := resolver.NewContext()
	reader, err := detector.OpenFile(zctx, srcfile, detector.OpenConfig{})
	require.NoError(t, err)
	defer reader.Close()

	err = Import(context.Background(), ark, reader)
	require.NoError(t, err)
}

func indexArchiveSpace(t *testing.T, datapath string, ruledef string) {
	rule, err := NewRule(ruledef)
	require.NoError(t, err)

	ark, err := OpenArchive(datapath, nil)
	require.NoError(t, err)

	err = IndexDirTree(ark, []Rule{*rule}, "_", nil)
	require.NoError(t, err)
}

func findQuery(t *testing.T, ark *Archive, query IndexQuery, opts ...FindOption) string {
	rc, err := FindReadCloser(context.Background(), ark, query, opts...)
	require.NoError(t, err)
	defer rc.Close()

	var buf bytes.Buffer
	w := zbuf.NopFlusher(tzngio.NewWriter(&buf))
	require.NoError(t, zbuf.Copy(w, rc))

	return buf.String()
}

func TestOpenOptions(t *testing.T) {
	datapath, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(datapath)

	thresh := int64(1000)
	createArchiveSpace(t, datapath, "../tests/suite/zdx/babble.tzng", &CreateOptions{
		LogSizeThreshold: &thresh,
	})

	_, err = OpenArchive(datapath, &OpenOptions{
		LogFilter: []string{"foo"},
	})
	require.Error(t, err)
	require.Regexp(t, "no spans", err.Error())

	indexArchiveSpace(t, datapath, ":int64")

	query, err := ParseIndexQuery("", []string{":int64=336"})
	require.NoError(t, err)

	ark1, err := OpenArchive(datapath, nil)
	require.NoError(t, err)
	exp := `
#zfile=string
#0:record[key:int64,_log:zfile]
0:[336;20200422/1587517412.06741443.zng;]
0:[336;20200421/1587508871.06471174.zng;]
`
	out := findQuery(t, ark1, query, AddPath(DefaultAddPathField, false))
	require.Equal(t, test.Trim(exp), out)

	ark2, err := OpenArchive(datapath, &OpenOptions{
		LogFilter: []string{"20200422/1587517412.06741443.zng"},
	})

	exp = `
#zfile=string
#0:record[key:int64,_log:zfile]
0:[336;20200422/1587517412.06741443.zng;]
`
	out = findQuery(t, ark2, query, AddPath(DefaultAddPathField, false))
	require.Equal(t, test.Trim(exp), out)
}

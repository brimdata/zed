package archive

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"testing"

	"github.com/brimsec/zq/pkg/iosrc"
	iosrcmock "github.com/brimsec/zq/pkg/iosrc/mock"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createArchiveSpace(t *testing.T, datapath string, srcfile string, co *CreateOptions) {
	ark, err := CreateOrOpenArchive(datapath, co, nil)
	require.NoError(t, err)

	importTestFile(t, ark, srcfile)
}

func importTestFile(t *testing.T, ark *Archive, srcfile string) {
	zctx := resolver.NewContext()
	reader, err := detector.OpenFile(zctx, srcfile, detector.OpenConfig{})
	require.NoError(t, err)
	defer reader.Close()

	err = Import(context.Background(), ark, zctx, reader)
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

func indexQuery(t *testing.T, ark *Archive, query IndexQuery, opts ...FindOption) string {
	rc, err := FindReadCloser(context.Background(), resolver.NewContext(), ark, query, opts...)
	require.NoError(t, err)
	defer rc.Close()

	var buf bytes.Buffer
	w := tzngio.NewWriter(&buf)
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
	require.Regexp(t, "no logs left after filter", err.Error())

	indexArchiveSpace(t, datapath, ":int64")

	query, err := ParseIndexQuery("", []string{":int64=336"})
	require.NoError(t, err)

	ark1, err := OpenArchive(datapath, nil)
	require.NoError(t, err)
	exp := `
#zfile=string
#0:record[key:int64,count:uint64,_log:zfile]
0:[336;1;20200422/1587518620.0622373.zng;]
0:[336;1;20200421/1587509469.06883172.zng;]
`
	out := indexQuery(t, ark1, query, AddPath(DefaultAddPathField, false))
	require.Equal(t, test.Trim(exp), out)

	ark2, err := OpenArchive(datapath, &OpenOptions{
		LogFilter: []string{"20200422/1587518620.0622373.zng"},
	})
	require.NoError(t, err)

	exp = `
#zfile=string
#0:record[key:int64,count:uint64,_log:zfile]
0:[336;1;20200422/1587518620.0622373.zng;]
`
	out = indexQuery(t, ark2, query, AddPath(DefaultAddPathField, false))
	require.Equal(t, test.Trim(exp), out)
}

func TestImportWhileOpen(t *testing.T) {
	// Create an archive with initial data
	datapath, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(datapath)

	// A large threshold ensures every import will result in a separate log.
	thresh := int64(math.MaxInt64)
	ark1, err := CreateOrOpenArchive(datapath, &CreateOptions{
		LogSizeThreshold: &thresh,
	}, nil)
	require.NoError(t, err)

	// Verify initial update count.
	update1, err := ark1.UpdateCheck()
	require.NoError(t, err)
	assert.Equal(t, 1, update1)

	importTestFile(t, ark1, "testdata/td1.zng")

	// Ensure UpdateCheck has incremented.
	update2, err := ark1.UpdateCheck()
	require.NoError(t, err)
	if !assert.Equal(t, 3, update2) {
		if fi, err := iosrc.Stat(ark1.mdURI()); err == nil {
			fmt.Fprintf(os.Stderr, "metadata mtime: %v, mdModTime %v", fi.ModTime(), ark1.mdModTime)
		}
	}

	// Verify data & that a span walk now does not increment the update counter.
	var initialSpans []SpanInfo
	err = SpanWalk(ark1, func(si SpanInfo, _ iosrc.URI) error {
		initialSpans = append(initialSpans, si)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, ark1.mdUpdateCount)
	exp := []SpanInfo{SpanInfo{
		Span:  nano.Span{Ts: 1587509776063858170, Dur: 4287004687211},
		LogID: "20200422/1587514063.06854538.zng"}}
	assert.Equal(t, exp, initialSpans)

	// With a separate handle, open & import more data to the archive
	ark2, err := OpenArchive(datapath, nil)
	require.NoError(t, err)

	importTestFile(t, ark2, "testdata/td2.zng")

	// Verify that the data appears to the earlier opened handle
	var postSpans []SpanInfo
	err = SpanWalk(ark1, func(si SpanInfo, _ iosrc.URI) error {
		postSpans = append(postSpans, si)
		return nil
	})
	require.NoError(t, err)

	exp = []SpanInfo{{
		Span:  nano.Span{Ts: 1587514075061481960, Dur: 4545000755341},
		LogID: "20200422/1587518620.0622373.zng",
	}, {
		Span:  nano.Span{Ts: 1587509776063858170, Dur: 4287004687211},
		LogID: "20200422/1587514063.06854538.zng",
	}}
	assert.Equal(t, exp, postSpans)

	if !assert.Equal(t, 4, ark1.mdUpdateCount) {
		if fi, err := iosrc.Stat(ark1.mdURI()); err == nil {
			fmt.Fprintf(os.Stderr, "metadata mtime: %v, ark1.mdModTime %v, ark2.mdModTime %v", fi.ModTime(), ark1.mdModTime, ark2.mdModTime)
		}
	}
}

func TestRemoteSourceImport(t *testing.T) {
	root, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(root)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	src := iosrcmock.NewMockSource(ctrl)
	var recvuri iosrc.URI
	src.EXPECT().NewWriter(gomock.Any()).
		DoAndReturn(func(uri iosrc.URI) (io.WriteCloser, error) {
			recvuri = uri
			return &nopWriteCloser{}, nil
		})
	thresh := int64(math.MaxInt64)
	co := &CreateOptions{
		DataPath:         "s3://test-bucket/test-key",
		LogSizeThreshold: &thresh,
	}
	ark, err := CreateOrOpenArchive(root, co, &OpenOptions{DataSource: src})
	require.NoError(t, err)
	importTestFile(t, ark, "testdata/td1.zng")

	assert.Equal(t, "s3://test-bucket/test-key/20200422/1587514063.06854538.zng", recvuri.String())
}

type nopWriteCloser struct{}

func (*nopWriteCloser) Close() error                { return nil }
func (*nopWriteCloser) Write(b []byte) (int, error) { return len(b), nil }

package emitter

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/brimdata/zed/pkg/iosrc"
	iosrcmock "github.com/brimdata/zed/pkg/iosrc/mock"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestDirS3Source(t *testing.T) {
	path := "s3://testbucket/dir"
	tzng := `
#0:record[_path:string,foo:string]
0:[conn;1;]
#1:record[_path:string,bar:string]
1:[http;2;]`
	uri, err := iosrc.ParseURI(path)
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	src := iosrcmock.NewMockSource(ctrl)

	src.EXPECT().NewWriter(context.Background(), uri.AppendPath("conn.tzng")).
		Return(&nopCloser{bytes.NewBuffer(nil)}, nil)
	src.EXPECT().NewWriter(context.Background(), uri.AppendPath("http.tzng")).
		Return(&nopCloser{bytes.NewBuffer(nil)}, nil)

	r := tzngio.NewReader(strings.NewReader(tzng), resolver.NewContext())
	require.NoError(t, err)
	w, err := NewDirWithSource(context.Background(), uri, "", os.Stderr, zio.WriterOpts{Format: "tzng"}, src)
	require.NoError(t, err)
	require.NoError(t, zbuf.Copy(w, r))
}

type nopCloser struct{ *bytes.Buffer }

func (nopCloser) Close() error { return nil }

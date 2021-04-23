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
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zson"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestDirS3Source(t *testing.T) {
	path := "s3://testbucket/dir"
	const input = `
{_path:"conn",foo:"1"}
{_path:"http",bar:"2"}
`
	uri, err := iosrc.ParseURI(path)
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	src := iosrcmock.NewMockSource(ctrl)

	src.EXPECT().NewWriter(context.Background(), uri.AppendPath("conn.zson")).
		Return(&nopCloser{bytes.NewBuffer(nil)}, nil)
	src.EXPECT().NewWriter(context.Background(), uri.AppendPath("http.zson")).
		Return(&nopCloser{bytes.NewBuffer(nil)}, nil)

	r := zson.NewReader(strings.NewReader(input), zson.NewContext())
	require.NoError(t, err)
	w, err := NewDirWithSource(context.Background(), uri, "", os.Stderr, anyio.WriterOpts{Format: "zson"}, src)
	require.NoError(t, err)
	require.NoError(t, zbuf.Copy(w, r))
}

type nopCloser struct{ *bytes.Buffer }

func (nopCloser) Close() error { return nil }

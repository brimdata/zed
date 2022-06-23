package emitter

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	storagemock "github.com/brimdata/zed/pkg/storage/mock"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestDirS3Source(t *testing.T) {
	path := "s3://testbucket/dir"
	const input = `
{_path:"conn",foo:"1"}
{_path:"http",bar:"2"}
`
	uri, err := storage.ParseURI(path)
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	engine := storagemock.NewMockEngine(ctrl)

	engine.EXPECT().Put(context.Background(), uri.AppendPath("conn.zson")).
		Return(&nopCloser{bytes.NewBuffer(nil)}, nil)
	engine.EXPECT().Put(context.Background(), uri.AppendPath("http.zson")).
		Return(&nopCloser{bytes.NewBuffer(nil)}, nil)

	r := zsonio.NewReader(zed.NewContext(), strings.NewReader(input))
	require.NoError(t, err)
	w, err := NewSplit(context.Background(), engine, uri, "", anyio.WriterOpts{Format: "zson"})
	require.NoError(t, err)
	require.NoError(t, zio.Copy(w, r))
}

type nopCloser struct{ *bytes.Buffer }

func (nopCloser) Close() error { return nil }

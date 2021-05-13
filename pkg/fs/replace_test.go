package fs

import (
	"errors"
	"io"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplaceFileAbort(t *testing.T) {
	fname := path.Join(t.TempDir(), "file1")
	data1 := "data1"
	err := ioutil.WriteFile(fname, []byte(data1), 0666)
	require.NoError(t, err)

	fakeErr := errors.New("fake error")
	err = ReplaceFile(fname, 0666, func(w io.Writer) error {
		_, err := w.Write([]byte("data2"))
		if err != nil {
			t.Fatal("replace write unexpectedly failed")
		}
		return fakeErr
	})
	require.Error(t, err)
	require.Equal(t, fakeErr.Error(), err.Error())

	b, err := ioutil.ReadFile(fname)
	require.NoError(t, err)
	require.Equal(t, data1, string(b))
}

package fs

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTailFile(t *testing.T) {
	f, err := ioutil.TempFile("", "tailfile.log")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	tf, err := TailFile(f.Name())
	require.NoError(t, err)
	buf := make([]byte, 100)

	for i := 0; i < 10; i++ {
		str := fmt.Sprintf("line #%d\n", i)
		_, err := f.WriteString(str)
		require.NoError(t, err)
		n, err := tf.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, str, string(buf[:n]))
	}
	go require.NoError(t, tf.Stop())
	n, err := tf.Read(buf)
	assert.Equal(t, 0, n)
	assert.Error(t, io.EOF, err)
}

func TestTailFileReadToEOF(t *testing.T) {
	expected := `line #0
line #1
line #2
line #3
line #4
line #5
line #6
line #7
line #8
line #9
`
	f, err := ioutil.TempFile("", "tailfile.log")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	tf, err := TailFile(f.Name())
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		str := fmt.Sprintf("line #%d\n", i)
		_, err := f.WriteString(str)
		require.NoError(t, err)
	}
	require.NoError(t, tf.Stop())
	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, tf)
	require.NoError(t, err)
	assert.Equal(t, expected, buf.String())
}

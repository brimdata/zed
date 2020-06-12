package s3io

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteInvalidPath(t *testing.T) {
	_, err := NewWriter("http://localhost/upload", nil)
	require.Equal(t, ErrInvalidS3Path, err)
}

func TestWriteSimple(t *testing.T) {
	results := bytes.NewBuffer(nil)
	expected := []byte("some test data")
	w, _ := NewWriter("s3://localhost/upload", nil)
	up := func(in *s3manager.UploadInput, _ ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
		_, err := io.Copy(results, in.Body)
		return &s3manager.UploadOutput{}, err
	}
	w.uploader = mockUploader(up)
	_, err := w.Write(expected)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.Equal(t, expected, results.Bytes())

}

func TestWriteImmediateError(t *testing.T) {
	expected := errors.New("expected error")
	w, _ := NewWriter("s3://localhost/upload", nil)
	w.uploader = mockUploader(func(in *s3manager.UploadInput, _ ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
		// _, err := io.Copy(results, in.Body)
		return &s3manager.UploadOutput{}, expected
	})
	_, err := w.Write([]byte("test data"))
	assert.Equal(t, expected, err)
	assert.Equal(t, expected, w.Close())
}

func TestWriteEventualError(t *testing.T) {
	data := []byte("test data")
	expected := errors.New("expected error")
	w, _ := NewWriter("s3://localhost/upload", nil)
	w.uploader = mockUploader(func(in *s3manager.UploadInput, _ ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
		buf := make([]byte, len(data))
		_, _ = in.Body.Read(buf)
		return &s3manager.UploadOutput{}, expected
	})
	_, err := w.Write(data)
	require.NoError(t, err)
	_, err = w.Write(data)
	assert.Equal(t, expected, err)
	assert.Equal(t, expected, w.Close())
}

type mockUploader func(*s3manager.UploadInput, ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)

func (m mockUploader) Upload(in *s3manager.UploadInput, opts ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	return m(in, opts...)
}

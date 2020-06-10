package s3io

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alecthomas/units"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/brimsec/zq/pkg/s3io/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteInvalidPath(t *testing.T) {
	_, err := NewWriter("http://localhost/upload", nil)
	require.Equal(t, ErrInvalidScheme, err)
}

func TestWriteSimple(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish() // once we upgrade to golang 1.14 we can get rid of this.
	expected := []byte("some test data")

	req := httptest.NewRequest(http.MethodPost, "http://localhost/upload", nil)
	client := mocks.NewMockS3API(ctrl)
	client.EXPECT().PutObjectRequest(gomock.Any()).
		DoAndReturn(func(input *s3.PutObjectInput) (*request.Request, *s3.PutObjectOutput) {
			out, err := ioutil.ReadAll(input.Body)
			require.NoError(t, err)
			assert.Equal(t, expected, out)
			assert.Equal(t, aws.String("localhost"), input.Bucket)
			assert.Equal(t, aws.String("/upload.zng"), input.Key)
			return &request.Request{HTTPRequest: req}, &s3.PutObjectOutput{}
		})

	w, err := NewWriterWithClient("s3://localhost/upload.zng", client)
	require.NoError(t, err)
	_, err = w.Write(expected)
	require.NoError(t, err)
	require.NoError(t, w.Close())
}

func TestWriteErrorSimple(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	expected := errors.New("something went wrong")

	req := &request.Request{
		HTTPRequest: httptest.NewRequest(http.MethodPost, "http://localhost/upload", nil),
		Error:       expected,
	}
	client := mocks.NewMockS3API(ctrl)
	client.EXPECT().PutObjectRequest(gomock.Any()).
		Return(req, &s3.PutObjectOutput{})

	w, err := NewWriterWithClient("s3://localhost/upload.zng", client)
	require.NoError(t, err)
	_, err = w.Write([]byte("some test data"))
	require.NoError(t, err)
	require.Equal(t, expected, w.Close())
}

func TestWriteMultipartErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	expected := errors.New("something wrong happend")
	uploadID := aws.String("testID")
	client := mocks.NewMockS3API(ctrl)
	client.EXPECT().CreateMultipartUploadWithContext(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&s3.CreateMultipartUploadOutput{UploadId: uploadID}, nil)
	client.EXPECT().UploadPartWithContext(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&s3.UploadPartOutput{}, expected)
	client.EXPECT().AbortMultipartUploadWithContext(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&s3.AbortMultipartUploadOutput{}, nil)

	w, err := NewWriterWithClient("s3://localhost/upload.zng", client)
	require.NoError(t, err)
	reader := generateTestReader(t, "11MB") // 11MB needs to trigger multipart upload
	_, err = io.Copy(w, reader)
	require.Error(t, err)
	assert.Regexp(t, expected.Error(), err.Error())
	closeErr := w.Close()
	require.Error(t, closeErr)
	assert.Regexp(t, expected.Error(), closeErr.Error())
}

func generateTestReader(t *testing.T, sizeStr string) io.Reader {
	size, err := units.ParseStrictBytes(sizeStr)
	require.NoError(t, err)
	data := []byte("some test data\n")
	n := int(size / int64(len(data)))
	buf := bytes.Repeat(data, n)
	return bytes.NewReader(buf)
}

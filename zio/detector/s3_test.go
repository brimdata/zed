package detector

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	minio "github.com/minio/minio/cmd"
	"github.com/minio/minio/pkg/madmin"
	"github.com/stretchr/testify/require"
)

func startServer(t *testing.T, dir string) *madmin.AdminClient {
	go func() { minio.Main([]string{"--address localhost:9000", "server", dir}) }()

	mcli, err := madmin.New("localhost:9000", "minioadmin", "minioadmin", false)
	require.NoError(t, err)
	return mcli
}

func waitForServer(t *testing.T, cli *madmin.AdminClient) {
	var err error
	for i := 0; i < 10; i++ {
		_, err = cli.ServerInfo(context.Background())
		if err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.FailNow(t, fmt.Sprintf("minio server did not come up: %s\n", err))
}

func stopServer(t *testing.T, cli *madmin.AdminClient) {
	err := cli.ServiceStop(context.Background())
	require.NoError(t, err)
}

func loadFile(t *testing.T, datadir, bucket, name string, data []byte) {
	bucketDir := path.Join(datadir, bucket)
	err := os.MkdirAll(bucketDir, 0700)
	require.NoError(t, err)
	ioutil.WriteFile(path.Join(bucketDir, name), data, 0644)
}

func TestS3Minio(t *testing.T) {
	lines := []string{
		"#0:record[ts:time,uid:bstring]",
		"0:[1521911721.255387;C8Tful1TvM3Zf5x8fl;]",
		"0:[1521911721.411148;CXWfTK3LRdiuQxBbM6;]",
	}
	dir, err := ioutil.TempDir("", "s3test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	loadFile(t, dir, "brim", "conn.tzng", []byte(strings.Join(lines, "\n")))

	dnsParquet, err := ioutil.ReadFile("testdata/dns.parquet")
	require.NoError(t, err)
	loadFile(t, dir, "brim", "dns.parquet", dnsParquet)

	dnsZng, err := ioutil.ReadFile("testdata/dns.zng")
	require.NoError(t, err)
	loadFile(t, dir, "brim", "dns.zng", dnsZng)

	mcli := startServer(t, dir)
	waitForServer(t, mcli)

	err = os.Setenv("AWS_ACCESS_KEY_ID", "minioadmin")
	require.NoError(t, err)
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")

	err = os.Setenv("AWS_SECRET_ACCESS_KEY", "minioadmin")
	require.NoError(t, err)
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	cfg := OpenConfig{
		AwsCfg: &aws.Config{
			Endpoint:         aws.String("http://localhost:9000"),
			Region:           aws.String("us-east-2"),
			S3ForcePathStyle: aws.Bool(true), // https://github.com/minio/minio/tree/master/docs/config#domain
		},
	}

	t.Run("Read single file", func(t *testing.T) {
		var out bytes.Buffer
		f, err := OpenFile(resolver.NewContext(), "s3://brim/conn.tzng", cfg)
		require.NoError(t, err)
		defer f.Close()

		w := tzngio.NewWriter(&out)
		err = zbuf.Copy(zbuf.NopFlusher(w), f)
		require.NoError(t, err)
		require.Equal(t, strings.Join(lines, "\n"), strings.TrimSpace(out.String()))
	})

	t.Run("Combine multiple files", func(t *testing.T) {
		var out bytes.Buffer
		f1, err := OpenFile(resolver.NewContext(), "s3://brim/conn.tzng", cfg)
		require.NoError(t, err)
		defer f1.Close()
		f2, err := OpenFile(resolver.NewContext(), "s3://brim/conn.tzng", cfg)
		require.NoError(t, err)
		defer f2.Close()

		c := zbuf.NewCombiner([]zbuf.Reader{f1, f2})

		w := tzngio.NewWriter(&out)
		err = zbuf.Copy(zbuf.NopFlusher(w), c)
		require.NoError(t, err)

		expected := strings.Join([]string{lines[0], lines[1], lines[1], lines[2], lines[2]}, "\n")
		require.Equal(t, expected, strings.TrimSpace(out.String()))
	})
	t.Run("Read parquet file", func(t *testing.T) {
		var out bytes.Buffer
		cfgP := cfg
		cfgP.Format = "parquet"
		f, err := OpenFile(resolver.NewContext(), "s3://brim/dns.parquet", cfgP)
		require.NoError(t, err)
		defer f.Close()

		w := zngio.NewWriter(&out, zio.WriterFlags{})
		err = zbuf.Copy(w, f)
		require.NoError(t, err)
		dnsZng, err := ioutil.ReadFile("testdata/dns.zng")
		require.NoError(t, err)
		require.Equal(t, dnsZng, out.Bytes())
	})
	stopServer(t, mcli)
}

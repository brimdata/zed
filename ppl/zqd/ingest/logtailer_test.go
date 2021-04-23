package ingest

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/suite"
)

var sortTs = compiler.MustParseProc("sort ts")

const expected = `{ts:1970-01-01T00:00:00Z}
{ts:1970-01-01T00:00:01Z}
{ts:1970-01-01T00:00:02Z}
{ts:1970-01-01T00:00:03Z}
{ts:1970-01-01T00:00:04Z}
{ts:1970-01-01T00:00:05Z}
{ts:1970-01-01T00:00:06Z}
{ts:1970-01-01T00:00:07Z}
{ts:1970-01-01T00:00:08Z}
{ts:1970-01-01T00:00:09Z}
{ts:1970-01-01T00:00:10Z}
{ts:1970-01-01T00:00:11Z}
{ts:1970-01-01T00:00:12Z}
{ts:1970-01-01T00:00:13Z}
{ts:1970-01-01T00:00:14Z}
{ts:1970-01-01T00:00:15Z}
{ts:1970-01-01T00:00:16Z}
{ts:1970-01-01T00:00:17Z}
{ts:1970-01-01T00:00:18Z}
{ts:1970-01-01T00:00:19Z}
`

type logTailerTSuite struct {
	suite.Suite
	dir  string
	zctx *zson.Context
	dr   *logTailer
}

func TestLogTailer(t *testing.T) {
	suite.Run(t, new(logTailerTSuite))
}

func (s *logTailerTSuite) SetupTest() {
	dir, err := ioutil.TempDir("", "TestLogTailer")
	s.Require().NoError(err)
	s.dir = dir
	s.T().Cleanup(func() { os.RemoveAll(s.dir) })
	s.zctx = zson.NewContext()
	s.dr, err = newLogTailer(s.zctx, s.dir, anyio.ReaderOpts{Format: "zson"})
	s.Require().NoError(err)
}

func (s *logTailerTSuite) TestCreatedFiles() {
	result, errCh := s.read()
	f1 := s.createFile("test1")
	f2 := s.createFile("test2")
	s.write(f1, f2)
	s.Require().NoError(<-errCh)
	s.Equal(expected, <-result)
}

func (s *logTailerTSuite) TestIgnoreDir() {
	result, errCh := s.read()
	f1 := s.createFile("test1")
	f2 := s.createFile("test2")
	err := os.Mkdir(filepath.Join(s.dir, "testdir"), 0755)
	s.Require().NoError(err)
	s.write(f1, f2)
	s.Require().NoError(<-errCh)
	s.Equal(expected, <-result)
}

func (s *logTailerTSuite) TestExistingFiles() {
	f1 := s.createFile("test1")
	f2 := s.createFile("test2")
	result, errCh := s.read()
	s.write(f1, f2)
	s.Require().NoError(<-errCh)
	s.Equal(expected, <-result)
}

func (s *logTailerTSuite) TestInvalidFile() {
	_, errCh := s.read()
	f1 := s.createFile("test1")
	_, err := f1.WriteString("this is an invalid line\n")
	s.Require().NoError(err)
	s.Require().NoError(f1.Sync())
	s.EqualError(<-errCh, `identifier "this" must be enum and requires decorator`)
	s.NoError(s.dr.Stop())
}

func (s *logTailerTSuite) TestEmptyFile() {
	result, errCh := s.read()
	f1 := s.createFile("test1")
	_ = s.createFile("test2")
	s.write(f1)
	s.Require().NoError(<-errCh)
	s.Equal(expected, <-result)
}

func (s *logTailerTSuite) createFile(name string) *os.File {
	f, err := os.Create(filepath.Join(s.dir, name))
	s.Require().NoError(err)
	// Call sync to ensure fs events are sent in a timely matter.
	s.Require().NoError(f.Sync())
	return f
}

func (s *logTailerTSuite) read() (<-chan string, <-chan error) {
	result := make(chan string)
	errCh := make(chan error)
	buf := bytes.NewBuffer(nil)
	w := zsonio.NewWriter(zio.NopCloser(buf), zsonio.WriterOpts{})
	go func() {
		err := driver.Copy(context.Background(), w, sortTs, s.zctx, s.dr, driver.Config{})
		if err != nil {
			close(result)
			errCh <- err
		} else {
			close(errCh)
			result <- buf.String()
		}
	}()
	return result, errCh
}

func (s *logTailerTSuite) write(files ...*os.File) {
	for i := 0; i < 20; {
		for _, f := range files {
			_, err := f.WriteString(fmt.Sprintf("{ts:%s}\n", nano.Unix(int64(i), 0)))
			s.Require().NoError(err)
			i++
		}
	}
	// Need to sync here as on windows the fsnotify event is not triggered
	// unless this is done. Presumably this happens in cases when not enough
	// data has been written so the system has not flushed the file buffer to disk.
	for _, f := range files {
		s.Require().NoError(f.Sync())
	}
	s.Require().NoError(s.dr.Stop())
}

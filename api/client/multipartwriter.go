package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"sync"
	"sync/atomic"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zio/ndjsonio"
)

type MultipartWriter struct {
	BytesTotal int64

	bytesRead int64
	form      *multipart.Writer
	pr        *io.PipeReader
	pw        *io.PipeWriter
	start     sync.Once
	readers   []io.Reader
	uris      []iosrc.URI
	json      *ndjsonio.TypeConfig
}

func newMultipartWriter() *MultipartWriter {
	pr, pw := io.Pipe()
	form := multipart.NewWriter(pw)
	return &MultipartWriter{form: form, pr: pr, pw: pw}
}

func MultipartFileWriter(files ...string) (*MultipartWriter, error) {
	lw := newMultipartWriter()
	for _, f := range files {
		u, err := iosrc.ParseURI(f)
		if err != nil {
			return nil, err
		}
		info, err := iosrc.Stat(context.Background(), u)
		if err != nil {
			return nil, err
		}
		lw.BytesTotal += info.Size()
		lw.uris = append(lw.uris, u)
	}
	return lw, nil
}

func MultipartDataWriter(readers ...io.Reader) (*MultipartWriter, error) {
	pr, pw := io.Pipe()
	form := multipart.NewWriter(pw)
	lw := &MultipartWriter{form: form, pr: pr, pw: pw}
	lw.readers = readers
	return lw, nil
}

func (l *MultipartWriter) SetJSONConfig(config *ndjsonio.TypeConfig) {
	l.json = config
}

func (l *MultipartWriter) ContentType() string {
	return l.form.FormDataContentType()
}

func (l *MultipartWriter) BytesRead() int64 {
	return atomic.LoadInt64(&l.bytesRead)
}

func (l *MultipartWriter) Read(b []byte) (int, error) {
	l.start.Do(func() {
		go l.run()
	})
	return l.pr.Read(b)
}

func (l *MultipartWriter) run() {
	if err := l.sendJSONConfig(); err != nil {
		l.pw.CloseWithError(err)
		return
	}
	for _, u := range l.uris {
		if err := l.writeFile(u); err != nil {
			l.pw.CloseWithError(err)
			return
		}
	}
	for i, r := range l.readers {
		if err := l.write(fmt.Sprintf("data%d", i+1), r); err != nil {
			l.pw.CloseWithError(err)
			return
		}
	}
	l.pw.CloseWithError(l.form.Close())
}

func (l *MultipartWriter) writeFile(u iosrc.URI) error {
	r, err := iosrc.NewReader(context.Background(), u)
	if err != nil {
		return err
	}
	defer r.Close()
	return l.write(u.String(), r)
}

func (l *MultipartWriter) write(name string, r io.Reader) error {
	w, err := l.form.CreateFormFile("", name)
	if err != nil {
		return err
	}
	c := &counter{reader: bufio.NewReader(r), nread: &l.bytesRead}
	_, err = io.Copy(w, c)
	return err
}

func (l *MultipartWriter) sendJSONConfig() error {
	if l.json == nil {
		return nil
	}
	w, err := l.form.CreateFormField("json_config")
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(l.json)
}

type counter struct {
	reader io.Reader
	nread  *int64
}

func (r *counter) Read(b []byte) (int, error) {
	n, err := r.reader.Read(b)
	atomic.AddInt64(r.nread, int64(n))
	return n, err
}

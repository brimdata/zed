package zbuf

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
)

type Control struct {
	Message interface{}
}

var _ error = (*Control)(nil)

func (c *Control) Error() string {
	return "control"
}

type SetChannel int
type EndChannel int

type noControl struct {
	zio.Reader
}

func NoControl(r zio.Reader) *noControl {
	return &noControl{Reader: r}
}

func (n *noControl) Read() (*zed.Value, error) {
	for {
		val, err := n.Reader.Read()
		if _, ok := err.(*Control); ok {
			continue
		}
		return val, err
	}
}

type ProgressReader interface {
	zio.Reader
	Progress() Progress
}

type ProgressReadCloser interface {
	zio.ReadCloser
	Progress() Progress
}

func MeterReadCloser(rc zio.ReadCloser) ProgressReadCloser {
	return &meterReadCloser{ReadCloser: rc}
}

type meterReadCloser struct {
	zio.ReadCloser
	progress Progress
}

func (m *meterReadCloser) Progress() Progress {
	return m.progress.Copy()
}

func (m *meterReadCloser) Read() (*zed.Value, error) {
	val, err := m.ReadCloser.Read()
	if ctrl, ok := err.(*Control); ok {
		if progress, ok := ctrl.Message.(Progress); ok {
			m.progress = progress
		}
	}
	return val, err
}

func ReadAll(r zio.Reader) (arr *Array, err error) {
	if err := zio.Copy(arr, r); err != nil {
		return nil, err
	}
	return
}

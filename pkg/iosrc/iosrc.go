//go:generate mockgen -destination=./mock/mock_source.go -package=mock github.com/brimsec/zq/pkg/iosrc Source

package iosrc

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const FileScheme = "file"

var DefaultRegistry = &Registry{
	schemes: map[string]Source{
		"file":  DefaultFileSource,
		"stdio": defaultStdioSource,
	},
}

type Source interface {
	NewReader(URI) (io.ReadCloser, error)
	NewWriter(URI) (io.WriteCloser, error)
	Remove(URI) error
	RemoveAll(URI) error
	// Exists returns true if the specified uri exists and an error is there
	// was an error finding this information.
	Exists(URI) (bool, error)
	Stat(URI) (Info, error)
}

type Info interface {
	Size() int64
	ModTime() time.Time
}

type DirMaker interface {
	MkdirAll(URI, os.FileMode) error
}

// A Replaceable source supports atomic updates to a URI.
type Replaceable interface {
	NewReplacer(URI) (io.WriteCloser, error)
}

type Registry struct {
	mu      sync.RWMutex
	schemes map[string]Source
}

func (r *Registry) initWithLock() {
	if r.schemes == nil {
		r.schemes = map[string]Source{}
	}
}

func (r *Registry) Add(scheme string, loader Source) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.initWithLock()
	r.schemes[scheme] = loader
}

func (r *Registry) NewReader(uri URI) (io.ReadCloser, error) {
	s, err := r.Source(uri)
	if err != nil {
		return nil, err
	}
	return s.NewReader(uri)
}

func (r *Registry) NewWriter(uri URI) (io.WriteCloser, error) {
	s, err := r.Source(uri)
	if err != nil {
		return nil, err
	}
	return s.NewWriter(uri)
}

func (r *Registry) Source(uri URI) (Source, error) {
	scheme := getScheme(uri)
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.initWithLock()
	loader, ok := r.schemes[scheme]
	if !ok {
		return nil, fmt.Errorf("unknown scheme: %q", scheme)
	}
	return loader, nil
}

func (r *Registry) GetScheme(uri URI) (string, bool) {
	scheme := getScheme(uri)
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.initWithLock()
	_, ok := r.schemes[scheme]
	return scheme, ok
}

func Register(scheme string, source Source) {
	DefaultRegistry.Add(scheme, source)
}

func NewReader(uri URI) (io.ReadCloser, error) {
	return DefaultRegistry.NewReader(uri)
}

func NewWriter(uri URI) (io.WriteCloser, error) {
	return DefaultRegistry.NewWriter(uri)
}

func Exists(uri URI) (bool, error) {
	source, err := DefaultRegistry.Source(uri)
	if err != nil {
		return false, nil
	}
	return source.Exists(uri)
}

func Remove(uri URI) error {
	source, err := DefaultRegistry.Source(uri)
	if err != nil {
		return nil
	}
	return source.Remove(uri)
}

func GetSource(uri URI) (Source, error) {
	return DefaultRegistry.Source(uri)
}

func Stat(uri URI) (Info, error) {
	source, err := DefaultRegistry.Source(uri)
	if err != nil {
		return nil, nil
	}
	return source.Stat(uri)
}

func getScheme(uri URI) string {
	if uri.Scheme == "" {
		return FileScheme
	}
	return uri.Scheme
}

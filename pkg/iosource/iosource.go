package iosource

import (
	"errors"
	"io"
	"net/url"
	"sync"
)

const FileScheme = "file"

var (
	DefaultRegistry = &Registry{}
)

func init() {
	resetDefaultRegistry()
}

func resetDefaultRegistry() {
	DefaultRegistry = &Registry{}
	DefaultRegistry.Add(FileScheme, DefaultFileSource)
}

type Registry struct {
	mu      sync.RWMutex
	schemes map[string]Source
}

func (r *Registry) Add(scheme string, loader Source) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.schemes == nil {
		r.schemes = map[string]Source{}
	}
	r.schemes[scheme] = loader
}

func (r *Registry) NewReader(path string) (io.ReadCloser, error) {
	s, err := r.Source(path)
	if err != nil {
		return nil, err
	}
	return s.NewReader(path)
}

func (r *Registry) NewWriter(path string) (io.WriteCloser, error) {
	s, err := r.Source(path)
	if err != nil {
		return nil, err
	}
	return s.NewWriter(path)
}

func (r *Registry) Source(path string) (Source, error) {
	scheme := getScheme(path)
	r.mu.RLock()
	defer r.mu.RUnlock()
	loader, ok := r.schemes[scheme]
	if !ok {
		return nil, errors.New("unknown scheme")
	}
	return loader, nil
}

func (r *Registry) GetScheme(path string) (string, bool) {
	scheme := getScheme(path)
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.schemes[scheme]
	return scheme, ok
}

type Source interface {
	NewReader(path string) (io.ReadCloser, error)
	NewWriter(path string) (io.WriteCloser, error)
}

func Register(scheme string, source Source) {
	DefaultRegistry.Add(scheme, source)
}

func NewReader(path string) (io.ReadCloser, error) {
	return DefaultRegistry.NewReader(path)
}

func NewWriter(path string) (io.WriteCloser, error) {
	return DefaultRegistry.NewWriter(path)
}

func getScheme(path string) string {
	u, _ := url.Parse(path)
	if u == nil {
		return FileScheme
	}
	if u.Scheme == "" {
		return FileScheme
	}
	return u.Scheme
}

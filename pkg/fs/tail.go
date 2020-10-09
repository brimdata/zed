package fs

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

var ErrIsDir = errors.New("path is a directory")

type TFile struct {
	f       *os.File
	watcher *fsnotify.Watcher
}

func TailFile(name string) (*TFile, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, ErrIsDir
	}
	f, err := OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := watcher.Add(name); err != nil {
		return nil, err
	}
	return &TFile{f, watcher}, nil
}

func (t *TFile) Read(b []byte) (int, error) {
read:
	n, err := t.f.Read(b)
	if errors.Is(err, os.ErrClosed) {
		err = io.EOF
	}
	if n == 0 && err == io.EOF {
		if err := t.waitWrite(); err != nil {
			return 0, err
		}
		goto read
	}
	return n, err
}

func (t *TFile) waitWrite() error {
	for {
		select {
		case ev, ok := <-t.watcher.Events:
			if !ok {
				return io.EOF
			}
			if ev.Op == fsnotify.Write {
				return nil
			}
		case err := <-t.watcher.Errors:
			return err
		}
	}
}

func (t *TFile) Stop() error {
	return t.watcher.Close()
}

func (t *TFile) Close() error {
	return t.f.Close()
}

type FileOp int

const (
	FileOpCreated FileOp = iota
	FileOpExisting
	FileOpRemoved
)

func (o FileOp) Exists() bool {
	return o == FileOpCreated || o == FileOpExisting
}

type FileEvent struct {
	Name string
	Op   FileOp
	Err  error
}

// DirWatcher observes a directory and will emit events when files are added
// or removed. When open for the first time this will emit an event for
// every existing file.
type DirWatcher struct {
	dir     string
	events  chan FileEvent
	once    sync.Once
	err     error
	watcher *fsnotify.Watcher
}

func NewDirWatcher(dir string) (*DirWatcher, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("provided path must be a directory")
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &DirWatcher{
		dir:     dir,
		events:  make(chan FileEvent, 5),
		watcher: watcher,
	}, err
}

func (w *DirWatcher) start() {
	defer close(w.events)
	defer w.watcher.Close()
	if err := w.watcher.Add(w.dir); err != nil {
		w.events <- FileEvent{Err: err}
		return
	}
	if err := w.emitExisting(); err != nil {
		w.events <- FileEvent{Err: err}
		return
	}
	for ev := range w.watcher.Events {
		switch ev.Op {
		case fsnotify.Create:
			w.events <- FileEvent{Name: ev.Name, Op: FileOpCreated}
		case fsnotify.Remove:
			w.events <- FileEvent{Name: ev.Name, Op: FileOpRemoved}
		}
	}
}

func (w *DirWatcher) emitExisting() error {
	infos, err := ioutil.ReadDir(w.dir)
	if err != nil {
		return err
	}
	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		w.events <- FileEvent{
			Name: filepath.Join(w.dir, info.Name()),
			Op:   FileOpExisting,
		}
	}
	return nil
}

func (w *DirWatcher) Events() <-chan FileEvent {
	w.once.Do(func() { go w.start() })
	return w.events
}

func (w *DirWatcher) Stop() error {
	return w.watcher.Close()
}

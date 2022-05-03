package service

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/branches"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/service/srverr"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zson"
	"github.com/gorilla/mux"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

type Request struct {
	*http.Request
	Logger *zap.Logger
}

func newRequest(w http.ResponseWriter, r *http.Request, logger *zap.Logger, defaultFormat string) (*ResponseWriter, *Request, bool) {
	logger = logger.With(zap.String("request_id", api.RequestIDFromContext(r.Context())))
	req := &Request{
		Request: r,
		Logger:  logger,
	}
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StylePackage)
	res := &ResponseWriter{
		ResponseWriter: w,
		Logger:         logger,
		marshaler:      m,
	}
	ss := strings.Split(r.Header.Get("Accept"), ",")
	if len(ss) == 0 {
		ss = []string{""}
	}
	for _, mime := range strings.Split(r.Header.Get("Accept"), ",") {
		format, err := api.MediaTypeToFormat(mime, defaultFormat)
		if err != nil {
			continue
		}
		res.Format = format
		return res, req, true
	}
	res.Error(srverr.ErrInvalid("could not find supported mime type in ACCEPT header"))
	return nil, nil, false
}

func (r *Request) PoolID(w *ResponseWriter, root *lake.Root) (ksuid.KSUID, bool) {
	s, ok := r.StringFromPath(w, "pool")
	if !ok {
		return ksuid.Nil, false
	}
	id, err := lakeparse.ParseID(s)
	if err != nil {
		id, err = root.PoolID(r.Context(), s)
		if errors.Is(err, pools.ErrNotFound) {
			w.Error(err)
			return ksuid.Nil, false
		}
		if err != nil {
			w.Error(srverr.ErrInvalid("invalid path param %q: %w", s, err))
			return ksuid.Nil, false
		}
	}
	return id, true
}

func (r *Request) CommitID(w *ResponseWriter) (ksuid.KSUID, bool) {
	return r.TagFromPath("commit", w)
}

func (r *Request) decodeCommitMessage(w *ResponseWriter) (api.CommitMessage, bool) {
	commitJSON := r.Header.Get("Zed-Commit")
	var message api.CommitMessage
	if commitJSON != "" {
		if err := json.Unmarshal([]byte(commitJSON), &message); err != nil {
			w.Error(srverr.ErrInvalid("load endpoint encountered invalid JSON in Zed-Commit header: %w", err))
			return message, false
		}
	}
	return message, true
}

func (r *Request) StringFromPath(w *ResponseWriter, arg string) (string, bool) {
	v := mux.Vars(r.Request)
	s, ok := v[arg]
	if !ok {
		w.Error(srverr.ErrInvalid("no arg %q in path", arg))
		return "", false
	}
	decoded, err := url.QueryUnescape(s)
	return decoded, err == nil
}

func (r *Request) TagFromPath(arg string, w *ResponseWriter) (ksuid.KSUID, bool) {
	v := mux.Vars(r.Request)
	s, ok := v[arg]
	if !ok {
		w.Error(srverr.ErrInvalid("no arg %q in path", arg))
		return ksuid.Nil, false
	}
	id, err := lakeparse.ParseID(s)
	if err != nil {
		w.Error(srverr.ErrInvalid("invalid path param %q: %w", arg, err))
		return ksuid.Nil, false
	}
	return id, true
}

func (r *Request) JournalIDFromQuery(param string, w *ResponseWriter) (journal.ID, bool) {
	s := r.URL.Query().Get(param)
	if s == "" {
		return journal.Nil, true
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		w.Error(srverr.ErrInvalid("invalid query param %q: %w", param, err))
		return journal.Nil, false
	}
	return journal.ID(id), true
}

func (r *Request) BoolFromQuery(param string, w *ResponseWriter) (bool, bool) {
	s := r.URL.Query().Get(param)
	if s == "" {
		return false, true
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		w.Error(srverr.ErrInvalid("invalid query param %q: %w", s, err))
		return false, false
	}
	return b, true
}

func (r *Request) Unmarshal(w *ResponseWriter, body interface{}, templates ...interface{}) bool {
	format, err := api.MediaTypeToFormat(r.Header.Get("Content-Type"), DefaultZedFormat)
	if err != nil {
		var uerr *api.ErrUnsupportedMimeType
		if errors.As(err, &uerr) && uerr.Type == "application/x-www-form-urlencoded" {
			// curl will by default set the Accept header to
			// application/x-www-from-urlencoded so assume zson if this is the
			// case.
			format = DefaultZedFormat
		} else {
			w.Error(srverr.ErrInvalid(err))
			return false
		}
	}
	zrc, err := anyio.NewReaderWithOpts(r.Body, zed.NewContext(), anyio.ReaderOpts{Format: format})
	if err != nil {
		w.Error(srverr.ErrInvalid(err))
		return false
	}
	defer zrc.Close()
	zv, err := zrc.Read()
	if err != nil {
		w.Error(srverr.ErrInvalid(err))
		return false
	}
	if zv == nil {
		return true
	}
	m := zson.NewZNGUnmarshaler()
	m.Bind(templates...)
	if err := m.Unmarshal(*zv, body); err != nil {
		w.Error(srverr.ErrInvalid(err))
		return false
	}
	return true
}

type ResponseWriter struct {
	http.ResponseWriter
	Format    string
	Logger    *zap.Logger
	zw        zio.WriteCloser
	marshaler *zson.MarshalZNGContext
	written   int32
}

func (w *ResponseWriter) ContentType() string {
	return w.Header().Get("Content-Type")
}

func (w *ResponseWriter) ZioWriter() zio.WriteCloser {
	if w.zw == nil {
		w.Header().Set("Content-Type", api.FormatToMediaType(w.Format))
		var err error
		w.zw, err = anyio.NewWriter(zio.NopCloser(w), anyio.WriterOpts{Format: w.Format})
		if err != nil {
			w.Error(err)
			return nil
		}
	}
	return w.zw
}
func (w *ResponseWriter) Write(b []byte) (int, error) {
	if atomic.CompareAndSwapInt32(&w.written, 0, 1) {
		w.Header().Set("Content-Type", api.FormatToMediaType(w.Format))
	}
	return w.ResponseWriter.Write(b)
}

func (w *ResponseWriter) Respond(status int, body interface{}) bool {
	w.WriteHeader(status)
	return w.Marshal(body)
}

func (w *ResponseWriter) Error(err error) {
	status, res := errorResponse(err)
	if status >= 500 {
		w.Logger.Warn("Error", zap.Int("status", status), zap.Error(err))
	}
	if atomic.CompareAndSwapInt32(&w.written, 0, 1) {
		// Should errors be returned in different encodings, i.e. adhere to
		// the encoding ?
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			w.Logger.Warn("Error writing response", zap.Error(err))
		}
	}
}

func (w *ResponseWriter) Marshal(body interface{}) bool {
	rec, err := w.marshaler.MarshalRecord(body)
	if err != nil {
		// XXX If status header has not been sent this should send error.
		w.Error(err)
		return false
	}
	zw := w.ZioWriter()
	if zw == nil {
		return false
	}
	if err := zw.Write(rec); err != nil {
		w.Error(err)
		return false
	}
	zw.Close()
	return true
}

func errorResponse(e error) (status int, ae *api.Error) {
	status = http.StatusInternalServerError
	ae = &api.Error{Type: "Error"}

	var pe *parser.Error
	if errors.As(e, &pe) {
		ae.Info = map[string]int{"parse_error_offset": pe.Offset}
	}

	var ze *srverr.Error
	if !errors.As(e, &ze) {
		var kind srverr.Kind
		switch {
		case errors.Is(e, branches.ErrExists) || errors.Is(e, pools.ErrExists):
			kind = srverr.Conflict
		case errors.Is(e, branches.ErrNotFound) || errors.Is(e, pools.ErrNotFound) ||
			errors.Is(e, fs.ErrNotExist):
			kind = srverr.NotFound
		default:
			ae.Message = e.Error()
			return
		}
		ze = &srverr.Error{Kind: kind, Err: e}
	}

	switch ze.Kind {
	case srverr.Invalid:
		status = http.StatusBadRequest
	case srverr.NotFound:
		status = http.StatusNotFound
	case srverr.Exists:
		status = http.StatusBadRequest
	case srverr.Conflict:
		status = http.StatusConflict
	case srverr.NoCredentials:
		status = http.StatusUnauthorized
	case srverr.Forbidden:
		status = http.StatusForbidden
	}

	ae.Kind = ze.Kind.String()
	ae.Message = ze.Message()
	return
}

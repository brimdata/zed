package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"github.com/gorilla/mux"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

type Request struct {
	*http.Request
	Logger *zap.Logger
}

func newRequest(w http.ResponseWriter, r *http.Request, logger *zap.Logger) (*ResponseWriter, *Request) {
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
	accept := r.Header.Get("Accept")
	if accept == "" || accept == "*/*" {
		accept = "application/json"
	}
	res.SetContentType(accept)
	return res, req
}

func (r *Request) PoolID(w *ResponseWriter, root *lake.Root) (ksuid.KSUID, bool) {
	s, ok := r.StringFromPath(w, "pool")
	if !ok {
		return ksuid.Nil, false
	}
	id, err := lakeparse.ParseID(s)
	if err != nil {
		id, err = root.PoolID(r.Context(), s)
		if errors.Is(err, lake.ErrPoolNotFound) {
			w.Error(zqe.ErrNotFound(err))
			return ksuid.Nil, false
		}
		if err != nil {
			w.Error(zqe.ErrInvalid("invalid path param %q: %w", s, err))
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
			w.Error(zqe.ErrInvalid("load endpoint encountered invalid JSON in Zed-Commit header: %w", err))
			return message, false
		}
	}
	return message, true
}

func (r *Request) StringFromPath(w *ResponseWriter, arg string) (string, bool) {
	v := mux.Vars(r.Request)
	s, ok := v[arg]
	if !ok {
		w.Error(zqe.ErrInvalid("no arg %q in path", arg))
		return "", false
	}
	decoded, err := url.QueryUnescape(s)
	return decoded, err == nil
}

func (r *Request) TagFromPath(arg string, w *ResponseWriter) (ksuid.KSUID, bool) {
	v := mux.Vars(r.Request)
	s, ok := v[arg]
	if !ok {
		w.Error(zqe.ErrInvalid("no arg %q in path", arg))
		return ksuid.Nil, false
	}
	id, err := lakeparse.ParseID(s)
	if err != nil {
		w.Error(zqe.ErrInvalid("invalid path param %q: %w", arg, err))
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
		w.Error(zqe.ErrInvalid("invalid query param %q: %w", param, err))
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
		w.Error(zqe.ErrInvalid("invalid query param %q: %w", s, err))
		return false, false
	}
	return b, true
}

func (r *Request) Unmarshal(w *ResponseWriter, body interface{}, templates ...interface{}) bool {
	typ := r.Header.Get("Content-Type")
	if typ == "" || typ == "application/x-www-form-urlencoded" {
		// If Content-Type is unset or is a form (probably set from curl), assume
		// JSON.
		typ = api.MediaTypeJSON
	}
	format, err := api.MediaTypeToFormat(typ)
	if err != nil {
		w.Error(zqe.ErrInvalid(err))
		return false
	}
	zr, err := anyio.NewReaderWithOpts(r.Body, zed.NewContext(), anyio.ReaderOpts{Format: format})
	if err != nil {
		w.Error(zqe.ErrInvalid(err))
		return false
	}
	zv, err := zr.Read()
	if err != nil {
		w.Error(zqe.ErrInvalid(err))
		return false
	}
	if zv == nil {
		return true
	}
	m := zson.NewZNGUnmarshaler()
	m.Bind(templates...)
	if err := m.Unmarshal(*zv, body); err != nil {
		w.Error(zqe.ErrInvalid(err))
		return false
	}
	return true
}

type ResponseWriter struct {
	http.ResponseWriter
	Logger    *zap.Logger
	zw        zio.WriteCloser
	marshaler *zson.MarshalZNGContext
	written   int32
}

func (w *ResponseWriter) ContentType() string {
	return w.Header().Get("Content-Type")
}

func (w *ResponseWriter) SetContentType(ct string) {
	w.Header().Set("Content-Type", ct)
}

func (w *ResponseWriter) ZioWriter() zio.WriteCloser {
	return w.ZioWriterWithOpts(anyio.WriterOpts{})
}

func (w *ResponseWriter) ZioWriterWithOpts(opts anyio.WriterOpts) zio.WriteCloser {
	if w.zw == nil {
		var err error
		if opts.Format == "" {
			opts.Format, err = api.MediaTypeToFormat(w.ContentType())
			if err != nil {
				w.Error(err)
				return nil
			}
		}
		w.zw, err = anyio.NewWriter(zio.NopCloser(w), opts)
		if err != nil {
			w.Error(err)
			return nil
		}
	}
	return w.zw
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
	atomic.StoreInt32(&w.written, 1)
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
		w.SetContentType("application/json")
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

	var ze *zqe.Error
	if !errors.As(e, &ze) {
		ae.Message = e.Error()
		return
	}

	switch ze.Kind {
	case zqe.Invalid:
		status = http.StatusBadRequest
	case zqe.NotFound:
		status = http.StatusNotFound
	case zqe.Exists:
		status = http.StatusBadRequest
	case zqe.Conflict:
		status = http.StatusConflict
	case zqe.NoCredentials:
		status = http.StatusUnauthorized
	case zqe.Forbidden:
		status = http.StatusForbidden
	}

	ae.Kind = ze.Kind.String()
	ae.Message = ze.Message()
	return
}

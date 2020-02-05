package search

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/scanner"
	"github.com/mccanne/zq/zio/detector"
	"github.com/mccanne/zq/zng/resolver"
	"github.com/mccanne/zq/zqd/api"
)

// This mtu is pretty small but it keeps the JSON object size below 64kb or so
// so the recevier can do reasonable, interactive streaming updates.
const defaultMTU = 100

// A Query is the internal representation of search query describing a source
// of tuples, a "search" applied to the tuples producing a set of matched
// tuples, and a proc to the process the tuples
type Query struct {
	Space string
	Dir   int
	Span  nano.Span
	Proc  ast.Proc
}

func parseSearchRequest(r io.Reader) (*Query, error) {
	var b bytes.Buffer
	const limit = 1024 * 1024
	if _, err := b.ReadFrom(io.LimitReader(r, int64(limit))); err != nil {
		return nil, err
	}
	if b.Len() == limit {
		return nil, errors.New("request too big")
	}
	var req api.SearchRequest
	if err := json.Unmarshal(b.Bytes(), &req); err != nil {
		return nil, err
	}
	if req.Span.Ts < 0 {
		return nil, errors.New("time span must have non-negative timestamp")
	}
	if req.Span.Dur < 0 {
		return nil, errors.New("time span must have non-negative duration")
	}
	// XXX allow either direction even through we do forward only right now
	if req.Dir != 1 && req.Dir != -1 {
		return nil, errors.New("time direction must be 1 or -1")
	}
	return UnpackQuery(&req)
}

// UnpackQuery transforms a api.SearchRequest into a Query.
func UnpackQuery(req *api.SearchRequest) (*Query, error) {
	proc, err := ast.UnpackProc(nil, req.Proc)
	if err != nil {
		return nil, err
	}
	return &Query{
		Space: req.Space,
		Dir:   req.Dir,
		Span:  req.Span,
		Proc:  proc,
	}, nil
}

func httpError(w http.ResponseWriter, msg string, code int) {
	b, err := json.Marshal(&api.Error{
		Type:    "Error",
		Message: msg,
	})
	if err != nil {
		b = []byte(err.Error())
	}
	http.Error(w, string(b), code)
}

func Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "bad method", http.StatusBadRequest)
		return
	}
	query, err := parseSearchRequest(r.Body)
	if err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}
	//logger.Debug("parseSearchRequest", zap.Stringer("query", query))
	dataPath := filepath.Join(".", query.Space, "all.bzng") //XXX need root dir param
	f, err := os.Open(dataPath)
	if err != nil {
		httpError(w, "no such space: "+query.Space, http.StatusNotFound)
		return
	}
	defer f.Close()
	zngReader := detector.LookupReader("bzng", f, resolver.NewContext())
	zctx := resolver.NewContext()
	mapper := scanner.NewMapper(zngReader, zctx)
	mux, err := launch(r.Context(), query, mapper, zctx)
	if err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}
	format := r.URL.Query().Get("format")
	switch format {
	default:
		msg := fmt.Sprintf("unsupported output format: %s", format)
		httpError(w, msg, http.StatusBadRequest)
		return
	case "zjson", "json":
		s, err := newJSON(r, w, defaultMTU)
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = run(mux, s, query.Span)
	case "bzng":
		s := newBzngOutput(r, w)
		err = run(mux, s, query.Span)
	}
	if err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
	}
}

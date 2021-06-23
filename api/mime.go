package api

import (
	"errors"
	"fmt"
	"mime"
)

const (
	MediaTypeAny    = "*/*"
	MediaTypeCSV    = "text/csv"
	MediaTypeJSON   = "application/json"
	MediaTypeNDJSON = "application/x-ndjson"
	MediaTypeZJSON  = "application/x-zjson"
	MediaTypeZNG    = "application/x-zng"
	MediaTypeZSON   = "application/x-zson"
)

var ErrMediaTypeUnspecified = errors.New("media type unspecified")

func IsAmbiguousMediaType(s string) bool {
	typ, _, err := mime.ParseMediaType(s)
	if err != nil && !errors.Is(err, mime.ErrInvalidMediaParameter) {
		return false
	}
	return typ == "" || typ == "*/*"
}

func MediaTypeToFormat(s string) (string, error) {
	typ, _, err := mime.ParseMediaType(s)
	if err != nil && !errors.Is(err, mime.ErrInvalidMediaParameter) {
		return "", err
	}
	switch typ {
	case MediaTypeAny, "":
		return "", ErrMediaTypeUnspecified
	case MediaTypeCSV:
		return "csv", nil
	case MediaTypeJSON:
		return "json", nil
	case MediaTypeNDJSON:
		return "ndjson", nil
	case MediaTypeZJSON:
		return "zjson", nil
	case MediaTypeZNG:
		return "zng", nil
	case MediaTypeZSON:
		return "zson", nil
	}
	return "", fmt.Errorf("unknown MIME type: %s", typ)
}

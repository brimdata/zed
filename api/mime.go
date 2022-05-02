package api

import (
	"errors"
	"fmt"
	"mime"
	"strings"
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

type ErrUnsupportedMimeType struct {
	Type string
}

func (m *ErrUnsupportedMimeType) Error() string {
	return fmt.Sprintf("unsupported MIME type: %s", m.Type)
}

// MediaTypeToFormat returns the anyio format of the media type value s. If s
// is MediaTypeAny or undefined the default format dflt will be returned.
func MediaTypeToFormat(s string, dflt string) (string, error) {
	if s = strings.TrimSpace(s); s == "" {
		return dflt, nil
	}
	typ, _, err := mime.ParseMediaType(s)
	if err != nil && !errors.Is(err, mime.ErrInvalidMediaParameter) {
		return "", err
	}
	switch typ {
	case MediaTypeAny, "":
		return dflt, nil
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
	return "", &ErrUnsupportedMimeType{typ}
}

func FormatToMediaType(format string) string {
	switch format {
	case "csv":
		return MediaTypeCSV
	case "json":
		return MediaTypeJSON
	case "ndjson":
		return MediaTypeNDJSON
	case "zjson":
		return MediaTypeZJSON
	case "zng":
		return MediaTypeZNG
	case "zson":
		return MediaTypeZSON
	default:
		return ""
	}
}

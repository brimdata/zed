package api

import (
	"errors"
	"fmt"
	"mime"
	"strings"
)

const (
	MediaTypeAny         = "*/*"
	MediaTypeArrowStream = "application/vnd.apache.arrow.stream"
	MediaTypeCSV         = "text/csv"
	MediaTypeJSON        = "application/json"
	MediaTypeLine        = "application/x-line"
	MediaTypeNDJSON      = "application/x-ndjson"
	MediaTypeParquet     = "application/x-parquet"
	MediaTypeZeek        = "application/x-zeek"
	MediaTypeZJSON       = "application/x-zjson"
	MediaTypeZNG         = "application/x-zng"
	MediaTypeZSON        = "application/x-zson"
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
	case MediaTypeArrowStream:
		return "arrows", nil
	case MediaTypeCSV:
		return "csv", nil
	case MediaTypeJSON:
		return "json", nil
	case MediaTypeLine:
		return "line", nil
	case MediaTypeNDJSON:
		return "ndjson", nil
	case MediaTypeParquet:
		return "parquet", nil
	case MediaTypeZeek:
		return "zeek", nil
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
	case "arrows":
		return MediaTypeArrowStream
	case "csv":
		return MediaTypeCSV
	case "json":
		return MediaTypeJSON
	case "line":
		return MediaTypeLine
	case "ndjson":
		return MediaTypeNDJSON
	case "parquet":
		return MediaTypeParquet
	case "zeek":
		return MediaTypeZeek
	case "zjson":
		return MediaTypeZJSON
	case "zng":
		return MediaTypeZNG
	case "zson":
		return MediaTypeZSON
	default:
		panic(fmt.Sprintf("unknown format type: %s", format))
	}
}

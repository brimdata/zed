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
	MediaTypeTSV         = "application/tab-separated-values"
	MediaTypeVNG         = "application/x-vng"
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
	case MediaTypeTSV:
		return "tsv", nil
	case MediaTypeVNG:
		return "vng", nil
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

func FormatToMediaType(format string) (string, error) {
	switch format {
	case "arrows":
		return MediaTypeArrowStream, nil
	case "csv":
		return MediaTypeCSV, nil
	case "json":
		return MediaTypeJSON, nil
	case "line":
		return MediaTypeLine, nil
	case "ndjson":
		return MediaTypeNDJSON, nil
	case "parquet":
		return MediaTypeParquet, nil
	case "tsv":
		return MediaTypeTSV, nil
	case "vng":
		return MediaTypeVNG, nil
	case "zeek":
		return MediaTypeZeek, nil
	case "zjson":
		return MediaTypeZJSON, nil
	case "zng":
		return MediaTypeZNG, nil
	case "zson":
		return MediaTypeZSON, nil
	default:
		return "", fmt.Errorf("unknown format type: %s", format)
	}
}

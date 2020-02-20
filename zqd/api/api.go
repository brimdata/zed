package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zio/zjsonio"
	"github.com/gorilla/mux"
)

// ExtractSpace returns the unescaped space from the path of a request.
// XXX is this in the right package?
func ExtractSpace(r *http.Request) (string, error) {
	v := mux.Vars(r)
	space, ok := v["space"]
	if !ok {
		return "", errors.New("no space found")
	}
	return url.PathUnescape(space)
}

type Error struct {
	Type    string      `json:"type"`
	Message string      `json:"error"`
	Info    interface{} `json:"info,omitempty"`
}

type TaskStart struct {
	Type   string `json:"type"`
	TaskID int64  `json:"task_id"`
}

type TaskEnd struct {
	Type   string `json:"type"`
	TaskID int64  `json:"task_id"`
	Error  *Error `json:"error,omitempty"`
}

type SearchRequest struct {
	Space string          `json:"space" validate:"required"`
	Proc  json.RawMessage `json:"proc" validate:"required"`
	Span  nano.Span       `json:"span"`
	Dir   int             `json:"dir" validate:"required"`
}

type SearchRecords struct {
	Type      string           `json:"type"`
	ChannelID int              `json:"channel_id"`
	Records   []zjsonio.Record `json:"records"`
}

type SearchWarnings struct {
	Type     string   `json:"type"`
	Warnings []string `json:"warnings"`
}

type SearchEnd struct {
	Type      string `json:"type"`
	ChannelID int    `json:"channel_id"`
	Reason    string `json:"reason"`
}

type SearchStats struct {
	Type       string  `json:"type"`
	StartTime  nano.Ts `json:"start_time"`
	UpdateTime nano.Ts `json:"update_time"`
	ScannerStats
}

type ScannerStats struct {
	CurrentTs       nano.Ts `json:"current_ts"`
	BytesRead       int64   `json:"bytes_read"`
	BytesMatched    int64   `json:"bytes_matched"`
	RecordsRead     int64   `json:"records_read"`
	RecordsMatched  int64   `json:"records_matched"`
	RecordsReceived int64   `json:"records_received"`
}

type SpaceInfo struct {
	Name          string   `json:"name"`
	MinTime       *nano.Ts `json:"min_time,omitempty"`
	MaxTime       *nano.Ts `json:"max_time,omitempty"`
	Size          int64    `json:"size" unit:"bytes"`
	PacketSupport bool     `json:"packet_support"`
	PacketSize    int64    `json:"packet_size" unit:"bytes"`
}

type StatusResponse struct {
	Ok      bool   `json:"ok"`
	Version string `json:"version"`
}

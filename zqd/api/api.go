package api

import (
	"encoding/json"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zio/zjsonio"
)

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
	Name    string   `json:"name"`
	MinTime *nano.Ts `json:"min_time,omitempty"`
	MaxTime *nano.Ts `json:"max_time,omitempty"`
	Size    int64    `json:"size" unit:"bytes"`
}

type StatusResponse struct {
	Ok      bool   `json:"ok"`
	Version string `json:"version"`
}

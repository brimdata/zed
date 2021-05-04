package api

import (
	"context"
	"encoding/json"

	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zjsonio"
	"github.com/segmentio/ksuid"
)

const RequestIDHeader = "X-Request-ID"

func RequestIDFromContext(ctx context.Context) string {
	if v := ctx.Value(RequestIDHeader); v != nil {
		return v.(string)
	}
	return ""
}

type Error struct {
	Type    string      `json:"type"`
	Kind    string      `json:"kind"`
	Message string      `json:"error"`
	Info    interface{} `json:"info,omitempty"`
}

func (e Error) Error() string {
	return e.Message
}

type ASTRequest struct {
	ZQL string `json:"zql"`
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
	JournalID uint64          `json:"journal_id"`
	Pool      ksuid.KSUID     `json:"pool"`
	Proc      json.RawMessage `json:"proc,omitempty"`
	Span      nano.Span       `json:"span"`
	Dir       int             `json:"dir"`
}

type SearchRecords struct {
	Type      string           `json:"type"`
	ChannelID int              `json:"channel_id"`
	Records   []zjsonio.Object `json:"records"`
}

type SearchWarning struct {
	Type    string `json:"type"`
	Warning string `json:"warning"`
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
	BytesRead      int64 `json:"bytes_read"`
	BytesMatched   int64 `json:"bytes_matched"`
	RecordsRead    int64 `json:"records_read"`
	RecordsMatched int64 `json:"records_matched"`
}

type Pool struct {
	ID   ksuid.KSUID `json:"id" zng:"id"`
	Name string      `json:"name" zng:"name"`
}

type PoolInfo struct {
	Pool
	Span *nano.Span `json:"span,omitempty"`
	Size int64      `json:"size" unit:"bytes"`
}

type VersionResponse struct {
	Version string `json:"version"`
}

type PoolPostRequest struct {
	Name   string         `json:"name"`
	Keys   []field.Static `json:"keys"`
	Order  zbuf.Order     `json:"order"`
	Thresh int64          `json:"thresh"`
}

type PoolPutRequest struct {
	Name string `json:"name"`
}

type CommitRequest struct {
	Commit  string `json:"commit"`
	User    string `json:"user"`
	Message string `json:"message"`
}

type LogPostRequest struct {
	Paths   []string        `json:"paths"`
	StopErr bool            `json:"stop_err"`
	Shaper  json.RawMessage `json:"shaper,omitempty"`
}

type LogPostWarning struct {
	Type    string `json:"type"`
	Warning string `json:"warning"`
}

type LogPostStatus struct {
	Type         string `json:"type"`
	LogTotalSize int64  `json:"log_total_size" unit:"bytes"`
	LogReadSize  int64  `json:"log_read_size" unit:"bytes"`
}

type LogPostResponse struct {
	Type      string      `json:"type"`
	BytesRead int64       `json:"bytes_read" unit:"bytes"`
	Commit    ksuid.KSUID `json:"commit"`
	Warnings  []string    `json:"warnings"`
}

type IndexSearchRequest struct {
	IndexName string   `json:"index_name"`
	Patterns  []string `json:"patterns"`
}

type IndexPostRequest struct {
	Keys     []string `json:"keys"`
	Name     string   `json:"name"`
	Patterns []string `json:"patterns"`
	Zed      string   `json:"zed,omitempty"`
}

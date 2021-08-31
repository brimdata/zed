package api

import (
	"context"
	"encoding/json"

	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
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
	JournalID uint64 `json:"journal_id"`
	// Pool has to have wrapped type because KSUID in request body can be either
	// hex or base62 and the regular ksuid.KSUID does not handle hex. This will
	// go away once the search endpoint is deprecated.
	Pool KSUID           `json:"pool"`
	Proc json.RawMessage `json:"proc,omitempty"`
	Span nano.Span       `json:"span"`
	Dir  int             `json:"dir"`
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

type VersionResponse struct {
	Version string `json:"version"`
}

type PoolPostRequest struct {
	Name   string       `json:"name"`
	Layout order.Layout `json:"layout"`
	Thresh int64        `json:"thresh"`
}

type PoolPutRequest struct {
	Name string `json:"name"`
}

type BranchPostRequest struct {
	Name   string `json:"name"`
	Commit string `json:"commit"`
}

type BranchMergeRequest struct {
	At string `json:"at"`
}

type CommitMessage struct {
	Author  string `zng:"author"`
	Message string `zng:"message"`
}

type CommitResponse struct {
	Commit   ksuid.KSUID `zng:"commit"`
	Warnings []string    `zng:"warnings"`
}

type IndexPostRequest struct {
	Keys     []string `json:"keys"`
	Name     string   `json:"name"`
	Patterns []string `json:"patterns"`
	Zed      string   `json:"zed,omitempty"`
}

type EventBranchCommit struct {
	CommitID string `json:"commit_id"`
	PoolID   string `json:"pool_id"`
	Branch   string `json:"branch"`
	Parent   string `json:"parent"`
}

type EventPool struct {
	PoolID string `json:"pool_id"`
}

type EventBranch struct {
	PoolID string `json:"pool_id"`
	Branch string `json:"branch"`
}

type QueryRequest struct {
	Query string `json:"query"`
}

type QueryChannelSet struct {
	ChannelID int `json:"channel_id" zng:"channel_id"`
}

type QueryChannelEnd struct {
	ChannelID int `json:"channel_id" zng:"channel_id"`
}

type QueryError struct {
	Error string `json:"error" zng:"error"`
}

type QueryStats struct {
	StartTime  nano.Ts `json:"start_time" zng:"start_time"`
	UpdateTime nano.Ts `json:"update_time" zng:"update_time"`
	ScannerStats
}

type QueryWarning struct {
	Warning string `json:"warning" zng:"warning"`
}

package api

import (
	"context"

	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
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
	Author string `zed:"author"`
	Body   string `zed:"body"`
}

type CommitResponse struct {
	Commit   ksuid.KSUID `zed:"commit"`
	Warnings []string    `zed:"warnings"`
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
	Query string              `json:"query"`
	Head  lakeparse.Commitish `json:"head"`
}

type QueryChannelSet struct {
	ChannelID int `json:"channel_id" zed:"channel_id"`
}

type QueryChannelEnd struct {
	ChannelID int `json:"channel_id" zed:"channel_id"`
}

type QueryError struct {
	Error string `json:"error" zed:"error"`
}

type QueryStats struct {
	StartTime  nano.Ts `json:"start_time" zed:"start_time"`
	UpdateTime nano.Ts `json:"update_time" zed:"update_time"`
	ScannerStats
}

type QueryWarning struct {
	Warning string `json:"warning" zed:"warning"`
}

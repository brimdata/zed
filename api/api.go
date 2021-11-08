package api

import (
	"context"

	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
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

type DeleteRequest struct {
	ObjectIDs []ksuid.KSUID `zed:"object_ids"`
}

type CommitMessage struct {
	Author string `zed:"author"`
	Body   string `zed:"body"`
	Meta   string `zed:"meta"`
}

type CommitResponse struct {
	Commit   ksuid.KSUID `zed:"commit"`
	Warnings []string    `zed:"warnings"`
}

type IndexRulesAddRequest struct {
	Rules []index.Rule `zed:"rules"`
}

type IndexRulesDeleteRequest struct {
	RuleIDs []string `zed:"rule_ids"`
}

type IndexRulesDeleteResponse struct {
	Rules []index.Rule `zed:"rules"`
}

type IndexApplyRequest struct {
	RuleName string        `zed:"rule_name"`
	Tags     []ksuid.KSUID `zed:"tags"`
}

type IndexUpdateRequest struct {
	RuleNames []string `zed:"rule_names"`
}

type EventBranchCommit struct {
	CommitID ksuid.KSUID `json:"commit_id"`
	PoolID   ksuid.KSUID `json:"pool_id"`
	Branch   string      `json:"branch"`
	Parent   string      `json:"parent"`
}

type EventPool struct {
	PoolID ksuid.KSUID `zed:"pool_id"`
}

type EventBranch struct {
	PoolID ksuid.KSUID `zed:"pool_id"`
	Branch string      `zed:"branch"`
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
	zbuf.ScannerStats
}

type QueryWarning struct {
	Warning string `json:"warning" zed:"warning"`
}

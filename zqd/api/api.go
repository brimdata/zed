package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zio/zjsonio"
)

type Error struct {
	Type    string      `json:"type"`
	Message string      `json:"error"`
	Info    interface{} `json:"info,omitempty"`
}

// Error implements the error interface so this struct can be passed around
// as an error.  The error string is the JSON encoding of the Error struct.
// with indentation.
func (e Error) Error() string {
	b, err := json.MarshalIndent(e, "", "\t")
	if err != nil {
		// this shouldn't happen
		return e.Message
	}
	return string(b)
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
	PacketPath    string   `json:"packet_path"`
}

type StatusResponse struct {
	Ok      bool   `json:"ok"`
	Version string `json:"version"`
}

type SpacePostRequest struct {
	Name    string `json:"name"`
	DataDir string `json:"data_dir"`
}

type SpacePostResponse SpacePostRequest

type PacketPostRequest struct {
	Path string `json:"path"`
}

type PacketPostStatus struct {
	Type           string   `json:"type"`
	StartTime      nano.Ts  `json:"start_time"`
	UpdateTime     nano.Ts  `json:"update_time"`
	PacketSize     int64    `json:"packet_total_size" unit:"bytes"`
	PacketReadSize int64    `json:"packet_read_size" unit:"bytes"`
	SnapshotCount  int      `json:"snapshot_count"`
	MinTime        *nano.Ts `json:"min_time,omitempty"`
	MaxTime        *nano.Ts `json:"max_time,omitempty"`
}

// PacketSearch are the query string args to the packet endpoint when searching
// for packets within a connection 5-tuple.
type PacketSearch struct {
	Span    nano.Span
	Proto   string `validate:"required"`
	SrcHost net.IP `validate:"required"`
	SrcPort uint16
	DstHost net.IP `validate:"required"`
	DstPort uint16
}

// ToQuery transforms a packet search into a url.Values.
func (ps *PacketSearch) ToQuery() url.Values {
	tssec, tsns := ps.Span.Ts.Split()
	dursec := int(ps.Span.Dur / 1000000000)
	durns := int(int64(ps.Span.Dur) - int64(dursec)*1000000000)
	q := url.Values{}
	q.Add("ts_sec", strconv.Itoa(int(tssec)))
	q.Add("ts_ns", strconv.Itoa(int(tsns)))
	q.Add("duration_sec", strconv.Itoa(dursec))
	q.Add("duration_ns", strconv.Itoa(durns))
	q.Add("proto", ps.Proto)
	q.Add("src_host", ps.SrcHost.String())
	q.Add("dst_host", ps.DstHost.String())
	if ps.SrcPort != 0 {
		q.Add("src_port", strconv.Itoa(int(ps.SrcPort)))
	}
	if ps.DstPort != 0 {
		q.Add("dst_port", strconv.Itoa(int(ps.DstPort)))
	}

	return q
}

// FromQuery parses a query string and populates the receiver's values.
func (ps *PacketSearch) FromQuery(v url.Values) error {
	var err error
	var tsSec, tsNs, durSec, durNs int64
	if tsSec, err = strconv.ParseInt(v.Get("ts_sec"), 10, 64); err != nil {
		return err
	}
	if tsNs, err = strconv.ParseInt(v.Get("ts_ns"), 10, 64); err != nil {
		return err
	}
	if durSec, err = strconv.ParseInt(v.Get("duration_sec"), 10, 64); err != nil {
		return err
	}
	if durNs, err = strconv.ParseInt(v.Get("duration_ns"), 10, 64); err != nil {
		return err
	}
	if v.Get("src_port") != "" {
		p, err := strconv.ParseUint(v.Get("src_port"), 10, 16)
		if err != nil {
			return err
		}
		ps.SrcPort = uint16(p)
	}
	if v.Get("dst_port") != "" {
		p, err := strconv.ParseUint(v.Get("dst_port"), 10, 16)
		if err != nil {
			return err
		}
		ps.DstPort = uint16(p)
	}
	span := nano.Span{
		Ts:  nano.Unix(tsSec, tsNs),
		Dur: nano.Duration(durSec, durNs),
	}
	ps.Span = span
	ps.Proto = v.Get("proto")
	if ps.SrcHost = net.ParseIP(v.Get("src_host")); ps.SrcHost == nil {
		return fmt.Errorf("invalid ip: %s", ps.SrcHost)
	}
	if ps.DstHost = net.ParseIP(v.Get("dst_host")); ps.DstHost == nil {
		return fmt.Errorf("invalid ip: %s", ps.DstHost)
	}
	return nil
}

package client

import (
	"encoding/json"
	"fmt"

	"github.com/brimdata/zq/api"
)

// unpack transforms a piped json stream into the appropriate api response
// and returns it as an empty interface so that the caller can receive
// a stream of objects, check their types, and process them accordingly.
func unpack(b []byte) (interface{}, error) {
	var v struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	var out interface{}
	switch v.Type {
	case "TaskStart":
		out = &api.TaskStart{}
	case "TaskEnd":
		out = &api.TaskEnd{}
	case "SearchRecords":
		out = &api.SearchRecords{}
	case "SearchWarning":
		out = &api.SearchWarning{}
	case "SearchStats":
		out = &api.SearchStats{}
	case "SearchEnd":
		out = &api.SearchEnd{}
	case "PcapPostStatus":
		out = &api.PcapPostStatus{}
	case "PcapPostWarning":
		out = &api.PcapPostWarning{}
	case "LogPostStatus":
		out = &api.LogPostStatus{}
	case "LogPostWarning":
		out = &api.LogPostWarning{}
	case "":
		return nil, fmt.Errorf("no type field in search result: %s", string(b))
	default:
		return nil, fmt.Errorf("unknown type in results stream: %s", v.Type)
	}
	if err := json.Unmarshal(b, out); err != nil {
		return nil, err
	}
	return out, nil
}

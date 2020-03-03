package api

import (
	"encoding/json"
	"fmt"
)

// unpack transforms a search result into a v2.SearchResult, v2.SearchStats, or v2.SearchEnd
// and returns it as an empty interface so that the caller can receive
// a stream of objects, check their types, and process them accordingly
func unpack(b []byte) (interface{}, error) {
	var v interface{}
	err := json.Unmarshal(b, &v)
	if err != nil {
		return nil, err
	}
	object, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("bad json object: %s", string(b))
	}
	which, ok := object["type"]
	if !ok {
		return nil, fmt.Errorf("no type field in search result: %s", string(b))
	}
	var out interface{}
	switch which {
	case "TaskStart":
		out = &TaskStart{}
	case "TaskEnd":
		out = &TaskEnd{}
	case "SearchRecords":
		out = &SearchRecords{}
	case "SearchWarnings":
		out = &SearchWarnings{}
	case "SearchStats":
		out = &SearchStats{}
	case "SearchEnd":
		out = &SearchEnd{}
	case "PacketPostStatus":
		out = &PacketPostStatus{}
	default:
		return nil, fmt.Errorf("unknown type in results stream: %s", which)
	}
	err = json.Unmarshal(b, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

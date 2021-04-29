package search

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zjsonio"
)

const MaxJSONRecords = 25000

// JSON implements the Output interface.
type JSON struct {
	stream   *zjsonio.Stream
	response http.ResponseWriter
	mtu      int
	ctrl     bool
	all      []interface{}
	stat     interface{}
}

func NewJSONOutput(resp http.ResponseWriter, mtu int, ctrl bool) *JSON {
	return &JSON{
		stream:   zjsonio.NewStream(),
		response: resp,
		mtu:      mtu,
		ctrl:     ctrl,
	}
}

func (j *JSON) SendBatch(cid int, set zbuf.Batch) error {
	records := set.Records()
	n := len(records)
	for n > 0 {
		frag := n
		if frag > j.mtu {
			frag = j.mtu
		}
		formatted, err := formatRecords(j.stream, records[0:frag])
		if err != nil {
			return err
		}
		v := &api.SearchRecords{
			Type:      "SearchRecords",
			ChannelID: cid,
			Records:   formatted,
		}
		if err := j.append(v); err != nil {
			return err
		}
		records = records[frag:]
		n -= frag
	}
	set.Unref()
	return nil
}

func (j *JSON) append(msg interface{}) error {
	j.all = append(j.all, msg)
	if len(j.all) > MaxJSONRecords {
		err := errors.New("memory limit exceeded for single JSON response")
		http.Error(j.response, err.Error(), http.StatusBadRequest)
		return err
	}
	return nil
}

func (v *JSON) SendControl(msg interface{}) error {
	if !v.ctrl {
		return nil
	}
	if _, ok := msg.(*api.SearchStats); ok {
		v.stat = msg
		return nil
	}
	return v.append(msg)
}

func (j *JSON) End(msg interface{}) error {
	if !j.ctrl {
		msg = nil
	}
	err := json.NewEncoder(j.response).Encode(j.all)
	if err != nil {
		http.Error(j.response, err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

func (s *JSON) ContentType() string {
	return MimeTypeJSON
}

package client

import (
	"fmt"
	"io"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/zson"
)

type EventsClient struct {
	rc          io.ReadCloser
	unmarshaler *zson.UnmarshalContext
}

func newEventsClient(resp *Response) *EventsClient {
	unmarshaler := zson.NewUnmarshaler()
	unmarshaler.Bind(
		api.EventPool{},
		api.EventBranch{},
		api.EventBranchCommit{},
	)
	return &EventsClient{
		rc:          resp.Body,
		unmarshaler: unmarshaler,
	}
}

func (l *EventsClient) Recv() (string, interface{}, error) {
	var kind, data string
	_, err := fmt.Fscanf(l.rc, "event: %s\ndata: %s\n\n\n", &kind, &data)
	if err != nil {
		return "", nil, err
	}
	var v interface{}
	if err := l.unmarshaler.Unmarshal(data, &v); err != nil {
		return "", nil, err
	}
	return kind, v, err
}

func (l *EventsClient) Close() error {
	return l.rc.Close()
}

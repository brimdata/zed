package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strconv"

	"net/http"
	"net/url"
	"time"

	"github.com/brimsec/zq/pcap/pcapio"
	"github.com/go-resty/resty/v2"
)

const (
	// DefaultPort zqd port to connect with.
	DefaultPort      = 9867
	DefaultUserAgent = "zqd-client-golang"
)

var (
	// ErrSpaceNotFound returns when specified space does not exist.
	ErrSpaceNotFound = errors.New("space not found")
	// ErrSpaceExists returns when specified the space already exists.
	ErrSpaceExists = errors.New("space exists")
	// ErrNoPcapResultsFound returns when a pcap search yields no results
	ErrNoPcapResultsFound = errors.New("no pcap results found for search")
)

type Connection struct {
	client *resty.Client
}

func newConnection(client *resty.Client) *Connection {
	client.SetError(Error{})
	client.OnAfterResponse(checkError)
	c := &Connection{client: client}
	c.SetUserAgent(DefaultUserAgent)
	return c
}

// NewConnection creates a new connection with the given useragent string
// and a base URL set up to talk to http://localhost:defaultport
func NewConnection() *Connection {
	u := "http://localhost:" + strconv.Itoa(DefaultPort)
	return NewConnectionTo(u)
}

// NewConnectionTo creates a new connection with the given useragent string
// and a base URL derived from the hostURL argument.
func NewConnectionTo(hostURL string) *Connection {
	client := resty.New()
	client.HostURL = hostURL
	return newConnection(client)
}

func (c *Connection) SetUserAgent(useragent string) {
	c.client.SetHeader("User-Agent", useragent)
}

func (c *Connection) Do(ctx context.Context, method, url string, body interface{}) (*resty.Response, error) {
	req := c.Request(ctx).SetBody(body)
	return req.Execute(method, url)
}

func checkError(client *resty.Client, resp *resty.Response) error {
	if resp.IsSuccess() {
		return nil
	}
	resErr := &ErrorResponse{Response: resp}
	if err := resp.Error(); err != nil {
		resErr.Err = err.(*Error)
	} else {
		resErr.Err = errors.New(resp.String())
	}
	return resErr
}

func (c *Connection) stream(req *resty.Request) (io.ReadCloser, error) {
	resp, err := req.SetDoNotParseResponse(true).Send() // disables middleware
	if err != nil {
		return nil, err
	}
	r := resp.RawBody()
	if resp.IsSuccess() {
		return r, nil
	}
	defer r.Close()
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	resErr := &ErrorResponse{Response: resp}
	if resty.IsJSONType(resp.Header().Get("Content-Type")) {
		var apierr Error
		if err := json.Unmarshal(body, &apierr); err != nil {
			return nil, err
		}
		resErr.Err = &apierr
	} else {
		resErr.Err = errors.New(string(body))
	}
	return nil, resErr
}

// SetTimeout sets the underlying http request timeout to the given duration
func (c *Connection) SetTimeout(to time.Duration) {
	c.client.SetTimeout(to)
}

func (c *Connection) URL() string {
	return c.client.HostURL
}

func (c *Connection) SetURL(u string) {
	c.client.SetHostURL(u)
}

func (c *Connection) Request(ctx context.Context) *resty.Request {
	return c.client.R().SetContext(ctx)
}

// Ping checks to see if the server and measure the time it takes to
// get back the response.
func (c *Connection) Ping(ctx context.Context) (time.Duration, error) {
	resp, err := c.Request(ctx).
		Get("/status")
	if err != nil {
		return 0, err
	}
	return resp.Time(), nil
}

// SpaceInfo retrieves information about the specified space.
func (c *Connection) SpaceInfo(ctx context.Context, id SpaceID) (*SpaceInfo, error) {
	path := path.Join("/space", url.PathEscape(string(id)))
	resp, err := c.Request(ctx).
		SetResult(&SpaceInfo{}).
		Get(path)
	if err != nil {
		if r, ok := err.(*ErrorResponse); ok && r.StatusCode() == http.StatusNotFound {
			return nil, ErrSpaceNotFound
		}
		return nil, err
	}
	return resp.Result().(*SpaceInfo), nil
}

func (c *Connection) SpacePost(ctx context.Context, req SpacePostRequest) (*SpaceInfo, error) {
	resp, err := c.Request(ctx).
		SetBody(req).
		SetResult(&SpaceInfo{}).
		Post("/space")
	if err != nil {
		if r, ok := err.(*ErrorResponse); ok && r.StatusCode() == http.StatusConflict {
			return nil, ErrSpaceExists
		}
		return nil, err
	}
	return resp.Result().(*SpaceInfo), nil
}

func (c *Connection) SubspacePost(ctx context.Context, id SpaceID, req SubspacePostRequest) (*SpaceInfo, error) {
	resp, err := c.Request(ctx).
		SetBody(req).
		SetResult(&SpaceInfo{}).
		Post(path.Join("/space", string(id), "subspace"))
	if err != nil {
		if r, ok := err.(*ErrorResponse); ok && r.StatusCode() == http.StatusConflict {
			return nil, ErrSpaceExists
		}
		return nil, err
	}
	return resp.Result().(*SpaceInfo), nil
}

func (c *Connection) SpacePut(ctx context.Context, id SpaceID, req SpacePutRequest) error {
	_, err := c.Request(ctx).
		SetBody(req).
		Put(path.Join("/space", string(id)))
	return err
}

func (c *Connection) SpaceList(ctx context.Context) ([]SpaceInfo, error) {
	var res []SpaceInfo
	_, err := c.Request(ctx).
		SetResult(&res).
		Get("/space")
	return res, err
}

func (c *Connection) SpaceDelete(ctx context.Context, id SpaceID) (err error) {
	path := path.Join("/space", url.PathEscape(string(id)))
	_, err = c.Request(ctx).Delete(path)
	if r, ok := err.(*ErrorResponse); ok && r.StatusCode() == http.StatusNotFound {
		return ErrSpaceNotFound
	}
	return err
}

// Search sends a search task to the server and returns a Search interface
// that the caller uses to stream back results via the Read method.
func (c *Connection) Search(ctx context.Context, search SearchRequest, params map[string]string) (Search, error) {
	req := c.Request(ctx).
		SetBody(search).
		SetQueryParam("format", "zng")
	req.SetQueryParams(params)
	req.Method = http.MethodPost
	req.URL = "/search"
	r, err := c.stream(req)
	if err != nil {
		return nil, err
	}
	return NewZngSearch(r), nil
}

func (c *Connection) IndexSearch(ctx context.Context, space SpaceID, search IndexSearchRequest, params map[string]string) (Search, error) {
	req := c.Request(ctx).
		SetBody(search).
		SetQueryParam("format", "zng")
	req.SetQueryParams(params)
	req.Method = http.MethodPost
	req.URL = path.Join("/space", string(space), "indexsearch")
	r, err := c.stream(req)
	if err != nil {
		return nil, err
	}
	return NewZngSearch(r), nil
}

func (c *Connection) PcapPost(ctx context.Context, space SpaceID, payload PcapPostRequest) (*Stream, error) {
	req := c.Request(ctx).
		SetBody(payload)
	req.Method = http.MethodPost
	req.URL = path.Join("/space", string(space), "pcap")
	r, err := c.stream(req)
	if err != nil {
		return nil, err
	}
	jsonpipe := NewJSONPipeScanner(r)
	return NewStream(jsonpipe), nil
}

func (c *Connection) PcapSearch(ctx context.Context, space SpaceID, payload PcapSearch) (*PcapReadCloser, error) {
	req := c.Request(ctx).
		SetQueryParamsFromValues(payload.ToQuery())
	req.Method = http.MethodGet
	req.URL = path.Join("/space", string(space), "pcap")
	r, err := c.stream(req)
	if err != nil {
		if r, ok := err.(*ErrorResponse); ok && r.StatusCode() == http.StatusNotFound {
			return nil, ErrNoPcapResultsFound
		}
		return nil, err
	}
	pr, err := pcapio.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &PcapReadCloser{pr, r}, nil
}

type PcapReadCloser struct {
	pcapio.Reader
	io.Closer
}

func (c *Connection) LogPost(ctx context.Context, space SpaceID, payload LogPostRequest) (*Stream, error) {
	req := c.Request(ctx).
		SetBody(payload)
	req.Method = http.MethodPost
	req.URL = path.Join("/space", url.PathEscape(string(space)), "log")
	r, err := c.stream(req)
	if err != nil {
		return nil, err
	}
	jsonpipe := NewJSONPipeScanner(r)
	return NewStream(jsonpipe), nil
}

type ErrorResponse struct {
	*resty.Response
	Err error
}

func (e *ErrorResponse) Unwrap() error {
	return e.Err
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("status code %d: %v", e.StatusCode(), e.Err)
}

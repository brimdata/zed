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
		var apierr *Error
		if err := json.Unmarshal(body, apierr); err != nil {
			return nil, err
		}
		resErr.Err = apierr
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
func (c *Connection) SpaceInfo(ctx context.Context, spaceName string) (*SpaceInfo, error) {
	path := path.Join("/space", url.PathEscape(spaceName))
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

func (c *Connection) SpacePost(ctx context.Context, req SpacePostRequest) (*SpacePostResponse, error) {
	resp, err := c.Request(ctx).
		SetBody(req).
		SetResult(&SpacePostResponse{}).
		Post("/space")
	if err != nil {
		if r, ok := err.(*ErrorResponse); ok && r.StatusCode() == http.StatusConflict {
			return nil, ErrSpaceExists
		}
		return nil, err
	}
	return resp.Result().(*SpacePostResponse), nil
}

func (c *Connection) SpaceList(ctx context.Context) ([]string, error) {
	var res []string
	_, err := c.Request(ctx).
		SetResult(&res).
		Get("/space")
	return res, err
}

func (c *Connection) SpaceDelete(ctx context.Context, spaceName string) (err error) {
	path := path.Join("/space", url.PathEscape(spaceName))
	_, err = c.Request(ctx).Delete(path)
	return err
}

// Search sends a search task to the server and returns a Search interface
// that the caller uses to stream back results via the Read method.
func (c *Connection) Search(ctx context.Context, search SearchRequest) (Search, error) {
	req := c.Request(ctx).
		SetBody(search).
		SetQueryParam("format", "bzng")
	req.Method = http.MethodPost
	req.URL = "/search"
	r, err := c.stream(req)
	if err != nil {
		return nil, err
	}
	return NewBzngSearch(r), nil
}

func (c *Connection) PostPacket(ctx context.Context, space string, payload PacketPostRequest) (*Stream, error) {
	req := c.Request(ctx).
		SetBody(payload).
		SetHeader("format", "bzng")
	req.Method = http.MethodPost
	req.URL = path.Join("/space", url.PathEscape(space), "packet")
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

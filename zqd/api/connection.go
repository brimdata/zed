package api

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"path"
	"strconv"
	"strings"

	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/brimsec/zq/pkg/catcher"
)

const (
	// DefaultPort zqd port to connect with.
	DefaultPort = 9867
)

var (
	// ErrSpaceNotFound returns when specified space does not exist.
	ErrSpaceNotFound = errors.New("space not found")
	// ErrSpaceExists returns when specified the space already exists.
	ErrSpaceExists = errors.New("space exists")
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Connection struct {
	httpClient
	url       *url.URL
	useragent string
	transport *http.Transport
	catcher   catcher.Catcher
}

func newConnection(useragent string, client httpClient) *Connection {
	return &Connection{
		httpClient: client,
		useragent:  useragent,
	}
}

// NewConnection creates a new connection with the given useragent string
// and a base URL set up to talk to http://localhost:defaultport
func NewConnection(useragent string) *Connection {
	u, _ := url.Parse("http://localhost:" + strconv.Itoa(DefaultPort))
	return NewConnectionToURL(useragent, u)
}

// NewConnectionTo creates a new connection with the given useragent string
// and a base URL derived from the hostURL argument.
func NewConnectionTo(useragent string, hostURL string) (*Connection, error) {
	url, err := url.Parse(hostURL)
	if err != nil {
		return nil, err
	}
	return NewConnectionToURL(useragent, url), nil
}

// NewConnectionToURL creates a new connection object with the given useragent string
// and a base URL specified by the url argument.
func NewConnectionToURL(useragent string, url *url.URL) *Connection {
	c := newConnection(useragent, &http.Client{})
	c.url = url
	return c
}

type testHttpClient struct {
	handler http.Handler
}

func (t testHttpClient) Do(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	t.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

func NewTestConnection(useragent string, h http.Handler) *Connection {
	c := newConnection(useragent, testHttpClient{h})
	c.url, _ = url.Parse("http://localhost:" + strconv.Itoa(DefaultPort))
	return c
}

// SetTimeout sets the underlying http request timeout to the given duration
func (c *Connection) SetTimeout(to time.Duration) {
	if client, ok := c.httpClient.(*http.Client); ok {
		client.Timeout = to
	}
}

func (c *Connection) URL() *url.URL {
	return c.url
}

func (c *Connection) SetURL(u *url.URL) {
	c.url = u
}

func (c *Connection) SetCatcher(p catcher.Catcher) {
	c.catcher = p
}

func (c *Connection) SetTLS(proxy func(req *http.Request) (*url.URL, error), skipVerify bool) {
	c.transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipVerify,
		},
		Proxy: proxy,
	}
	c.url.Scheme = "https" //XXX
}

// bodyToReader takes the body argument and figures out if it's already
// a reader, and if not, tries to encode it as json into a bytes buffer
// and returns the result as a reader.
func bodyToReader(body interface{}) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	reader, ok := body.(io.Reader)
	if ok {
		return reader, nil
	}
	// encode the body data structure into json as a bytes.Buffer so
	// it can be turned around and read by the http request
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		return nil, err
	}
	return buf, nil
}

// Request constructs and returns a http.Request using the zqd connection info
// provided to the client.
func (c *Connection) Request(method, path string, hdr http.Header, body interface{}) (*http.Request, error) {
	u, err := c.url.Parse(path)
	if err != nil {
		return nil, err
	}
	bodyReader, err := bodyToReader(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	if hdr != nil {
		req.Header = hdr
	}
	if c.useragent != "" {
		req.Header.Set("User-Agent", c.useragent)
	}
	return req, nil
}

// _call transmits a request to the server and waits for a response and
// returns the http.Response.  The request is constructed from method, path, body.
// The body can be an io.Reader for the big stuff.  Body can also be an arbitrary
// gc protocol object type-aligned to the expected parameter.
func (c *Connection) _call(ctx context.Context, method, path string, hdr http.Header, body interface{}) (*http.Response, context.CancelFunc, error) {
	req, err := c.Request(method, path, hdr, body)
	if err != nil {
		return nil, nil, err
	}
	var cancel context.CancelFunc
	if c.catcher != nil {
		ctx, cancel = c.catcher.Catch(ctx)
	}
	req = req.WithContext(ctx)
	resp, err := c.Do(req)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return resp, nil, err
	}
	err = responseError(resp)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return resp, nil, err
	}
	return resp, cancel, nil
}

// call transmits a request to the server and waits for a response and
// returns the response in the provided result object represented here
// as an empty interface.  The request is constructed from method, path, body.
// If a result is expected, then the result value is filled in.  Otherwise,
// result should be nil.
func (c *Connection) call(method, path string, hdr http.Header, body, result interface{}) (status int, err error) {
	resp, cancel, err := c._call(context.Background(), method, path, hdr, body)
	if cancel != nil {
		cancel()
	}
	if resp != nil {
		status = resp.StatusCode
	}
	if err != nil {
		return status, err
	}
	if result == nil {
		return status, nil
	}
	err = json.NewDecoder(resp.Body).Decode(result)
	err2 := resp.Body.Close()
	if err != nil {
		return status, err
	}
	return status, err2
}

// stream transmits a request to the server and returns a bufio.Scanner that
// parses double-newline chunks of data and returns them as byte slices.
// It's up to the caller to wrap a decoder around this scanner.
func (c *Connection) stream(ctx context.Context, method, path string, hdr http.Header, body interface{}) (*bufio.Scanner, context.CancelFunc, error) {
	resp, cancel, err := c._call(ctx, method, path, hdr, body)
	if err != nil {
		// cancel is nil
		return nil, nil, err
	}
	return NewJSONPipeScanner(resp.Body), cancel, nil
}

func responseError(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var apierr Error
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		err := json.Unmarshal(body, &apierr)
		if err != nil {
			return err
		}

	} else {
		apierr.Type = "Error"
		apierr.Message = string(body)
	}
	return &apierr
}

// Ping checks to see if the server and measure the time it takes to
// get back the response.
func (c *Connection) Ping() (time.Duration, error) {
	now := time.Now()
	if _, err := c.call("GET", "/status", nil, nil, nil); err != nil {
		return 0, err
	}
	return time.Since(now), nil
}

// SpaceInfo retrieves information about the specified space.
func (c *Connection) SpaceInfo(spaceID string) (*SpaceInfo, error) {
	path := path.Join("/space", spaceID)
	var res SpaceInfo
	if status, err := c.call("GET", path, nil, nil, &res); err != nil {
		if status == http.StatusNotFound {
			return nil, ErrSpaceNotFound
		}
		return nil, err
	}
	return &res, nil
}

func (c *Connection) SpacePost(spaceName string) (*SpaceInfo, error) {
	body := &SpacePostRequest{Name: spaceName}
	var res SpaceInfo
	if status, err := c.call("POST", "/space", nil, body, &res); err != nil {
		if status == http.StatusConflict {
			return nil, ErrSpaceExists
		}
		return nil, err
	}
	return &res, nil
}

func (c *Connection) SpaceList() ([]string, error) {
	var res []string
	_, err := c.call("GET", "/space", nil, nil, &res)
	return res, err
}

// Not Yet
// func (c *Connection) SpaceDelete(spaceName string) (err error) {
// path := "/space/" + spaceName
// _, err = c.call("DELETE", path, nil, nil, nil)
// return err
// }

/* not yet

// Packets xxx
func (c *Connection) Packets(spaceName string, s *gc.PacketSearch) ([]byte, error) {
	path := "/space/" + spaceName + "/packet"
	t.Method = "GET"
	// XXX need to add s.ToQuery() to path XXX
	//t.Query = s.ToQuery() XXX

	if err := t.send(); err != nil {
		return nil, err
	}
	//XXX this should stream a pcap not read the whole thing and send it
	return t.recv()
}

// SearchSync xxx.
func (c *Connection) SearchSync(s gc.SearchRequest) ([]*tuple.Tuple, error) {
	return c.newTask().SearchSync(s)
}
*/

// Close releases the connection's resources.
func (c *Connection) Close() error {
	// notyet c.transport.CloseIdleConnections()
	return nil
}

func gzHeader() http.Header {
	hdr := make(http.Header)
	hdr.Set("Accept-Encoding", "identity")
	hdr.Set("Content-Encoding", "gzip")
	return hdr
}

// PostSearch sends a search task to the server and returns a Search interface
// that the caller uses to stream back results via the Pull method.
func (c *Connection) PostSearch(req SearchRequest, format string, params url.Values) (Search, error) {
	return c.PostSearchWithContext(context.Background(), req, format, params)
}

// PostSearchWithContext is like PostSearch, except that it takes a
// context that will be used in the underlying http request.
func (c *Connection) PostSearchWithContext(ctx context.Context, req SearchRequest, format string, params url.Values) (Search, error) {
	// XXX Format is passed in here as an option separate from the query params
	// since we will change this to a content-encoding header instead of a
	// query param (PROD-1189).
	if format == "" {
		format = "bzng"
	}
	if params == nil {
		params = url.Values{}
	}
	params.Set("format", format)
	path := "/search"
	if params.Encode() != "" {
		path += "?" + params.Encode()
	}
	resp, cancel, err := c._call(ctx, "POST", path, nil, req)
	if err != nil {
		return nil, err
	}
	switch format {
	default:
		// POST should catch this above
		return nil, errors.New("bad search format requested: " + format)
	case "json", "zjson":
		return NewJsonSearch(resp.Body, cancel), nil
	case "bzng":
		return NewBzngSearch(resp.Body, cancel), nil
	}
}

func (c *Connection) PostPacket(space string, req PacketPostRequest) (*Stream, error) {
	u := path.Join("/space", space, "packet")
	jsonpipe, cancel, err := c.stream(context.Background(), "POST", u, nil, req)
	if err != nil {
		return nil, err
	}
	return NewStream(jsonpipe, cancel), nil
}

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/go-resty/resty/v2"
	"github.com/segmentio/ksuid"
)

const (
	// DefaultPort zqd port to connect with.
	DefaultPort      = 9867
	DefaultUserAgent = "zqd-client-golang"
)

var (
	// ErrPoolNotFound returns when specified pool does not exist.
	ErrPoolNotFound = errors.New("pool not found")
	// ErrPoolExists returns when specified the pool already exists.
	ErrPoolExists = errors.New("pool exists")
)

type Connection struct {
	client *resty.Client
}

func newConnection(client *resty.Client) *Connection {
	client.SetError(api.Error{})
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

// ClientHostURL allows us to print the host in log messages and internal error messages
func (c *Connection) ClientHostURL() string {
	return c.client.HostURL
}

func (c *Connection) SetAuthToken(token string) {
	c.client.SetAuthToken(token)
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
		resErr.Err = err.(*api.Error)
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
		var apierr api.Error
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
	req := c.client.R().SetContext(ctx)
	if requestID := api.RequestIDFromContext(ctx); requestID != "" {
		req = req.SetHeader(api.RequestIDHeader, requestID)
	}
	return req
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

// Version retrieves the version string from the service.
func (c *Connection) Version(ctx context.Context) (string, error) {
	resp, err := c.Request(ctx).
		SetResult(&api.VersionResponse{}).
		Get("/version")
	if err != nil {
		return "", err
	}
	return resp.Result().(*api.VersionResponse).Version, nil
}

// ZtoAST sends a request to the server to translate a Z program into its
// AST form.
func (c *Connection) ZtoAST(ctx context.Context, zprog string) ([]byte, error) {
	resp, err := c.Request(ctx).
		SetBody(api.ASTRequest{ZQL: zprog}).
		Post("/ast")
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

// PoolInfo retrieves information about the specified pool.
func (c *Connection) PoolInfo(ctx context.Context, id ksuid.KSUID) (*api.PoolInfo, error) {
	path := path.Join("/pool", id.String())
	resp, err := c.Request(ctx).
		SetResult(&api.PoolInfo{}).
		Get(path)
	if err != nil {
		if r, ok := err.(*ErrorResponse); ok && r.StatusCode() == http.StatusNotFound {
			return nil, ErrPoolNotFound
		}
		return nil, err
	}
	return resp.Result().(*api.PoolInfo), nil
}

func (c *Connection) PoolPost(ctx context.Context, req api.PoolPostRequest) (*api.Pool, error) {
	resp, err := c.Request(ctx).
		SetBody(req).
		SetResult(&api.Pool{}).
		Post("/pool")
	if err != nil {
		if r, ok := err.(*ErrorResponse); ok && r.StatusCode() == http.StatusConflict {
			return nil, ErrPoolExists
		}
		return nil, err
	}
	return resp.Result().(*api.Pool), nil
}

func (c *Connection) PoolPut(ctx context.Context, id ksuid.KSUID, req api.PoolPutRequest) error {
	_, err := c.Request(ctx).
		SetBody(req).
		Put(path.Join("/pool", id.String()))
	return err
}

func (c *Connection) PoolList(ctx context.Context) ([]api.Pool, error) {
	var res []api.Pool
	_, err := c.Request(ctx).
		SetResult(&res).
		Get("/pool")
	return res, err
}

func (c *Connection) PoolDelete(ctx context.Context, id ksuid.KSUID) (err error) {
	path := path.Join("/pool", id.String())
	_, err = c.Request(ctx).Delete(path)
	if r, ok := err.(*ErrorResponse); ok && r.StatusCode() == http.StatusNotFound {
		return ErrPoolNotFound
	}
	return err
}

func (c *Connection) SearchRaw(ctx context.Context, search api.SearchRequest, params map[string]string) (io.ReadCloser, error) {
	req := c.Request(ctx).
		SetBody(search).
		SetQueryParam("format", "zng")
	req.SetQueryParams(params)
	req.Method = http.MethodPost
	req.URL = "/search"
	return c.stream(req)
}

// Search sends a search request to the server and returns a ZngSearch
// that the caller uses to stream back results via the Read method.
// Example usage:
//
//	conn := client.NewConnectionTo("http://localhost:9867")
//	poolID, err := conn.PoolLookup(ctx, "poolName")
//	if err != nil { return err }
//	search, err := conn.Search(ctx, poolID, "_path=conn | count()")
//	if err != nil { return err }
//	for {
//		rec, err := search.Read()
//		if err != nil { return err }
//		if rec == nil {
//			// End of results.
//			return nil
//		}
//		fmt.Println(rec)
//	}
//
func (c *Connection) Search(ctx context.Context, id ksuid.KSUID, query string) (*ZngSearch, error) {
	procBytes, err := c.ZtoAST(ctx, query)
	if err != nil {
		return nil, err
	}
	r, err := c.SearchRaw(ctx, api.SearchRequest{
		Pool: id,
		Proc: procBytes,
		Dir:  -1,
	}, nil)
	if err != nil {
		return nil, err
	}
	return NewZngSearch(r), nil
}

func (c *Connection) IndexSearch(ctx context.Context, id ksuid.KSUID, search api.IndexSearchRequest, params map[string]string) (*ZngSearch, error) {
	req := c.Request(ctx).
		SetBody(search).
		SetQueryParam("format", "zng")
	req.SetQueryParams(params)
	req.Method = http.MethodPost
	req.URL = path.Join("/pool", id.String(), "indexsearch")
	r, err := c.stream(req)
	if err != nil {
		return nil, err
	}
	return NewZngSearch(r), nil
}

func (c *Connection) IndexPost(ctx context.Context, id ksuid.KSUID, post api.IndexPostRequest) error {
	_, err := c.Request(ctx).
		SetBody(post).
		Post(path.Join("/pool", id.String(), "index"))
	return err
}

type LogPostOpts struct {
	StopError bool
	Shaper    ast.Proc
}

func (c *Connection) LogPostPath(ctx context.Context, id ksuid.KSUID, opts *LogPostOpts, paths ...string) (*Stream, error) {
	body := api.LogPostRequest{
		Paths:   paths,
		StopErr: opts.StopError,
	}
	if opts != nil && opts.Shaper != nil {
		raw, err := json.Marshal(opts.Shaper)
		if err != nil {
			return nil, err
		}
		body.Shaper = raw
	}
	req := c.Request(ctx).
		SetBody(body)
	req.Method = http.MethodPost
	req.URL = path.Join("/pool", id.String(), "log/paths")
	r, err := c.stream(req)
	if err != nil {
		return nil, err
	}
	jsonpipe := NewJSONPipeScanner(r)
	return NewStream(jsonpipe), nil
}

func (c *Connection) LogPost(ctx context.Context, id ksuid.KSUID, opts *LogPostOpts, paths ...string) (api.LogPostResponse, error) {
	w, err := MultipartFileWriter(paths...)
	if err != nil {
		return api.LogPostResponse{}, err
	}
	return c.LogPostWriter(ctx, id, opts, w)
}

func (c *Connection) LogPostReaders(ctx context.Context, id ksuid.KSUID, opts *LogPostOpts, readers ...io.Reader) (api.LogPostResponse, error) {
	w, err := MultipartDataWriter(readers...)
	if err != nil {
		return api.LogPostResponse{}, err
	}
	return c.LogPostWriter(ctx, id, opts, w)
}

func (c *Connection) LogPostWriter(ctx context.Context, id ksuid.KSUID, opts *LogPostOpts, writer *MultipartWriter) (api.LogPostResponse, error) {
	req := c.Request(ctx).
		SetBody(writer).
		SetResult(&api.LogPostResponse{}).
		SetHeader("Content-Type", writer.ContentType())
	if opts != nil {
		if opts.Shaper != nil {
			writer.SetShaper(opts.Shaper)
		}
		if opts.StopError {
			req.SetQueryParam("stop_err", "true")
		}
	}
	u := path.Join("/pool", id.String(), "log")
	resp, err := req.Post(u)
	if err != nil {
		return api.LogPostResponse{}, err
	}
	v := resp.Result().(*api.LogPostResponse)
	return *v, nil
}

func (c *Connection) AuthMethod(ctx context.Context) (*api.AuthMethodResponse, error) {
	resp, err := c.Request(ctx).
		SetResult(&api.AuthMethodResponse{}).
		Get("/auth/method")
	if err != nil {
		return nil, err
	}
	return resp.Result().(*api.AuthMethodResponse), nil
}

func (c *Connection) AuthIdentity(ctx context.Context) (*api.AuthIdentityResponse, error) {
	resp, err := c.Request(ctx).
		SetResult(&api.AuthIdentityResponse{}).
		Get("/auth/identity")
	if err != nil {
		return nil, err
	}
	return resp.Result().(*api.AuthIdentityResponse), nil
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

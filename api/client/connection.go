package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/go-resty/resty/v2"
	"github.com/segmentio/ksuid"
)

const (
	// DefaultPort zqd port to connect with.
	DefaultPort      = 9867
	DefaultUserAgent = "zqd-client-golang"
)

var (
	// ErrPoolNotFound is returned when the specified pool does not exist.
	ErrPoolNotFound = errors.New("pool not found")
	// ErrPoolExists is returned when the specified the pool already exists.
	ErrPoolExists = errors.New("pool exists")
	// ErrBranchNotFound is returned when the specified branch does not exist.
	ErrBranchNotFound = errors.New("branch not found")
	// ErrBranchExists is returned when the specified the branch already exists.
	ErrBranchExists = errors.New("branch exists")
)

type Response struct {
	Body        io.ReadCloser
	ContentType string
	StatusCode  int
}

type Connection struct {
	client  *resty.Client
	storage storage.Engine
}

func newConnection(client *resty.Client) *Connection {
	client.SetError(api.Error{})
	client.OnAfterResponse(checkError)
	c := &Connection{
		client:  client,
		storage: storage.NewLocalEngine(),
	}
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
	// For now connection only accepts zng responses.
	client.SetHeader("Accept", api.MediaTypeZNG)
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
	req.Header.Set("Accept", api.MediaTypeJSON)
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

func (c *Connection) stream(req *resty.Request) (*Response, error) {
	resp, err := req.SetDoNotParseResponse(true).Send() // disables middleware
	if err != nil {
		return nil, err
	}
	r := resp.RawBody()
	if resp.IsSuccess() {
		typ, _, err := mime.ParseMediaType(resp.Header().Get("Content-Type"))
		if err != nil {
			return nil, err
		}
		return &Response{
			Body:        r,
			ContentType: typ,
			StatusCode:  resp.StatusCode(),
		}, err
	}
	defer r.Close()
	body, err := io.ReadAll(r)
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

// ZedToAST sends a request to the server to translate a Zed program into its
// AST form.
func (c *Connection) ZedToAST(ctx context.Context, zprog string) ([]byte, error) {
	resp, err := c.Request(ctx).
		SetBody(api.ASTRequest{ZQL: zprog}).
		SetHeader("Accept", api.MediaTypeJSON).
		Post("/ast")
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c *Connection) PoolStats(ctx context.Context, id ksuid.KSUID) (*Response, error) {
	req := c.Request(ctx)
	req.Method = http.MethodGet
	req.URL = path.Join("/pool", id.String(), "stats")
	r, err := c.stream(req)
	var errRes *ErrorResponse
	if errors.As(err, &errRes) && errRes.StatusCode() == http.StatusNotFound {
		return nil, ErrPoolNotFound
	}
	return r, err
}

func (c *Connection) BranchGet(ctx context.Context, poolID ksuid.KSUID, branchName string) (*Response, error) {
	req := c.Request(ctx)
	req.Method = http.MethodGet
	req.URL = urlPath("pool", poolID.String(), "branch", branchName)
	r, err := c.stream(req)
	var errRes *ErrorResponse
	if errors.As(err, &errRes) && errRes.StatusCode() == http.StatusNotFound {
		return nil, ErrBranchNotFound
	}
	return r, err
}

func (c *Connection) PoolPost(ctx context.Context, payload api.PoolPostRequest) (*Response, error) {
	req := c.Request(ctx).
		SetBody(payload)
	req.Method = http.MethodPost
	req.URL = "/pool"
	resp, err := c.stream(req)
	var errRes *ErrorResponse
	if errors.As(err, &errRes) && errRes.StatusCode() == http.StatusConflict {
		return nil, ErrPoolExists
	}
	return resp, err
}

func (c *Connection) PoolPut(ctx context.Context, id ksuid.KSUID, req api.PoolPutRequest) error {
	_, err := c.Request(ctx).
		SetBody(req).
		Put(path.Join("/pool", id.String()))
	return err
}

func (c *Connection) PoolRemove(ctx context.Context, id ksuid.KSUID) error {
	path := path.Join("/pool", id.String())
	_, err := c.Request(ctx).Delete(path)
	if r, ok := err.(*ErrorResponse); ok && r.StatusCode() == http.StatusNotFound {
		return ErrPoolNotFound
	}
	return err
}

func (c *Connection) BranchPost(ctx context.Context, poolID ksuid.KSUID, payload api.BranchPostRequest) (*Response, error) {
	req := c.Request(ctx).
		SetBody(payload)
	req.Method = http.MethodPost
	req.URL = path.Join("/pool", poolID.String())
	resp, err := c.stream(req)
	var errRes *ErrorResponse
	if errors.As(err, &errRes) && errRes.StatusCode() == http.StatusConflict {
		return nil, ErrBranchExists
	}
	return resp, err
}

func (c *Connection) MergeBranch(ctx context.Context, poolID ksuid.KSUID, childBranch, parentBranch string, message api.CommitMessage) (*Response, error) {
	req := c.Request(ctx)
	if err := encodeCommitMessage(req, message); err != nil {
		return nil, err
	}
	req.Method = http.MethodPost
	req.URL = urlPath("pool", poolID.String(), "branch", parentBranch, "merge", childBranch)
	return c.stream(req)
}

func (c *Connection) Undo(ctx context.Context, poolID ksuid.KSUID, branchName string, commitID ksuid.KSUID, message api.CommitMessage) (*Response, error) {
	req := c.Request(ctx)
	if err := encodeCommitMessage(req, message); err != nil {
		return nil, err
	}
	req.Method = http.MethodPost
	req.URL = urlPath("pool", poolID.String(), "branch", branchName, "undo", commitID.String())
	return c.stream(req)
}

func (c *Connection) SearchRaw(ctx context.Context, search api.SearchRequest, params map[string]string) (*Response, error) {
	req := c.Request(ctx).
		SetBody(search).
		SetQueryParam("format", "zng")
	req.SetQueryParams(params)
	req.Method = http.MethodPost
	req.URL = "/search"
	return c.stream(req)
}

func (c *Connection) Query(ctx context.Context, src string, filenames ...string) (*Response, error) {
	src, srcInfo, err := parser.ConcatSource(filenames, src)
	if err != nil {
		return nil, err
	}
	req := c.Request(ctx).
		SetBody(api.QueryRequest{Query: src})
	req.Method = http.MethodPost
	req.URL = "/query"
	res, err := c.stream(req)
	var ae *api.Error
	if errors.As(err, &ae) {
		if m, ok := ae.Info.(map[string]interface{}); ok {
			if offset, ok := m["parse_error_offset"].(float64); ok {
				return res, parser.NewError(src, srcInfo, int(offset))
			}
		}
	}
	return res, err
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
func (c *Connection) Search(ctx context.Context, id ksuid.KSUID, query string) (*Response, error) {
	procBytes, err := c.ZedToAST(ctx, query)
	if err != nil {
		return nil, err
	}
	return c.SearchRaw(ctx, api.SearchRequest{
		Pool: api.KSUID(id),
		Proc: procBytes,
		Dir:  -1,
	}, nil)
}

func (c *Connection) IndexPost(ctx context.Context, id ksuid.KSUID, post api.IndexPostRequest) error {
	_, err := c.Request(ctx).
		SetBody(post).
		Post(path.Join("/pool", id.String(), "index"))
	return err
}

func (c *Connection) Load(ctx context.Context, poolID ksuid.KSUID, branchName string, r io.Reader, message api.CommitMessage) (*Response, error) {
	req := c.Request(ctx).
		SetBody(r)
	if err := encodeCommitMessage(req, message); err != nil {
		return nil, err
	}
	req.Method = http.MethodPost
	req.URL = urlPath("pool", poolID.String(), "branch", branchName)
	return c.stream(req)
}

func encodeCommitMessage(req *resty.Request, message api.CommitMessage) error {
	encoded, err := json.Marshal(message)
	if err != nil {
		return err
	}
	req.SetHeader("Zed-Commit", string(encoded))
	return nil
}

func (c *Connection) Delete(ctx context.Context, poolID ksuid.KSUID, branchName string, ids []ksuid.KSUID, message api.CommitMessage) (*Response, error) {
	req := c.Request(ctx).
		SetBody(ids)
	if err := encodeCommitMessage(req, message); err != nil {
		return nil, err
	}
	req.Method = http.MethodPost
	req.URL = urlPath("pool", poolID.String(), "branch", branchName, "delete")
	return c.stream(req)
}

func (c *Connection) AuthMethod(ctx context.Context) (*Response, error) {
	req := c.Request(ctx)
	req.Method = http.MethodGet
	req.URL = "/auth/method"
	return c.stream(req)
}

func (c *Connection) AuthIdentity(ctx context.Context) (*Response, error) {
	req := c.Request(ctx)
	req.Method = http.MethodGet
	req.URL = "/auth/identity"
	return c.stream(req)
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

func urlPath(elem ...string) string {
	var s string
	for _, e := range elem {
		s += "/" + url.PathEscape(e)
	}
	return path.Clean(s)
}

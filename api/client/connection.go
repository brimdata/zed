package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/branches"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
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

type Connection struct {
	client        *http.Client
	defaultHeader http.Header
	hostURL       string
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
	h := http.Header{"Accept": []string{api.MediaTypeZNG}}
	return &Connection{
		client:        &http.Client{},
		defaultHeader: h,
		hostURL:       hostURL,
	}
}

// ClientHostURL allows us to print the host in log messages and internal error messages
func (c *Connection) ClientHostURL() string {
	return c.hostURL
}

func (c *Connection) SetAuthToken(token string) {
	value := fmt.Sprintf("Bearer %s", token)
	c.defaultHeader.Set(http.CanonicalHeaderKey("Authorization"), value)
}

func (c *Connection) SetUserAgent(useragent string) {
	c.defaultHeader.Set("User-Agent", useragent)
}

type Response struct {
	*http.Response
	Duration time.Duration
}

func (c *Connection) Do(req *Request) (*Response, error) {
	httpreq, err := req.HTTPRequest()
	if err != nil {
		return nil, err
	}
	res, err := c.client.Do(httpreq)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = parseError(res)
	}
	return &Response{
		Response: res,
		Duration: req.Duration(),
	}, err
}

func (c *Connection) doAndUnmarshal(req *Request, i interface{}) error {
	res, err := c.Do(req)
	if err != nil {
		return err
	}
	zr := zngio.NewReader(res.Body, zed.NewContext())
	rec, err := zr.Read()
	if err != nil {
		return err
	}
	if rec == nil {
		return errors.New("expected one record from response but got none")
	}
	return zson.UnmarshalZNGRecord(rec, i)
}

// parseError parses an error from an http.Response with an error status code. For now the response type of errors is assumed to be JSON.
func parseError(r *http.Response) error {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	resErr := &ErrorResponse{Response: r}
	if r.Header.Get("Content-Type") == api.MediaTypeJSON {
		var apierr api.Error
		if err := json.Unmarshal(body, &apierr); err != nil {
			return err
		}
		resErr.Err = &apierr
	} else {
		resErr.Err = errors.New(string(body))
	}
	return resErr
}

func errIsStatus(err error, code int) bool {
	var errRes *ErrorResponse
	return errors.As(err, &errRes) && errRes.StatusCode == code
}

func (c *Connection) NewRequest(ctx context.Context, method, path string, body interface{}) *Request {
	h := make(http.Header)
	for key, val := range c.defaultHeader {
		h[key] = val
	}
	req := newRequest(ctx, c.hostURL, h)
	req.Method = method
	req.Path = path
	req.Body = body
	return req
}

// Ping checks to see if the server and measure the time it takes to
// get back the response.
func (c *Connection) Ping(ctx context.Context) (time.Duration, error) {
	req := c.NewRequest(ctx, http.MethodGet, "/status", nil)
	res, err := c.Do(req)
	res.Body.Close()
	if err != nil {
		return 0, err
	}
	return res.Duration, nil
}

// Version retrieves the version string from the service.
func (c *Connection) Version(ctx context.Context) (string, error) {
	req := c.NewRequest(ctx, http.MethodGet, "/version", nil)
	var res api.VersionResponse
	if err := c.doAndUnmarshal(req, &res); err != nil {
		return "", err
	}
	return res.Version, nil
}

func (c *Connection) PoolStats(ctx context.Context, id ksuid.KSUID) (lake.PoolStats, error) {
	req := c.NewRequest(ctx, http.MethodGet, path.Join("/pool", id.String(), "stats"), nil)
	var stats lake.PoolStats
	err := c.doAndUnmarshal(req, &stats)
	if errIsStatus(err, http.StatusNotFound) {
		err = ErrPoolNotFound
	}
	return stats, err
}

func (c *Connection) BranchGet(ctx context.Context, poolID ksuid.KSUID, branchName string) (api.CommitResponse, error) {
	path := urlPath("pool", poolID.String(), "branch", branchName)
	req := c.NewRequest(ctx, http.MethodGet, path, nil)
	var commit api.CommitResponse
	err := c.doAndUnmarshal(req, &commit)
	if errIsStatus(err, http.StatusNotFound) {
		err = ErrBranchNotFound
	}
	return commit, err
}

func (c *Connection) PoolPost(ctx context.Context, payload api.PoolPostRequest) (lake.BranchMeta, error) {
	req := c.NewRequest(ctx, http.MethodPost, "/pool", payload)
	var meta lake.BranchMeta
	err := c.doAndUnmarshal(req, &meta)
	if errIsStatus(err, http.StatusConflict) {
		err = ErrPoolExists
	}
	return meta, err
}

func (c *Connection) PoolPut(ctx context.Context, id ksuid.KSUID, put api.PoolPutRequest) error {
	req := c.NewRequest(ctx, http.MethodPut, path.Join("/pool", id.String()), put)
	res, err := c.Do(req)
	res.Body.Close()
	return err
}

func (c *Connection) PoolRemove(ctx context.Context, id ksuid.KSUID) error {
	req := c.NewRequest(ctx, http.MethodDelete, path.Join("/pool", id.String()), nil)
	res, err := c.Do(req)
	res.Body.Close()
	if errIsStatus(err, http.StatusNotFound) {
		err = ErrPoolNotFound
	}
	return err
}

func (c *Connection) BranchPost(ctx context.Context, poolID ksuid.KSUID, payload api.BranchPostRequest) (branches.Config, error) {
	req := c.NewRequest(ctx, http.MethodPost, path.Join("/pool", poolID.String()), payload)
	var branch branches.Config
	err := c.doAndUnmarshal(req, &branch)
	if errIsStatus(err, http.StatusConflict) {
		err = ErrBranchExists
	}
	return branch, err
}

func (c *Connection) MergeBranch(ctx context.Context, poolID ksuid.KSUID, childBranch, parentBranch string, message api.CommitMessage) (api.CommitResponse, error) {
	path := urlPath("pool", poolID.String(), "branch", parentBranch, "merge", childBranch)
	req := c.NewRequest(ctx, http.MethodPost, path, nil)
	if err := encodeCommitMessage(req, message); err != nil {
		return api.CommitResponse{}, err
	}
	var commit api.CommitResponse
	err := c.doAndUnmarshal(req, &commit)
	return commit, err
}

func (c *Connection) Revert(ctx context.Context, poolID ksuid.KSUID, branchName string, commitID ksuid.KSUID, message api.CommitMessage) (api.CommitResponse, error) {
	path := urlPath("pool", poolID.String(), "branch", branchName, "revert", commitID.String())
	req := c.NewRequest(ctx, http.MethodPost, path, nil)
	if err := encodeCommitMessage(req, message); err != nil {
		return api.CommitResponse{}, err
	}
	var commit api.CommitResponse
	err := c.doAndUnmarshal(req, &commit)
	return commit, err
}

func (c *Connection) Query(ctx context.Context, head *lakeparse.Commitish, src string, filenames ...string) (*Response, error) {
	src, srcInfo, err := parser.ConcatSource(filenames, src)
	if err != nil {
		return nil, err
	}
	body := api.QueryRequest{Query: src}
	if head != nil {
		body.Head = *head
	}
	req := c.NewRequest(ctx, http.MethodPost, "/query", body)
	res, err := c.Do(req)
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

func (c *Connection) Load(ctx context.Context, poolID ksuid.KSUID, branchName string, r io.Reader, message api.CommitMessage) (api.CommitResponse, error) {
	path := urlPath("pool", poolID.String(), "branch", branchName)
	req := c.NewRequest(ctx, http.MethodPost, path, r)
	if err := encodeCommitMessage(req, message); err != nil {
		return api.CommitResponse{}, err
	}
	var commit api.CommitResponse
	err := c.doAndUnmarshal(req, &commit)
	return commit, err
}

func encodeCommitMessage(req *Request, message api.CommitMessage) error {
	encoded, err := json.Marshal(message)
	if err != nil {
		return err
	}
	req.Header.Set("Zed-Commit", string(encoded))
	return nil
}

func (c *Connection) Delete(ctx context.Context, poolID ksuid.KSUID, branchName string, ids []ksuid.KSUID, message api.CommitMessage) (api.CommitResponse, error) {
	path := urlPath("pool", poolID.String(), "branch", branchName, "delete")
	req := c.NewRequest(ctx, http.MethodPost, path, ids)
	if err := encodeCommitMessage(req, message); err != nil {
		return api.CommitResponse{}, err
	}
	var commit api.CommitResponse
	err := c.doAndUnmarshal(req, &commit)
	return commit, err
}

func (c *Connection) AuthMethod(ctx context.Context) (api.AuthMethodResponse, error) {
	req := c.NewRequest(ctx, http.MethodGet, "/auth/method", nil)
	var method api.AuthMethodResponse
	err := c.doAndUnmarshal(req, &method)
	return method, err
}

func (c *Connection) AuthIdentity(ctx context.Context) (api.AuthIdentityResponse, error) {
	req := c.NewRequest(ctx, http.MethodGet, "/auth/identity", nil)
	var ident api.AuthIdentityResponse
	err := c.doAndUnmarshal(req, &ident)
	return ident, err
}

type ErrorResponse struct {
	*http.Response
	Err error
}

func (e *ErrorResponse) Unwrap() error {
	return e.Err
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("status code %d: %v", e.StatusCode, e.Err)
}

func urlPath(elem ...string) string {
	var s string
	for _, e := range elem {
		s += "/" + url.PathEscape(e)
	}
	return path.Clean(s)
}

package devauth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
)

type Config struct {
	Audience   string
	ClientID   string
	Domain     string
	Scope      string
	UserPrompt func(UserCodePrompt) error
}

type UserCodePrompt struct {
	UserCode        string
	VerificationURL string
}

type Result struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	Expiration   time.Time
}

// DeviceAuthorizationFlow implements the Auth0 device authorization flow
// described at:
// https://auth0.com/docs/flows/call-your-api-using-the-device-authorization-flow
// https://auth0.com/docs/api/authentication#device-authorization-flow
func DeviceAuthorizationFlow(ctx context.Context, cfg Config) (Result, error) {
	d, err := newFlow(ctx, cfg)
	if err != nil {
		return Result{}, err
	}

	dcr, err := d.getDeviceCode()
	if err != nil {
		return Result{}, err
	}

	if err := cfg.UserPrompt(UserCodePrompt{
		UserCode:        dcr.UserCode,
		VerificationURL: dcr.VerificationURIComplete,
	}); err != nil {
		return Result{}, err
	}

	return d.pollForResult(dcr)
}

type daFlow struct {
	cfg  Config
	ctx  context.Context
	rcli *resty.Client
}

func newFlow(ctx context.Context, cfg Config) (*daFlow, error) {
	if cfg.UserPrompt == nil {
		return nil, errors.New("no user prompt configured for device authorization")
	}
	if _, err := url.Parse(cfg.Domain); err != nil {
		return nil, fmt.Errorf("invalid auth0 domain url: %w", err)
	}
	return &daFlow{
		cfg: cfg,
		ctx: ctx,
		rcli: resty.New().
			SetError(auth0Error{}).
			SetHostURL(cfg.Domain).
			OnAfterResponse(func(cli *resty.Client, resp *resty.Response) error {
				if resp.IsSuccess() {
					return nil
				}
				if err := resp.Error(); err != nil {
					return err.(*auth0Error)
				}
				return fmt.Errorf("status code %d: %v", resp.StatusCode(), resp.String())
			}),
	}, nil
}

type auth0Error struct {
	Kind             string `json:"error"` // renamed to avoid Error() clash
	ErrorDescription string `json:"error_description"`
}

func (e *auth0Error) Error() string {
	return fmt.Sprintf("auth0 error: %v description: %v", e.Kind, e.ErrorDescription)
}

type deviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
}

func (d *daFlow) getDeviceCode() (deviceCodeResponse, error) {
	res, err := d.rcli.NewRequest().
		SetContext(d.ctx).
		SetFormData(map[string]string{
			"client_id": d.cfg.ClientID,
			"scope":     d.cfg.Scope,
			"audience":  d.cfg.Audience,
		}).
		SetResult(deviceCodeResponse{}).
		Post("/oauth/device/code")
	if err != nil {
		return deviceCodeResponse{}, err
	}
	return *res.Result().(*deviceCodeResponse), nil
}

func (d *daFlow) pollForResult(dcr deviceCodeResponse) (Result, error) {
	type auth0res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}

	delay := time.Duration(dcr.Interval) * time.Second
	if delay <= 0 {
		delay = time.Second
	}

	for {
		select {
		case <-time.After(delay):
		case <-d.ctx.Done():
			return Result{}, d.ctx.Err()
		}
		r, err := d.rcli.NewRequest().
			SetContext(d.ctx).
			SetFormData(map[string]string{
				"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
				"device_code": dcr.DeviceCode,
				"client_id":   d.cfg.ClientID,
			}).
			SetResult(auth0res{}).
			Post("/oauth/token")
		if err != nil {
			if aerr, ok := err.(*auth0Error); ok && aerr.Kind == "authorization_pending" {
				continue
			}
			return Result{}, err
		}
		res := r.Result().(*auth0res)
		return Result{
			AccessToken:  res.AccessToken,
			RefreshToken: res.RefreshToken,
			IDToken:      res.IDToken,
			Expiration:   time.Now().Add(time.Duration(res.ExpiresIn) * time.Second),
		}, nil
	}
}

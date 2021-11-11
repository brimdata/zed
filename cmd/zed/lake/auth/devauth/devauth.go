package devauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
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
	cfg Config
	ctx context.Context
	url url.URL
}

func newFlow(ctx context.Context, cfg Config) (*daFlow, error) {
	if cfg.UserPrompt == nil {
		return nil, errors.New("no user prompt configured for device authorization")
	}
	u, err := url.Parse(cfg.Domain)
	if err != nil {
		return nil, fmt.Errorf("invalid auth0 domain url: %w", err)
	}
	return &daFlow{
		cfg: cfg,
		ctx: ctx,
		url: *u,
	}, nil
}

type auth0Error struct {
	Kind             string `json:"error"` // renamed to avoid Error() clash
	ErrorDescription string `json:"error_description"`
}

func (e *auth0Error) Error() string {
	return fmt.Sprintf("auth0 error: %s: %s", e.Kind, e.ErrorDescription)
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
	req, err := d.newRequest("/oauth/device/code", map[string]string{
		"client_id": d.cfg.ClientID,
		"scope":     d.cfg.Scope,
		"audience":  d.cfg.Audience,
	})
	if err != nil {
		return deviceCodeResponse{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return deviceCodeResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return deviceCodeResponse{}, parseError(resp.Body)
	}
	var payload deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return deviceCodeResponse{}, err
	}
	return payload, nil
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
		req, err := d.newRequest("/oauth/token", map[string]string{
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
			"device_code": dcr.DeviceCode,
			"client_id":   d.cfg.ClientID,
		})
		if err != nil {
			return Result{}, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return Result{}, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			err = parseError(resp.Body)
			if aerr, ok := err.(*auth0Error); ok && aerr.Kind == "authorization_pending" {
				continue
			}
			return Result{}, err
		}
		var payload auth0res
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return Result{}, err
		}
		return Result{
			AccessToken:  payload.AccessToken,
			RefreshToken: payload.RefreshToken,
			IDToken:      payload.IDToken,
			Expiration:   time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second),
		}, nil
	}
}

func (d *daFlow) newRequest(relativePath string, v map[string]string) (*http.Request, error) {
	u := d.url
	u.Path = relativePath
	form := make(url.Values)
	for key, value := range v {
		form.Set(key, value)
	}
	req, err := http.NewRequestWithContext(d.ctx, "POST", u.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

func parseError(body io.Reader) error {
	var aerr auth0Error
	if err := json.NewDecoder(body).Decode(&aerr); err != nil {
		return err
	}
	return &aerr
}

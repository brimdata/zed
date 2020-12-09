package zqd_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/brimsec/zq/api/client"
	"github.com/brimsec/zq/ppl/zqd"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func makeToken(t *testing.T, kid string, c jwt.MapClaims) string {
	b, err := ioutil.ReadFile("testdata/auth-private-key")
	require.NoError(t, err)
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(b)
	require.NoError(t, err)
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims = c
	token.Header["kid"] = kid
	s, err := token.SignedString(privateKey)
	require.NoError(t, err)
	return s
}

func TestAuthentication(t *testing.T) {
	authConfig := zqd.AuthConfig{
		Enabled:  true,
		Audience: "https://test.brimsecurity.com/",
		Issuer:   "https://testauth.brimsecurity.com",
		JWKSPath: "testdata/auth-public-jwks.json",
	}
	core, conn := newCoreWithConfig(t, zqd.Config{
		Logger: zap.NewNop(),
		Auth:   authConfig,
	})
	_, err := conn.SpaceList(context.Background())
	require.Error(t, err)
	require.Equal(t, 1.0, promCounterValue(core.Registry(), "request_errors_unauthorized_total"))

	var resperr *client.ErrorResponse
	require.True(t, errors.As(err, &resperr))
	require.Equal(t, http.StatusUnauthorized, resperr.StatusCode())

	rclient := resty.New()
	resp, err := rclient.R().
		Get(conn.URL() + "/auth/identity")
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode())

	token := makeToken(t, "testkey", map[string]interface{}{
		"aud":            authConfig.Audience,
		"iss":            authConfig.Issuer,
		"brim_tenant_id": "test_tenant_id",
		"brim_user_id":   "test_user_id",
	})
	resp, err = rclient.R().
		SetAuthToken(token).
		Get(conn.URL() + "/auth/identity")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess())

	var ident zqd.Identity
	err = json.Unmarshal(resp.Body(), &ident)
	require.NoError(t, err)
	require.Equal(t, zqd.Identity{
		TenantID: "test_tenant_id",
		UserID:   "test_user_id",
	}, ident)

	resp, err = rclient.R().
		SetAuthToken(token).
		Get(conn.URL() + "/space")
	require.NoError(t, err)
	require.True(t, resp.IsSuccess())
}

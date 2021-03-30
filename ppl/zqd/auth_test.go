package zqd_test

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/ppl/zqd"
	"github.com/brimdata/zed/ppl/zqd/auth"
	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testAuthConfig() zqd.AuthConfig {
	return zqd.AuthConfig{
		Enabled:  true,
		JWKSPath: "testdata/auth-public-jwks.json",
		Domain:   "https://testdomain",
		ClientID: "testclientid",
	}
}

func genToken(t *testing.T, tenantID auth.TenantID, userID auth.UserID) string {
	ac := testAuthConfig()
	token, err := auth.GenerateAccessToken("testkey", "testdata/auth-private-key",
		1*time.Hour, ac.Domain, tenantID, userID)
	require.NoError(t, err)
	return token
}

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

func TestAuthIdentity(t *testing.T) {
	authConfig := testAuthConfig()
	core, conn := newCoreWithConfig(t, zqd.Config{
		Auth:   authConfig,
		Logger: zap.NewNop(),
	})
	_, err := conn.SpaceList(context.Background())
	require.Error(t, err)
	require.Equal(t, 1.0, promCounterValue(core.Registry(), "request_errors_unauthorized_total"))

	var spaceErr *client.ErrorResponse
	require.True(t, errors.As(err, &spaceErr))
	require.Equal(t, http.StatusUnauthorized, spaceErr.StatusCode())

	var identErr *client.ErrorResponse
	_, err = conn.AuthIdentity(context.Background())
	require.Error(t, err)
	require.True(t, errors.As(err, &identErr))
	require.Equal(t, http.StatusUnauthorized, identErr.StatusCode())

	token := genToken(t, "test_tenant_id", "test_user_id")
	conn.SetAuthToken(token)
	res, err := conn.AuthIdentity(context.Background())
	require.NoError(t, err)
	require.Equal(t, &api.AuthIdentityResponse{
		TenantID: "test_tenant_id",
		UserID:   "test_user_id",
	}, res)

	_, err = conn.SpaceList(context.Background())
	require.NoError(t, err)
}

func TestAuthMethodGet(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		_, connNoAuth := newCoreWithConfig(t, zqd.Config{
			Logger: zap.NewNop(),
		})
		resp, err := connNoAuth.AuthMethod(context.Background())
		require.NoError(t, err)
		require.Equal(t, &api.AuthMethodResponse{
			Kind: api.AuthMethodNone,
		}, resp)
	})

	t.Run("auth0", func(t *testing.T) {
		authConfig := testAuthConfig()
		_, connWithAuth := newCoreWithConfig(t, zqd.Config{
			Auth:   authConfig,
			Logger: zap.NewNop(),
		})
		resp, err := connWithAuth.AuthMethod(context.Background())
		require.NoError(t, err)
		require.Equal(t, &api.AuthMethodResponse{
			Kind: "auth0",
			Auth0: &api.AuthMethodAuth0Details{
				Audience: auth.AudienceClaimValue,
				Domain:   authConfig.Domain,
				ClientID: authConfig.ClientID,
			},
		}, resp)
	})
}

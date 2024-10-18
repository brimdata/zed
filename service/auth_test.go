package service_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/brimdata/super/api"
	"github.com/brimdata/super/api/client"
	"github.com/brimdata/super/service"
	"github.com/brimdata/super/service/auth"
	"github.com/stretchr/testify/require"
)

func testAuthConfig() service.AuthConfig {
	return service.AuthConfig{
		Enabled:  true,
		JWKSPath: "testdata/auth-public-jwks.json",
		Audience: "testaudience",
		Domain:   "https://testdomain",
		ClientID: "testclientid",
	}
}

func genToken(t *testing.T, tenantID auth.TenantID, userID auth.UserID) string {
	ac := testAuthConfig()
	token, err := auth.GenerateAccessToken("testkey", "testdata/auth-private-key",
		1*time.Hour, ac.Audience, ac.Domain, tenantID, userID)
	require.NoError(t, err)
	return token
}

func TestAuthIdentity(t *testing.T) {
	authConfig := testAuthConfig()
	core, conn := newCoreWithConfig(t, service.Config{
		Auth: authConfig,
	})
	_, err := conn.Query(context.Background(), nil, false, "from [pools]")
	require.Error(t, err)
	require.Equal(t, 1.0, promCounterValue(core.Registry(), "request_errors_unauthorized_total"))

	var poolErr *client.ErrorResponse
	require.True(t, errors.As(err, &poolErr))
	require.Equal(t, http.StatusUnauthorized, poolErr.StatusCode)

	var identErr *client.ErrorResponse
	_, err = conn.AuthIdentity(context.Background())
	require.Error(t, err)
	require.True(t, errors.As(err, &identErr))
	require.Equal(t, http.StatusUnauthorized, identErr.StatusCode)

	token := genToken(t, "test_tenant_id", "test_user_id")
	conn.SetAuthToken(token)
	res := conn.TestAuthIdentity()
	require.Equal(t, api.AuthIdentityResponse{
		TenantID: "test_tenant_id",
		UserID:   "test_user_id",
	}, res)

	_, err = conn.Query(context.Background(), nil, false, "from :pools")
	require.NoError(t, err)
}

func TestAuthMethodGet(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		_, connNoAuth := newCoreWithConfig(t, service.Config{})
		resp := connNoAuth.TestAuthMethod()
		require.Equal(t, api.AuthMethodResponse{
			Kind: api.AuthMethodNone,
		}, resp)
	})

	t.Run("auth0", func(t *testing.T) {
		authConfig := testAuthConfig()
		_, connWithAuth := newCoreWithConfig(t, service.Config{
			Auth: authConfig,
		})
		resp := connWithAuth.TestAuthMethod()
		require.Equal(t, api.AuthMethodResponse{
			Kind: "auth0",
			Auth0: &api.AuthMethodAuth0Details{
				Audience: authConfig.Audience,
				Domain:   authConfig.Domain,
				ClientID: authConfig.ClientID,
			},
		}, resp)
	})
}

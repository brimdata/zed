package zqd

import (
	"context"
	"crypto/rsa"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/dgrijalva/jwt-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

const (
	// AudienceClaimValue is the value of the "aud" standard claim that clients
	// should use when requesting access tokens for this api.
	// Though formatted as a URL, it does not need to be a reachable location.
	AudienceClaimValue = "https://app.brimsecurity.com"

	// These are the namespaced custom claims we expect on any JWT
	// access token.
	TenantIDClaim = AudienceClaimValue + "/tenant_id"
	UserIDClaim   = AudienceClaimValue + "/user_id"
)

type AuthConfig struct {
	Enabled  bool
	JWKSPath string

	// ClientID and Domain are sent in the /auth/method response so that api
	// clients can interact with the right Auth0 tenant (production, testing, etc)
	// to obtain tokens.
	ClientID string
	Domain   string
}

func (c *AuthConfig) SetFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.Enabled, "auth.enabled", false, "enable authentication checks")
	fs.StringVar(&c.ClientID, "auth.clientid", "", "Auth0 client ID for API clients (will be publicly accessible")
	fs.StringVar(&c.Domain, "auth.domain", "", "Auth0 domain (as a URL) for API clients (will be publicly accessible)")
	fs.StringVar(&c.JWKSPath, "auth.jwkspath", "", "path to JSON Web Key Set file")
}

func (c *AuthConfig) Validate() (*url.URL, error) {
	if !c.Enabled {
		return nil, nil
	}
	if c.ClientID == "" || c.Domain == "" || c.JWKSPath == "" {
		return nil, errors.New("auth.clientid, auth.domain, and auth.jwkspath must be set when auth enabled")
	}
	u, err := url.Parse(c.Domain)
	if err != nil {
		return nil, fmt.Errorf("bad auth.domain URL: %w", err)
	}
	return u, nil
}

type Auth0Authenticator struct {
	checker        *jwtmiddleware.JWTMiddleware
	methodResponse api.AuthMethodResponse
}

// newAuthenticator returns an Auth0Authenticator that checks for a JWT signed
// by a key referenced in the JWKS file, has the required audience and issuer
// claims, and contains claims for a brim tenant and user id.
func newAuthenticator(ctx context.Context, logger *zap.Logger, registerer prometheus.Registerer, config AuthConfig) (*Auth0Authenticator, error) {
	domainURL, err := config.Validate()
	if err != nil {
		return nil, err
	}
	// Auth0 issuer is always the domain URL with trailing "/".
	// https://auth0.com/docs/tokens/access-tokens/get-access-tokens#custom-domains-and-the-management-api
	expectedIssuer := domainURL.String() + "/"
	keys, err := loadPublicKeys(config.JWKSPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load JWKS file: %w", err)
	}
	unauthorized := promauto.With(registerer).NewCounter(prometheus.CounterOpts{
		Name: "request_errors_unauthorized_total",
		Help: "Number of request errors due to bad or missing authorization.",
	})
	checker := jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			claims := token.Claims.(jwt.MapClaims)
			// jwt-go (called from jwtmiddleware) verifies any expiry claim, but
			// will not fail if the expiry claim is missing. The call here with
			// req=true ensures that the claim is both present and valid.
			if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
				return token, errors.New("invalid expiration")
			}
			if !claims.VerifyIssuer(expectedIssuer, true) {
				return token, errors.New("invalid issuer")
			}
			if !verifyAPIAudience(claims) {
				return token, errors.New("invalid audience")
			}
			if tid, _ := claims[TenantIDClaim].(string); tid == "" {
				return token, errors.New("missing tenant id")
			}
			if uid, _ := claims[UserIDClaim].(string); uid == "" {
				return token, errors.New("missing user id")
			}
			tokenKeyID, _ := token.Header["kid"].(string)
			key, ok := keys[tokenKeyID]
			if !ok {
				return token, errors.New("unknown token key id")
			}
			return key, nil
		},
		UserProperty: authTokenContextValue,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, errstr string) {
			unauthorized.Inc()
			logger.Info("unauthorized request",
				zap.String("request_id", getRequestID(r.Context())),
				zap.String("error", errstr))
			http.Error(w, errstr, http.StatusUnauthorized)
		},
		SigningMethod: jwt.SigningMethodRS256,
	})
	return &Auth0Authenticator{
		checker: checker,
		methodResponse: api.AuthMethodResponse{
			Kind: api.AuthMethodAuth0,
			Auth0: &api.AuthMethodAuth0Details{
				Audience: AudienceClaimValue,
				Domain:   config.Domain,
				ClientID: config.ClientID,
			},
		},
	}, nil
}

func (a *Auth0Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.checker.CheckJWT(w, r) != nil {
			// response sent by jwtmiddleware.Options.ErrorHandler
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Auth0Authenticator) MethodResponse() api.AuthMethodResponse {
	return a.methodResponse
}

func verifyAPIAudience(claims jwt.MapClaims) bool {
	// Audience claim may either be a string, or a slice of interfaces that are
	// strings.
	// https://auth0.com/docs/tokens/access-tokens/get-access-tokens#multiple-audiences
	if str, ok := claims["aud"].(string); ok {
		return str == AudienceClaimValue
	}
	if arr, ok := claims["aud"].([]interface{}); ok {
		for _, a := range arr {
			s, _ := a.(string)
			if s == AudienceClaimValue {
				return true
			}
		}
	}
	return false
}

// authTokenContextValue is the string the jwtmiddleware library uses to
// store the validated & parsed JWT in a request's context.
const authTokenContextValue = "zqd-core-auth-token"

type Identity struct {
	TenantID string
	UserID   string
}

func IdentifyFromContext(ctx context.Context) (Identity, bool) {
	var token *jwt.Token
	if token = ctx.Value(authTokenContextValue).(*jwt.Token); token == nil {
		return Identity{}, false
	}
	mc := token.Claims.(jwt.MapClaims)
	tid := mc[TenantIDClaim].(string)
	uid := mc[UserIDClaim].(string)
	return Identity{TenantID: tid, UserID: uid}, true
}

// jwks matches the format of a JSON Web Key Set file:
// https://auth0.com/docs/tokens/json-web-tokens/json-web-key-sets
type jwks struct {
	Keys []struct {
		Kty string   `json:"kty"`
		Kid string   `json:"kid"`
		Use string   `json:"use"`
		N   string   `json:"n"`
		E   string   `json:"e"`
		X5c []string `json:"x5c"`
	} `json:"keys"`
}

func loadPublicKeys(jwkspath string) (map[string]*rsa.PublicKey, error) {
	var jwks jwks
	if err := fs.UnmarshalJSONFile(jwkspath, &jwks); err != nil {
		return nil, err
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, jwk := range jwks.Keys {
		if len(jwk.X5c) == 0 {
			continue
		}
		cert := "-----BEGIN CERTIFICATE-----\n" + jwk.X5c[0] + "\n-----END CERTIFICATE-----"
		public, err := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
		if err != nil {
			return nil, err
		}
		keys[jwk.Kid] = public
	}
	return keys, nil
}

package zqd

import (
	"context"
	"crypto/rsa"
	"errors"
	"flag"
	"fmt"
	"net/http"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/dgrijalva/jwt-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

const (
	// authTokenContextValue is the string the jwtmiddleware library uses to
	// store the validated & parsed JWT in a request's context.
	authTokenContextValue = "zqd-core-auth-token"

	tenantIDClaim = "brim_tenant_id"
	userIDClaim   = "brim_user_id"
)

type AuthConfig struct {
	Enabled  bool
	Audience string
	Issuer   string
	JWKSPath string
}

func (c *AuthConfig) SetFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.Enabled, "auth.enabled", false, "enable authentication checks")
	fs.StringVar(&c.Audience, "auth.audience", "", "expected JWT audience claim")
	fs.StringVar(&c.Issuer, "auth.issuer", "", "expected JWT issuer claim")
	fs.StringVar(&c.JWKSPath, "auth.jwkspath", "", "path to JSON Web Key Set file")
}

func newAuthenticator(ctx context.Context, logger *zap.Logger, registerer prometheus.Registerer, config AuthConfig) (middleware, error) {
	if !config.Enabled {
		return &noopAuthenticator{}, nil
	}
	return newJWTAuthenticator(ctx, logger, registerer, config)
}

type noopAuthenticator struct{}

func (a *noopAuthenticator) Middleware(next http.Handler) http.Handler {
	return next
}

// jwtAuthenticator is a middleware that checks for a JWT signed by a key
// referenced in the JWKS file, has expected audience and issuer claims, and
// contains values for a brim tenant and user id.
type jwtAuthenticator struct {
	checker *jwtmiddleware.JWTMiddleware
	config  AuthConfig
	// keys is a map of jsonWebKeys.Kid to parsed public key.
	keys map[string]*rsa.PublicKey

	unauthorized prometheus.Counter
}

func newJWTAuthenticator(ctx context.Context, logger *zap.Logger, registerer prometheus.Registerer, config AuthConfig) (*jwtAuthenticator, error) {
	keys, err := loadPublicKeys(config.JWKSPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load JWKS file: %w", err)
	}
	factory := promauto.With(registerer)
	a := &jwtAuthenticator{
		config: config,
		keys:   keys,
		unauthorized: factory.NewCounter(prometheus.CounterOpts{
			Name: "request_errors_unauthorized_total",
			Help: "Number of request errors due to bad or missing authorization.",
		}),
	}
	a.checker = jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			claims := token.Claims.(jwt.MapClaims)
			if !claims.VerifyAudience(config.Audience, true) {
				return token, errors.New("invalid audience")
			}
			if !claims.VerifyIssuer(config.Issuer, true) {
				return token, errors.New("invalid issuer")
			}
			if tid, _ := claims[tenantIDClaim].(string); tid == "" {
				return token, errors.New("missing tenant id")
			}
			if uid, _ := claims[userIDClaim].(string); uid == "" {
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
			a.unauthorized.Inc()
			logger.Info("unauthorized request",
				zap.String("request_id", getRequestID(r.Context())),
				zap.String("error", errstr))
			http.Error(w, errstr, http.StatusUnauthorized)
		},
		SigningMethod: jwt.SigningMethodRS256,
	})
	return a, nil
}

func (a *jwtAuthenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := a.checker.CheckJWT(w, r); err != nil {
			// response sent by jwtmiddleware.Options.ErrorHandler
			return
		}
		next.ServeHTTP(w, r)
	})
}

type Identity struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
}

func IdentifyFromContext(ctx context.Context) (Identity, bool) {
	var token *jwt.Token
	if token = ctx.Value(authTokenContextValue).(*jwt.Token); token == nil {
		return Identity{}, false
	}
	mc := token.Claims.(jwt.MapClaims)
	tid := mc[tenantIDClaim].(string)
	uid := mc[userIDClaim].(string)
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

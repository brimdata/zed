package zqd

import (
	"context"
	"errors"
	"flag"
	"net/http"

	"github.com/brimdata/zq/api"
	"github.com/brimdata/zq/ppl/zqd/auth"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
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

type Auth0Authenticator struct {
	logger         *zap.Logger
	methodResponse api.AuthMethodResponse
	unauthorized   prometheus.Counter
	validator      *auth.TokenValidator
}

// NewAuthenticator returns an Auth0Authenticator that checks for a JWT signed
// by a key referenced in the JWKS file, has the required audience and issuer
// claims, and contains claims for a brim tenant and user id.
func NewAuthenticator(ctx context.Context, logger *zap.Logger, registerer prometheus.Registerer, config AuthConfig) (*Auth0Authenticator, error) {
	if config.ClientID == "" || config.Domain == "" || config.JWKSPath == "" {
		return nil, errors.New("auth.clientid, auth.domain, and auth.jwkspath must be set when auth enabled")
	}
	validator, err := auth.NewTokenValidator(config.Domain, config.JWKSPath)
	if err != nil {
		return nil, err
	}
	unauthorized := promauto.With(registerer).NewCounter(prometheus.CounterOpts{
		Name: "request_errors_unauthorized_total",
		Help: "Number of request errors due to bad or missing authorization.",
	})
	return &Auth0Authenticator{
		logger: logger.Named("auth"),
		methodResponse: api.AuthMethodResponse{
			Kind: api.AuthMethodAuth0,
			Auth0: &api.AuthMethodAuth0Details{
				Audience: auth.AudienceClaimValue,
				Domain:   config.Domain,
				ClientID: config.ClientID,
			},
		},
		unauthorized: unauthorized,
		validator:    validator,
	}, nil
}

func (a *Auth0Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ident, err := a.validator.ValidateRequest(r)
		if err != nil {
			a.unauthorized.Inc()
			a.logger.Info("Unauthorized request",
				zap.String("request_id", api.RequestIDFromContext(r.Context())),
				zap.Error(err))
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		ctx := auth.ContextWithAuthToken(r.Context(), token)
		ctx = auth.ContextWithIdentity(ctx, ident)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func (a *Auth0Authenticator) MethodResponse() api.AuthMethodResponse {
	return a.methodResponse
}

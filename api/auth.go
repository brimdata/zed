package api

type AuthIdentityResponse struct {
	TenantID string `json:"tenant_id" zng:"tenant_id"`
	UserID   string `json:"user_id" zng:"user_id"`
}

type AuthMethod string

const (
	AuthMethodNone  AuthMethod = ""
	AuthMethodAuth0 AuthMethod = "auth0"
)

type AuthMethodResponse struct {
	Kind  AuthMethod              `json:"kind" zng:"kind"`
	Auth0 *AuthMethodAuth0Details `json:"auth0,omitempty" zng:"auth0,omitempty"`
}

type AuthMethodAuth0Details struct {
	// Audience is the value to use for the "aud" standard claim when
	// requesting an access token for this service.
	Audience string `json:"audience"`
	// ClientID is the public client id to use when interacting with
	// the above Auth0 domain.
	ClientID string `json:"client_id"`
	// Domain is the Auth0 domain (in url form) to use as the endpoint
	// for any oauth flows.
	Domain string `json:"domain"`
}

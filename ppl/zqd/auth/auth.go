package auth

import (
	"context"
)

type TenantID string
type UserID string

const (
	AnonymousTenantID TenantID = "tenant_000000000000000000000000001"
	AnonymousUserID   UserID   = "user_000000000000000000000000001"
)

type Identity struct {
	TenantID TenantID
	UserID   UserID
}

type identityKey struct{}

func IdentifyFromContext(ctx context.Context) Identity {
	ident, ok := ctx.Value(identityKey{}).(Identity)
	if !ok {
		return Identity{TenantID: AnonymousTenantID, UserID: AnonymousUserID}
	}
	return ident
}

func IdentityToContext(ctx context.Context, ident Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, ident)
}

type jwtKey struct{}

func AuthTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(jwtKey{}).(string)
	return token, ok
}

func AuthTokenToContext(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, jwtKey{}, token)
}

package auth

import (
	"crypto/rsa"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func loadPrivateKey(keyFile string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	return jwt.ParseRSAPrivateKeyFromPEM(b)
}

func makeToken(keyID string, keyFile string, claims jwt.MapClaims) (string, error) {
	privateKey, err := loadPrivateKey(keyFile)
	if err != nil {
		return "", err
	}
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims = claims
	token.Header["kid"] = keyID
	return token.SignedString(privateKey)
}

// GenerateAccessToken creates a JWT in string format with the expected audience,
// issuer, and claims to pass zqd authentication checks.
func GenerateAccessToken(keyID string, privateKeyFile string, expiration time.Duration, domain string, tenantID TenantID, userID UserID) (string, error) {
	dstr, err := url.Parse(domain)
	if err != nil {
		return "", fmt.Errorf("bad domain URL: %w", err)
	}
	return makeToken(keyID, privateKeyFile, jwt.MapClaims{
		"aud":         AudienceClaimValue,
		"exp":         time.Now().Add(expiration).Unix(),
		"iss":         dstr.String() + "/",
		TenantIDClaim: string(tenantID),
		UserIDClaim:   string(userID),
	})
}

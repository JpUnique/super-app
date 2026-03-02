package jwt

import (
	"crypto/rsa"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenGenerator interface {
	GenerateToken(userID string, roles []string, ttl time.Duration) (string, error)
}

type jwtGenerator struct {
	privateKey *rsa.PrivateKey
}

func NewTokenGenerator(privateKeyPath string) (TokenGenerator, error) {
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, ErrSigningKeyError
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return nil, ErrParsingKeyError
	}

	return &jwtGenerator{privateKey: privateKey}, nil
}

func (g *jwtGenerator) GenerateToken(userID string, roles []string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(g.privateKey)
	if err != nil {
		return "", err
	}

	return signed, nil
}

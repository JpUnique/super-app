package jwt

import (
	"crypto/rsa"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type TokenValidator interface {
	ValidateToken(r *http.Request) (*Claims, error)
}

type jwtValidator struct {
	publicKey *rsa.PublicKey
	headerKey string
}

func NewTokenValidator(publicKeyPath, headerKey string) (TokenValidator, error) {
	data, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, ErrParsingKeyError
	}

	pub, err := jwt.ParseRSAPublicKeyFromPEM(data)
	if err != nil {
		return nil, ErrParsingKeyError
	}

	return &jwtValidator{
		publicKey: pub,
		headerKey: headerKey,
	}, nil
}

func (v *jwtValidator) ValidateToken(r *http.Request) (*Claims, error) {
	tokenStr := strings.TrimSpace(r.Header.Get(v.headerKey))
	if tokenStr == "" {
		return nil, ErrTokenMissing
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return v.publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

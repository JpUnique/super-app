package jwt

import "errors"

var (
	ErrTokenInvalid    = errors.New("jwt: invalid token")
	ErrTokenExpired    = errors.New("jwt: token has expired")
	ErrTokenMissing    = errors.New("jwt: token missing from request")
	ErrSigningKeyError = errors.New("jwt: could not load signing key")
	ErrParsingKeyError = errors.New("jwt: could not parse public key")
)

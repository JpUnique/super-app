package auth

import "errors"

var (
	ErrInvalidAPIKey     = errors.New("invalid API key")
	ErrMissingAPIKey     = errors.New("auth: API key missing from request")
	ErrUnauthorized      = errors.New("auth: unauthorized request")
	ErrOAuthTokenInvalid = errors.New("auth: invalid OAuth token")
	ErrOAuthProvider     = errors.New("auth: unsupported OAuth provider")
)

package auth

import (
	"context"
)

type OAuthUser struct {
	ID    string
	Email string
	Name  string
}

type OAuthProvider interface {
	ValidateToken(ctx context.Context, token string) (*OAuthUser, error)
}

type GoogleOAuth struct{}
type AppleOAuth struct{}
type FacebookOAuth struct{}

func (g *GoogleOAuth) ValidateToken(ctx context.Context, token string) (*OAuthUser, error) {
	if token == "" {
		return nil, ErrOAuthTokenInvalid
	}

	// Placeholder logic — real implementation calls Google OAuth API
	return &OAuthUser{
		ID:    "google-123",
		Email: "user@gmail.com",
		Name:  "Google User",
	}, nil
}

func (a *AppleOAuth) ValidateToken(ctx context.Context, token string) (*OAuthUser, error) {
	if token == "" {
		return nil, ErrOAuthTokenInvalid
	}

	return &OAuthUser{
		ID:    "apple-567",
		Email: "apple@icloud.com",
		Name:  "Apple User",
	}, nil
}

func (f *FacebookOAuth) ValidateToken(ctx context.Context, token string) (*OAuthUser, error) {
	if token == "" {
		return nil, ErrOAuthTokenInvalid
	}

	return &OAuthUser{
		ID:    "fb-999",
		Email: "fb@facebook.com",
		Name:  "Facebook User",
	}, nil
}

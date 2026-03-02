package auth

import (
	"context"
	"fmt"
	"net/http"
)

type AuthService interface {
	AuthenticateAPIKey(r *http.Request) error
	AuthenticateOAuth(ctx context.Context, provider, token string) (*OAuthUser, error)
}

type authService struct {
	apiKeyValidator APIKeyValidator
	oauthProviders  map[string]OAuthProvider
}

func NewAuthService(apiKeyValidator APIKeyValidator) AuthService {
	return &authService{
		apiKeyValidator: apiKeyValidator,
		oauthProviders: map[string]OAuthProvider{
			"google":   &GoogleOAuth{},
			"apple":    &AppleOAuth{},
			"facebook": &FacebookOAuth{},
		},
	}
}

func (s *authService) AuthenticateAPIKey(r *http.Request) error {
	return s.apiKeyValidator.ValidateRequest(r)
}

func (s *authService) AuthenticateOAuth(ctx context.Context, provider, token string) (*OAuthUser, error) {
	p, ok := s.oauthProviders[provider]
	if !ok {
		return nil, fmt.Errorf("auth: %w: %s", ErrOAuthProvider, provider)
	}

	return p.ValidateToken(ctx, token)
}

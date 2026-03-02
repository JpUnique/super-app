package auth

import (
	"net/http"
	"strings"
)

type APIKeyValidator interface {
	ValidateRequest(r *http.Request) error
}

type apiKeyValidator struct {
	allowedKeys map[string]struct{}
	headerName  string
}

func NewAPIKeyValidator(headerName string, keys []string) APIKeyValidator {
	keyMap := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		keyMap[k] = struct{}{}
	}

	return &apiKeyValidator{
		allowedKeys: keyMap,
		headerName:  headerName,
	}
}

func (v *apiKeyValidator) ValidateRequest(r *http.Request) error {
	key := strings.TrimSpace(r.Header.Get(v.headerName))
	if key == "" {
		return ErrMissingAPIKey
	}

	if _, exists := v.allowedKeys[key]; !exists {
		return ErrInvalidAPIKey
	}

	return nil
}

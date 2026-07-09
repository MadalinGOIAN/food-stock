//go:build !integration

package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MadalinGOIAN/food-stock/internal/api/auth"
)

const (
	key      = "testKey"
	email    = "user@example.com"
	password = "password"
	name     = "User"
)

func providerReturning(t *testing.T, status int) auth.Provider {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)

	return auth.NewSupabaseProvider(srv.URL, key)
}

func TestProvider_Signup_RateLimited(t *testing.T) {
	p := providerReturning(t, http.StatusTooManyRequests)

	err := p.Signup(context.Background(), auth.Signup{
		Email: email, Password: password, Name: name,
	})
	if !errors.Is(err, auth.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, received %v", err)
	}
}

func TestProvider_Login_RateLimited(t *testing.T) {
	p := providerReturning(t, http.StatusTooManyRequests)

	if _, err := p.Login(context.Background(), auth.Login{
		Email: email, Password: password,
	}); !errors.Is(err, auth.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, received %v", err)
	}
}

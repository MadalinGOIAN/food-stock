//go:build integration

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/MadalinGOIAN/food-stock/internal/api/auth"
)

const password = "password"

func setup(t *testing.T) (auth.Provider, string, string) {
	t.Helper()

	url := os.Getenv("SUPABASE_URL")
	key := os.Getenv("SUPABASE_ANON_KEY")

	if url == "" || key == "" {
		t.Skip("env variables not set; skipping auth integration tests")
	}

	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("test-%d@example.com", timestamp)
	username := fmt.Sprintf("test-%d", timestamp)
	signUp(t, url, key, email, username, password)

	return auth.NewSupabaseProvider(url, key), email, password
}

func signUp(t *testing.T, url, key, email, username, password string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body, err := json.Marshal(map[string]any{
		"email":    email,
		"password": password,
		"data": map[string]string{
			"username": username,
			"name":     "Test User",
		},
	})
	if err != nil {
		t.Fatalf("marshal signup body: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/auth/v1/signup", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build signup request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", key)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("signup request failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("signup returned %d, expected 200: %s", res.StatusCode, b)
	}
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestProvider_Login_Success(t *testing.T) {
	provider, email, password := setup(t)

	token, err := provider.Login(testContext(t), auth.Login{Email: email, Password: password})
	if err != nil {
		t.Fatalf("expected successful login, got error: %v", err)
	}
	if token.AccessToken == "" || token.RefreshToken == "" {
		t.Fatalf("expected non-empty tokens, got %+v", token)
	}
	if token.ExpiresIn <= 0 {
		t.Fatalf("expected positive ExpiresIn, got %d", token.ExpiresIn)
	}
}

func TestProvider_Login_InvalidCredentials(t *testing.T) {
	provider, email, _ := setup(t)

	_, err := provider.Login(testContext(t), auth.Login{Email: email, Password: "wrong-password"})
	if !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestProvider_Refresh_Success(t *testing.T) {
	provider, email, password := setup(t)
	ctx := testContext(t)

	token, err := provider.Login(ctx, auth.Login{Email: email, Password: password})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	refreshed, err := provider.Refresh(ctx, token.RefreshToken)
	if err != nil {
		t.Fatalf("expected successful refresh, got error: %v", err)
	}
	if refreshed.AccessToken == "" || refreshed.RefreshToken == "" {
		t.Fatalf("expected non-empty rotated tokens, got %+v", refreshed)
	}
}

func TestProvider_Refresh_InvalidToken(t *testing.T) {
	provider, _, _ := setup(t)

	_, err := provider.Refresh(testContext(t), "invalid-refresh-token")
	if !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestProvider_Logout(t *testing.T) {
	provider, email, password := setup(t)
	ctx := testContext(t)

	token, err := provider.Login(ctx, auth.Login{Email: email, Password: password})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := provider.Logout(ctx, token.AccessToken); err != nil {
		t.Fatalf("expected successful logout, got error: %v", err)
	}
}

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Provider interface {
	Signup(ctx context.Context, signup Signup) error
	Login(ctx context.Context, login Login) (*Token, error)
	Refresh(ctx context.Context, refreshToken string) (*Token, error)
	Logout(ctx context.Context, accessToken string) error
}

type SupabaseProvider struct {
	url    string
	key    string
	client *http.Client
}

func NewSupabaseProvider(url, key string) *SupabaseProvider {
	return &SupabaseProvider{
		url:    url,
		key:    key,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *SupabaseProvider) Signup(ctx context.Context, signup Signup) error {
	payload := map[string]any{
		"email":    signup.Email,
		"password": signup.Password,
		"data": map[string]string{
			"name": signup.Name,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url+"/auth/v1/signup", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.key)

	res, err := s.client.Do(req)
	if err != nil {
		return ErrUnreachable
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return mapSignupError(res)
	}

	return nil
}

func (s *SupabaseProvider) Login(ctx context.Context, login Login) (*Token, error) {
	return s.requestToken(
		ctx,
		"/auth/v1/token?grant_type=password",
		map[string]string{"email": login.Email, "password": login.Password},
	)
}

func (s *SupabaseProvider) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	return s.requestToken(
		ctx,
		"/auth/v1/token?grant_type=refresh_token",
		map[string]string{"refresh_token": refreshToken},
	)
}

func (s *SupabaseProvider) Logout(ctx context.Context, accessToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url+"/auth/v1/logout", nil)
	if err != nil {
		return err
	}

	req.Header.Set("apikey", s.key)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	res, err := s.client.Do(req)
	if err != nil {
		return ErrUnreachable
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		return fmt.Errorf("logout failed: status %d", res.StatusCode)
	}

	return nil
}

func (s *SupabaseProvider) requestToken(
	ctx context.Context,
	path string,
	payload any,
) (*Token, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.key)

	res, err := s.client.Do(req)
	if err != nil {
		return nil, ErrUnreachable
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if res.StatusCode != http.StatusOK {
		return nil, ErrUnauthorized
	}

	var token Token
	if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
		return nil, ErrBadResponse
	}

	return &token, nil
}

func mapSignupError(res *http.Response) error {
	var errFromProvider struct {
		ErrorCode string `json:"error_code"`
		Msg       string `json:"msg"`
	}
	_ = json.NewDecoder(res.Body).Decode(&errFromProvider)

	switch {
	case res.StatusCode == http.StatusConflict,
		errFromProvider.ErrorCode == "user_already_exists":
		return ErrConflict

	case res.StatusCode == http.StatusTooManyRequests:
		return ErrRateLimited

	case errFromProvider.ErrorCode == "weak_password",
		res.StatusCode == http.StatusUnprocessableEntity,
		res.StatusCode == http.StatusBadRequest:
		return ErrInvalid

	default:
		return ErrBadResponse
	}

}

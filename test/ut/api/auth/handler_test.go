//go:build !integration

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MadalinGOIAN/food-stock/internal/api/auth"
	"github.com/gin-gonic/gin"
)

const (
	accessTokenValue  = "access"
	refreshTokenValue = "refresh"
	expiresIn         = 3600

	validLoginReqBody   = `{"email":"user@example.com","password":"password"}`
	invalidLoginReqBody = `{"email":"user@example.com"}`
	emailValue          = "user@example.com"
	passwordValue       = "password"
)

type stubProvider struct {
	loginToken *auth.Token
	loginErr   error

	refreshToken *auth.Token
	refreshErr   error

	logoutErr error

	gotLogin        auth.Login
	gotRefreshToken string
	gotLogoutToken  string
	logoutCalled    bool
}

func (s *stubProvider) Login(_ context.Context, login auth.Login) (*auth.Token, error) {
	s.gotLogin = login
	return s.loginToken, s.loginErr
}

func (s *stubProvider) Refresh(_ context.Context, refreshToken string) (*auth.Token, error) {
	s.gotRefreshToken = refreshToken
	return s.refreshToken, s.refreshErr
}

func (s *stubProvider) Logout(_ context.Context, accessToken string) error {
	s.logoutCalled = true
	s.gotLogoutToken = accessToken
	return s.logoutErr
}

func newRouterWithSecurity(p auth.Provider, secure bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := auth.NewHandler(p, secure)
	auth.Routes(r.Group("/auth"), h)
	return r
}

func newRouter(p auth.Provider) *gin.Engine {
	return newRouterWithSecurity(p, false)
}

func do(r http.Handler, method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	for _, c := range cookies {
		req.AddCookie(c)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

func findCookie(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func assertCleared(t *testing.T, w *httptest.ResponseRecorder, name string) {
	t.Helper()

	c := findCookie(w, name)
	if c == nil {
		t.Fatalf("expected %q cookie to be present (cleared), got none", name)
	}
	if c.Value != "" || c.MaxAge >= 0 {
		t.Fatalf("expected %q cookie to be cleared, got value=%q maxAge=%d", name, c.Value, c.MaxAge)
	}
}

func assertSessionCookie(t *testing.T, w *httptest.ResponseRecorder, name string, expectedSecure bool) {
	t.Helper()

	c := findCookie(w, name)
	if c == nil {
		t.Fatalf("expected %q cookie, got none", name)
	}
	if !c.HttpOnly {
		t.Errorf("%q cookie must be HttpOnly", name)
	}
	if c.Secure != expectedSecure {
		t.Errorf("%q cookie Secure = %v, want %v", name, c.Secure, expectedSecure)
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("%q cookie SameSite = %v, want Lax", name, c.SameSite)
	}
}

func assertErrorMessage(t *testing.T, w *httptest.ResponseRecorder, expected string) {
	t.Helper()

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("could not decode error body %q: %v", w.Body.String(), err)
	}
	if body.Error != expected {
		t.Fatalf("expected error message %q, got %q", expected, body.Error)
	}
}

func TestLogin_Success(t *testing.T) {
	p := &stubProvider{loginToken: &auth.Token{
		AccessToken:  accessTokenValue,
		RefreshToken: refreshTokenValue,
		ExpiresIn:    expiresIn,
	}}
	r := newRouter(p)

	w := do(r, http.MethodPost, "/auth/login", validLoginReqBody)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if p.gotLogin.Email != emailValue || p.gotLogin.Password != passwordValue {
		t.Fatalf("provider received wrong credentials: %+v", p.gotLogin)
	}

	access := findCookie(w, "access_token")
	if access == nil || access.Value != accessTokenValue {
		t.Fatalf("expected access_token cookie, got %+v", access)
	}

	refresh := findCookie(w, "refresh_token")
	if refresh == nil || refresh.Value != refreshTokenValue {
		t.Fatalf("expected refresh_token cookie, got %+v", refresh)
	}

	assertSessionCookie(t, w, "access_token", false)
	assertSessionCookie(t, w, "refresh_token", false)
}

func TestLogin_SecureCookies(t *testing.T) {
	p := &stubProvider{loginToken: &auth.Token{
		AccessToken:  accessTokenValue,
		RefreshToken: refreshTokenValue,
		ExpiresIn:    expiresIn,
	}}
	r := newRouterWithSecurity(p, true)

	w := do(r, http.MethodPost, "/auth/login", validLoginReqBody)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}

	assertSessionCookie(t, w, "access_token", true)
	assertSessionCookie(t, w, "refresh_token", true)
}

func TestLogin_InvalidBody(t *testing.T) {
	p := &stubProvider{}
	r := newRouter(p)

	w := do(r, http.MethodPost, "/auth/login", invalidLoginReqBody)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if (p.gotLogin != auth.Login{}) {
		t.Fatalf("provider should not have been called, got %+v", p.gotLogin)
	}
}

func TestLogin_ProviderErrors(t *testing.T) {
	cases := []struct {
		name           string
		err            error
		expectedCode   int
		expectedErrMsg string
	}{
		{"unauthorized", auth.ErrUnauthorized, http.StatusUnauthorized, "invalid email or password"},
		{"unreachable", auth.ErrUnreachable, http.StatusBadGateway, "auth service unreachable"},
		{"bad response", auth.ErrBadResponse, http.StatusInternalServerError, "auth request failed"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRouter(&stubProvider{loginErr: tc.err})
			w := do(r, http.MethodPost, "/auth/login", validLoginReqBody)
			if w.Code != tc.expectedCode {
				t.Fatalf("expected %d, got %d", tc.expectedCode, w.Code)
			}
			assertErrorMessage(t, w, tc.expectedErrMsg)
		})
	}
}

func TestRefresh_MissingCookie(t *testing.T) {
	p := &stubProvider{}
	r := newRouter(p)

	w := do(r, http.MethodPost, "/auth/refresh", "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if p.gotRefreshToken != "" {
		t.Fatal("provider should not be called without a refresh token")
	}
}

func TestRefresh_Success(t *testing.T) {
	p := &stubProvider{refreshToken: &auth.Token{
		AccessToken:  accessTokenValue,
		RefreshToken: refreshTokenValue,
		ExpiresIn:    expiresIn,
	}}
	r := newRouter(p)

	w := do(r, http.MethodPost, "/auth/refresh", "",
		&http.Cookie{Name: "refresh_token", Value: "oldRefresh"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if p.gotRefreshToken != "oldRefresh" {
		t.Fatalf("provider got wrong refresh token: %q", p.gotRefreshToken)
	}
	if c := findCookie(w, "access_token"); c == nil || c.Value != accessTokenValue {
		t.Fatalf("expected rotated access_token, got %+v", c)
	}
	if c := findCookie(w, "refresh_token"); c == nil || c.Value != refreshTokenValue {
		t.Fatalf("expected rotated refresh_token, got %+v", c)
	}
}

func TestRefresh_InvalidTokenClearsCookies(t *testing.T) {
	p := &stubProvider{refreshErr: auth.ErrUnauthorized}
	r := newRouter(p)

	w := do(r, http.MethodPost, "/auth/refresh", "",
		&http.Cookie{Name: "refresh_token", Value: refreshTokenValue})

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	assertErrorMessage(t, w, "could not refresh session")

	assertCleared(t, w, "access_token")
	assertCleared(t, w, "refresh_token")
}

func TestLogout_WithToken(t *testing.T) {
	p := &stubProvider{}
	r := newRouter(p)

	w := do(r, http.MethodPost, "/auth/logout", "",
		&http.Cookie{Name: "access_token", Value: accessTokenValue})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !p.logoutCalled || p.gotLogoutToken != accessTokenValue {
		t.Fatalf(
			"expected provider.Logout with the access token, called=%v token=%q",
			p.logoutCalled,
			p.gotLogoutToken,
		)
	}

	assertCleared(t, w, "access_token")
	assertCleared(t, w, "refresh_token")
}

func TestLogout_WithoutToken(t *testing.T) {
	p := &stubProvider{}
	r := newRouter(p)

	w := do(r, http.MethodPost, "/auth/logout", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if p.logoutCalled {
		t.Fatal("provider.Logout should not be called without an access token")
	}

	assertCleared(t, w, "access_token")
	assertCleared(t, w, "refresh_token")
}

//go:build !integration

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MadalinGOIAN/food-stock/internal/api/auth"
	"github.com/MadalinGOIAN/food-stock/internal/middleware"
	"github.com/gin-gonic/gin"
)

const (
	testUserId = "user123"

	accessTokenValue  = "access"
	refreshTokenValue = "refresh"
	expiresIn         = 3600

	validSignupReqBody        = `{"email":"user@example.com","password":"password","name":"User"}`
	validLoginReqBody         = `{"email":"user@example.com","password":"password"}`
	validDeleteAccountReqBody = `{"password":"password"}`
	invalidReqBody            = `{"email":"user@example.com"}`
	emptyReqBody              = "{}"

	emailValue    = "user@example.com"
	passwordValue = "password"
	nameValue     = "User"
)

type stubProvider struct {
	signupErr      error
	receivedSignup auth.Signup

	loginToken *auth.Token
	loginErr   error

	refreshToken *auth.Token
	refreshErr   error

	logoutErr error

	receivedLogin        auth.Login
	receivedRefreshToken string
	receivedLogoutToken  string
	logoutCalled         bool
}

func (s *stubProvider) Signup(_ context.Context, signup auth.Signup) error {
	s.receivedSignup = signup
	return s.signupErr
}

func (s *stubProvider) Login(_ context.Context, login auth.Login) (*auth.Token, error) {
	s.receivedLogin = login
	return s.loginToken, s.loginErr
}

func (s *stubProvider) Refresh(_ context.Context, refreshToken string) (*auth.Token, error) {
	s.receivedRefreshToken = refreshToken
	return s.refreshToken, s.refreshErr
}

func (s *stubProvider) Logout(_ context.Context, accessToken string) error {
	s.logoutCalled = true
	s.receivedLogoutToken = accessToken
	return s.logoutErr
}

type stubStorage struct {
	active      bool
	isActiveErr error

	deactivateErr        error
	deactivated          bool
	receivedDeactivateID string
}

func (s *stubStorage) IsActive(_ context.Context, _ string) (bool, error) {
	return s.active, s.isActiveErr
}

func (s *stubStorage) Deactivate(_ context.Context, userID string) error {
	s.deactivated = true
	s.receivedDeactivateID = userID
	return s.deactivateErr
}

func stubAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ContextUserId, testUserId)
		c.Set(middleware.ContextUserEmail, emailValue)
		c.Next()
	}
}

func newEngine(p auth.Provider, s auth.Storage, secure bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := auth.NewHandler(p, s, secure)
	auth.Routes(r.Group("/auth"), h, stubAuthMiddleware())
	return r
}

func newRouterWithSecurity(p auth.Provider, secure bool) *gin.Engine {
	return newEngine(p, &stubStorage{active: true}, secure)
}

func newRouter(p auth.Provider) *gin.Engine {
	return newEngine(p, &stubStorage{active: true}, false)
}

func newRouterWithStorage(p auth.Provider, s auth.Storage) *gin.Engine {
	return newEngine(p, s, false)
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
		t.Fatalf("expected %q cookie to be present (cleared), received none", name)
	}
	if c.Value != "" || c.MaxAge >= 0 {
		t.Fatalf("expected %q cookie to be cleared, received value=%q maxAge=%d", name, c.Value, c.MaxAge)
	}
}

func assertSessionCookie(t *testing.T, w *httptest.ResponseRecorder, name string, expectedSecure bool) {
	t.Helper()

	c := findCookie(w, name)
	if c == nil {
		t.Fatalf("expected %q cookie, received none", name)
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
		t.Fatalf("expected error message %q, received %q", expected, body.Error)
	}
}

func TestSignup_Success(t *testing.T) {
	p := &stubProvider{}
	r := newRouter(p)

	w := do(r, http.MethodPost, "/auth/signup", validSignupReqBody)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, received %d (%s)", w.Code, w.Body.String())
	}
	if p.receivedSignup.Email != emailValue || p.receivedSignup.Name != nameValue {
		t.Fatalf("provider received wrong registration: %+v", p.receivedSignup)
	}
}

func TestSignup_InvalidBody(t *testing.T) {
	p := &stubProvider{}
	r := newRouter(p)

	w := do(r, http.MethodPost, "/auth/signup", invalidReqBody)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, received %d", w.Code)
	}
	if (p.receivedSignup != auth.Signup{}) {
		t.Fatalf("provider should not have been called, received  %+v", p.receivedSignup)
	}
}

func TestSignup_Conflict(t *testing.T) {
	r := newRouter(&stubProvider{signupErr: auth.ErrConflict})

	w := do(r, http.MethodPost, "/auth/signup", validSignupReqBody)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, received %d", w.Code)
	}
	assertErrorMessage(t, w, "email already registered")
}

func TestSignup_RateLimited(t *testing.T) {
	r := newRouter(&stubProvider{signupErr: auth.ErrRateLimited})

	w := do(r, http.MethodPost, "/auth/signup", validSignupReqBody)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, received %d", w.Code)
	}
	assertErrorMessage(t, w, "server too busy; try again later")
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
		t.Fatalf("expected 200, received %d (%s)", w.Code, w.Body.String())
	}
	if p.receivedLogin.Email != emailValue || p.receivedLogin.Password != passwordValue {
		t.Fatalf("provider received wrong credentials: %+v", p.receivedLogin)
	}

	access := findCookie(w, "access_token")
	if access == nil || access.Value != accessTokenValue {
		t.Fatalf("expected access_token cookie, received %+v", access)
	}

	refresh := findCookie(w, "refresh_token")
	if refresh == nil || refresh.Value != refreshTokenValue {
		t.Fatalf("expected refresh_token cookie, received %+v", refresh)
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
		t.Fatalf("expected 200, received %d (%s)", w.Code, w.Body.String())
	}

	assertSessionCookie(t, w, "access_token", true)
	assertSessionCookie(t, w, "refresh_token", true)
}

func TestLogin_StorageError(t *testing.T) {
	p := &stubProvider{loginToken: &auth.Token{
		AccessToken:  accessTokenValue,
		RefreshToken: refreshTokenValue,
		ExpiresIn:    expiresIn,
		User:         auth.User{Id: testUserId},
	}}
	r := newRouterWithStorage(p, &stubStorage{isActiveErr: errors.New("DB down")})

	w := do(r, http.MethodPost, "/auth/login", validLoginReqBody)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, received %d (%s)", w.Code, w.Body.String())
	}
	assertErrorMessage(t, w, "auth request failed")
	if c := findCookie(w, "access_token"); c != nil {
		t.Fatalf("no cookie expected on storage error, received %+v", c)
	}
}

func TestLogin_InvalidBody(t *testing.T) {
	p := &stubProvider{}
	r := newRouter(p)

	w := do(r, http.MethodPost, "/auth/login", invalidReqBody)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, received %d", w.Code)
	}
	if (p.receivedLogin != auth.Login{}) {
		t.Fatalf("provider should not have been called, received %+v", p.receivedLogin)
	}
}

func TestLogin_InactiveAccount(t *testing.T) {
	p := &stubProvider{loginToken: &auth.Token{
		AccessToken:  accessTokenValue,
		RefreshToken: refreshTokenValue,
		ExpiresIn:    expiresIn,
		User:         auth.User{Id: testUserId},
	}}
	r := newRouterWithStorage(p, &stubStorage{active: false})

	w := do(r, http.MethodPost, "/auth/login", validLoginReqBody)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, received %d (%s)", w.Code, w.Body.String())
	}

	assertErrorMessage(t, w, "account has been deleted")

	if c := findCookie(w, "access_token"); c != nil {
		t.Fatalf("no access_token cookie expected for inactive account, received %+v", c)
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
		{"rate limited", auth.ErrRateLimited, http.StatusTooManyRequests, "server too busy; try again later"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRouter(&stubProvider{loginErr: tc.err})
			w := do(r, http.MethodPost, "/auth/login", validLoginReqBody)
			if w.Code != tc.expectedCode {
				t.Fatalf("expected %d, received %d", tc.expectedCode, w.Code)
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
		t.Fatalf("expected 401, received %d", w.Code)
	}
	if p.receivedRefreshToken != "" {
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
		t.Fatalf("expected 200, received %d (%s)", w.Code, w.Body.String())
	}
	if p.receivedRefreshToken != "oldRefresh" {
		t.Fatalf("provider received wrong refresh token: %q", p.receivedRefreshToken)
	}
	if c := findCookie(w, "access_token"); c == nil || c.Value != accessTokenValue {
		t.Fatalf("expected rotated access_token, received %+v", c)
	}
	if c := findCookie(w, "refresh_token"); c == nil || c.Value != refreshTokenValue {
		t.Fatalf("expected rotated refresh_token, received %+v", c)
	}
}

func TestRefresh_InvalidTokenClearsCookies(t *testing.T) {
	p := &stubProvider{refreshErr: auth.ErrUnauthorized}
	r := newRouter(p)

	w := do(r, http.MethodPost, "/auth/refresh", "",
		&http.Cookie{Name: "refresh_token", Value: refreshTokenValue})

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, received %d", w.Code)
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
		t.Fatalf("expected 200, received %d", w.Code)
	}
	if !p.logoutCalled || p.receivedLogoutToken != accessTokenValue {
		t.Fatalf(
			"expected provider.Logout with the access token, called=%v token=%q",
			p.logoutCalled,
			p.receivedLogoutToken,
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
		t.Fatalf("expected 200, received %d", w.Code)
	}
	if p.logoutCalled {
		t.Fatal("provider.Logout should not be called without an access token")
	}

	assertCleared(t, w, "access_token")
	assertCleared(t, w, "refresh_token")
}

func TestDeleteAccount_Success(t *testing.T) {
	p := &stubProvider{loginToken: &auth.Token{AccessToken: accessTokenValue}}
	s := &stubStorage{active: true}
	r := newRouterWithStorage(p, s)

	w := do(
		r,
		http.MethodDelete,
		"/auth/account",
		validDeleteAccountReqBody,
		&http.Cookie{Name: "access_token", Value: accessTokenValue},
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, received %d (%s)", w.Code, w.Body.String())
	}
	if p.receivedLogin.Email != emailValue || p.receivedLogin.Password != passwordValue {
		t.Fatalf("re-auth used wrong credentials: %+v", p.receivedLogin)
	}
	if !s.deactivated || s.receivedDeactivateID != testUserId {
		t.Fatalf(
			"expected deactivate for %q, received deactivated=%v id=%q",
			testUserId,
			s.deactivated,
			s.receivedDeactivateID,
		)
	}
	if !p.logoutCalled || p.receivedLogoutToken != accessTokenValue {
		t.Fatalf("expected session revoke, called=%v token=%q", p.logoutCalled, p.receivedLogoutToken)
	}

	assertCleared(t, w, "access_token")
	assertCleared(t, w, "refresh_token")
}

func TestDeleteAccount_WrongPassword(t *testing.T) {
	p := &stubProvider{loginErr: auth.ErrUnauthorized}
	s := &stubStorage{active: true}
	r := newRouterWithStorage(p, s)

	w := do(
		r,
		http.MethodDelete,
		"/auth/account",
		validDeleteAccountReqBody,
		&http.Cookie{Name: "access_token", Value: accessTokenValue},
	)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, received %d", w.Code)
	}
	assertErrorMessage(t, w, "invalid password")
	if s.deactivated {
		t.Fatal("account must not be deactivated when the password is wrong")
	}
}

func TestDeleteAccount_InvalidBody(t *testing.T) {
	p := &stubProvider{}
	s := &stubStorage{active: true}
	r := newRouterWithStorage(p, s)

	w := do(
		r,
		http.MethodDelete,
		"/auth/account",
		emptyReqBody,
		&http.Cookie{Name: "access_token", Value: accessTokenValue},
	)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, received %d", w.Code)
	}
	if (p.receivedLogin != auth.Login{}) || s.deactivated {
		t.Fatal("provider/storage methods must not be called on invalid body")
	}
}

func TestDeleteAccount_DeactivateError(t *testing.T) {
	p := &stubProvider{loginToken: &auth.Token{AccessToken: accessTokenValue}}
	s := &stubStorage{active: true, deactivateErr: errors.New("DB down")}
	r := newRouterWithStorage(p, s)

	w := do(
		r,
		http.MethodDelete,
		"/auth/account",
		validDeleteAccountReqBody,
		&http.Cookie{Name: "access_token", Value: accessTokenValue},
	)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, received %d (%s)", w.Code, w.Body.String())
	}
	assertErrorMessage(t, w, "could not delete account")
	if p.logoutCalled {
		t.Fatal("session must not be revoked when deactivate failed")
	}
}

func TestDeleteAccount_RateLimited(t *testing.T) {
	p := &stubProvider{loginErr: auth.ErrRateLimited}
	s := &stubStorage{active: true}
	r := newRouterWithStorage(p, s)

	w := do(
		r,
		http.MethodDelete,
		"/auth/account",
		validDeleteAccountReqBody,
		&http.Cookie{Name: "access_token", Value: accessTokenValue},
	)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, received %d", w.Code)
	}
	assertErrorMessage(t, w, "server too busy; try again later")
	if s.deactivated {
		t.Fatal("account must not be deactivated when re-auth is rate limited")
	}
}

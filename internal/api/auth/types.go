package auth

import "errors"

var (
	ErrConflict     = errors.New("auth: already registered")
	ErrInvalid      = errors.New("auth: invalid registration")
	ErrUnauthorized = errors.New("auth: unauthorized")
	ErrUnreachable  = errors.New("auth: provider unreachable")
	ErrBadResponse  = errors.New("auth: bad provider response")
)

type Signup struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name" binding:"required"`
}

type Login struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AccountDeletion struct {
	Password string `json:"password" binding:"required"`
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         User   `json:"user"`
}

type User struct {
	Id string `json:"id"`
}

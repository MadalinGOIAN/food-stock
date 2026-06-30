package auth

type Login struct {
	Email		string `json:"email" binding:"required,email"`
	Password 	string `json:"password" binding:"required"`
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

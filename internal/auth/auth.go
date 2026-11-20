package auth

import (
	"net/http"

	"elastic-ui/internal/config"
)

type Auth struct {
	Method   config.AuthMethod
	Username string
	Password string
	APIKey   string
}

func (a Auth) Apply(req *http.Request) {
	if a.Method == config.AuthAPIKey {
		req.Header.Set("Authorization", "ApiKey "+a.APIKey)
		return
	}
	req.SetBasicAuth(a.Username, a.Password)
}

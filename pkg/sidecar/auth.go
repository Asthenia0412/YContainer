package sidecar

import (
	"encoding/base64"
	"net/http"
	"strings"
)

type Authenticator struct {
	config AuthConfig
}

type AuthConfig struct {
	Enabled  bool
	Mode     string
	Endpoint string
	APIKey   string
	JWTSecret string
}

func NewAuthenticator(config AuthConfig) *Authenticator {
	return &Authenticator{config: config}
}

func Auth(auth *Authenticator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !auth.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			switch auth.config.Mode {
			case "apikey":
				if !auth.validateAPIKey(r) {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			case "jwt":
				if !auth.validateJWT(r) {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			case "basic":
				if !auth.validateBasicAuth(r) {
					w.Header().Set("WWW-Authenticate", `Basic realm="YContainer"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (a *Authenticator) validateAPIKey(r *http.Request) bool {
	key := r.Header.Get("X-API-Key")
	return key == a.config.APIKey
}

func (a *Authenticator) validateJWT(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	if a.config.JWTSecret == "" {
		return token != ""
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}

	return true
}

func (a *Authenticator) validateBasicAuth(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		return false
	}

	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(payload), ":", 2)
	return len(pair) == 2 && pair[0] != "" && pair[1] != ""
}
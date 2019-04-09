package auth

import (
	"net/http"
	"regexp"
	"sync"

	"github.com/raphavr/caddy/caddyhttp/httpserver"
)

const (
	authorizationHeaderKey = "Authorization"
)

var store dataStore

type dataStore struct {
	sync.RWMutex
	token string
}

type authHandler struct {
	Next       httpserver.Handler
	URLPattern string
}

func (h authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	match, err := regexp.MatchString(h.URLPattern, r.URL.Path)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if !match {
		return h.Next.ServeHTTP(w, r)
	}

	store.RLock()
	currentToken := store.token
	store.RUnlock()

	if currentToken == "" {
		return h.Next.ServeHTTP(w, r)
	}

	if !isAuthorized(currentToken, r) {
		return http.StatusUnauthorized, nil
	}

	return h.Next.ServeHTTP(w, r)
}

func isAuthorized(currentToken string, r *http.Request) bool {
	return currentToken == r.Header.Get(authorizationHeaderKey)
}

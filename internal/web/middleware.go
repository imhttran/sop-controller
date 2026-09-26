package web

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"
)

type ctxKey int

const csrfKey ctxKey = iota

const (
	csrfCookie   = "sop_ctrl_csrf"
	accessCookie = "sop_ctrl_access"
)

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// csrfToken returns the per-session CSRF token attached by csrfProtect.
func csrfToken(r *http.Request) string {
	if v, ok := r.Context().Value(csrfKey).(string); ok {
		return v
	}
	return ""
}

// csrfProtect issues a CSRF cookie and rejects state-changing requests whose
// token doesn't match (FR: state-changing requests use POST + CSRF).
func csrfProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if c, err := r.Cookie(csrfCookie); err == nil {
			token = c.Value
		}
		if token == "" {
			token = newToken()
			http.SetCookie(w, &http.Cookie{
				Name: csrfCookie, Value: token, Path: "/",
				HttpOnly: true, SameSite: http.SameSiteStrictMode,
			})
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			got := r.FormValue("csrf")
			if got == "" {
				got = r.Header.Get("X-CSRF-Token")
			}
			if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
				http.Error(w, "invalid CSRF token", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), csrfKey, token)))
	})
}

// accessToken gates every route except health and static assets (network mode).
func accessToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/static/") {
				next.ServeHTTP(w, r)
				return
			}
			if q := r.URL.Query().Get("token"); q != "" && subtle.ConstantTimeCompare([]byte(q), []byte(token)) == 1 {
				http.SetCookie(w, &http.Cookie{
					Name: accessCookie, Value: token, Path: "/",
					HttpOnly: true, SameSite: http.SameSiteLaxMode,
				})
				next.ServeHTTP(w, r)
				return
			}
			provided := r.Header.Get("X-Access-Token")
			if c, err := r.Cookie(accessCookie); err == nil {
				provided = c.Value
			}
			if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
				http.Error(w, "access token required", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

package httpadapter

import (
	"net/http"
	"time"
)

const (
	refreshCookieName = "cuadra_refresh"
	// refreshCookiePath limits the cookie to the auth endpoints: the browser
	// never sends the refresh token to the rest of the API.
	refreshCookiePath = "/api/v1/auth"
)

// refreshCookie returns the Set-Cookie value carrying a refresh token.
// HttpOnly keeps it from scripts; Secure from plain HTTP (browsers treat
// http://localhost as secure, so it works in development); SameSite=Strict
// keeps other sites from sending it (CSRF on refresh and logout).
func refreshCookie(token string, expiresAt, now time.Time) string {
	return (&http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		MaxAge:   int(expiresAt.Sub(now).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}).String()
}

// clearedRefreshCookie returns the Set-Cookie value that deletes the cookie.
func clearedRefreshCookie() string {
	return (&http.Cookie{
		Name:     refreshCookieName,
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}).String()
}

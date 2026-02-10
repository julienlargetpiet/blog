package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
    "strings"
)

const sessionCookie = "admin_session"

func sign(secret, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func setSession(w http.ResponseWriter, secret string) {
	val := "admin"
	sig := sign(secret, val)

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    val + "|" + sig,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookie,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

func isAuthenticated(r *http.Request, secret string) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}

	parts := strings.SplitN(c.Value, "|", 2)
	if len(parts) != 2 {
		return false
	}

	val, sig := parts[0], parts[1]
	return hmac.Equal([]byte(sig), []byte(sign(secret, val)))
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAuthenticated(r, s.AdminPass) {
			next(w, r)
			return
		}

		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
	}
}





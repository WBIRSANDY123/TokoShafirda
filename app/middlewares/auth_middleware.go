package middlewares

import (
	"net/http"

	"github.com/gieart87/gotoko/app/core/session/auth"
	"github.com/gieart87/gotoko/app/core/session/flash"
)

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auth.IsLoggedIn(r) {
			flash.SetFlash(w, r, "error", "Anda perlu login!")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		// pengecekan login user
		next.ServeHTTP(w, r)
	})
}

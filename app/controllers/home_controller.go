package controllers

import (
	"net/http"

	"github.com/gieart87/gotoko/app/core/session/auth"
	"github.com/unrolled/render"
)

func (server *Server) Home(w http.ResponseWriter, r *http.Request) {
	// Mengatur konfigurasi untuk rendering template HTML.
	render := render.New(render.Options{
		Layout:     "layout",                   // Menentukan file layout utama untuk halaman.
		Extensions: []string{".html", ".tmpl"}, // Menentukan ekstensi file template yang didukung.
	})

	// Mendapatkan data pengguna yang sedang login.
	user := auth.CurrentUser(server.DB, w, r)

	// Merender halaman "home" dengan status HTTP 200.
	_ = render.HTML(w, http.StatusOK, "home", map[string]interface{}{
		"user": user, // Menyisipkan data pengguna ke dalam template.
	})
}

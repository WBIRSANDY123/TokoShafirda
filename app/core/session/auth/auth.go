package auth

import (
	"net/http"
	"os"

	"github.com/gieart87/gotoko/app/models"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Inisialisasi penyimpanan sesi menggunakan kunci dari variabel lingkungan
var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
var sessionUser = "user-session" // Sesi untuk pengguna yang login

func GetSessionUser(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, sessionUser)
}

// IsLoggedIn memeriksa apakah pengguna sudah login dengan melihat sesi "sessionUser".
func IsLoggedIn(r *http.Request) bool {
	// Mendapatkan sesi pengguna dari store dengan nama "sessionUser".
	session, _ := store.Get(r, sessionUser)
	if session.Values["id"] == nil { // Jika "id" tidak ada dalam sesi, artinya pengguna belum login.
		return false
	}

	// Jika sesi ada dan "id" ditemukan, artinya pengguna sudah login.
	return true
}

// ComparePassword membandingkan password yang diberikan dengan password yang sudah di-hash.
func ComparePassword(password string, hashedPassword string) bool {
	// Membandingkan password yang di-hash dengan password yang diberikan.
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

// MakePassword meng-hash password yang diberikan dan mengembalikan hasilnya.
func MakePassword(password string) (string, error) {
	// Meng-hash password menggunakan bcrypt dengan biaya default.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	// Mengembalikan password yang di-hash dan error jika ada.
	return string(hashedPassword), err
}

// CurrentUser mengembalikan objek pengguna yang sedang login berdasarkan sesi.
func CurrentUser(db *gorm.DB, w http.ResponseWriter, r *http.Request) *models.User {
	// Memeriksa apakah pengguna sudah login.
	if !IsLoggedIn(r) {
		return nil
	}

	// Mendapatkan sesi pengguna dari store dengan nama "sessionUser".
	session, _ := store.Get(r, sessionUser)

	// Membuat objek User dan mencari pengguna berdasarkan ID yang ada di sesi.
	userModel := models.User{}
	user, err := userModel.FindByID(db, session.Values["id"].(string))
	if err != nil { // Jika terjadi error saat mencari pengguna berdasarkan ID.
		session.Values["id"] = nil // Menghapus ID pengguna dari sesi.
		session.Save(r, w)         // Menyimpan perubahan sesi.
		return nil
	}

	// Mengembalikan objek pengguna jika ditemukan.
	return user
}

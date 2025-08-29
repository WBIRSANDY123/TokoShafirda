package flash

import (
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

// Inisialisasi penyimpanan sesi menggunakan kunci dari variabel lingkungan
var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

// Nama sesi yang digunakan dalam aplikasi
var sessionFlash = "flash-session" // Sesi untuk pesan flash

// SetFlash menyimpan pesan flash ke dalam sesi dengan nama tertentu.
func SetFlash(w http.ResponseWriter, r *http.Request, name string, value string) {
	// Mendapatkan sesi yang ada dari store dengan menggunakan nama sesi "sessionFlash".
	session, err := store.Get(r, sessionFlash)
	if err != nil { // Jika terjadi error saat mengambil sesi.
		http.Error(w, err.Error(), http.StatusInternalServerError) // Menampilkan error 500 (Internal Server Error).
		return
	}

	// Menambahkan flash message dengan nama 'name' dan value 'value' ke sesi.
	session.AddFlash(value, name)
	// Menyimpan perubahan sesi ke dalam response writer.
	session.Save(r, w)
}

// GetFlash mengambil pesan flash yang tersimpan berdasarkan nama tertentu.
func GetFlash(w http.ResponseWriter, r *http.Request, name string) []string {
	// Mendapatkan sesi yang ada dari store dengan menggunakan nama sesi "sessionFlash".
	session, err := store.Get(r, sessionFlash)
	if err != nil { // Jika terjadi error saat mengambil sesi.
		http.Error(w, err.Error(), http.StatusInternalServerError) // Menampilkan error 500 (Internal Server Error).
		return nil
	}

	// Mengambil semua flash messages berdasarkan nama tertentu.
	fm := session.Flashes(name)
	if len(fm) < 0 { // Jika tidak ada flash messages ditemukan.
		return nil
	}

	// Menyimpan perubahan sesi ke dalam response writer.
	session.Save(r, w)
	var flashes []string
	// Mengonversi setiap flash message yang ada ke dalam slice string.
	for _, fl := range fm {
		flashes = append(flashes, fl.(string))
	}

	// Mengembalikan slice berisi pesan flash yang ditemukan.
	return flashes
}

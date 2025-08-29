package controllers

import (
	"net/http"

	"github.com/gieart87/gotoko/app/consts"
	"github.com/gieart87/gotoko/app/middlewares"
	"github.com/gorilla/mux"
)

func (server *Server) initializeRoutes() {
	server.Router = mux.NewRouter() // Membuat router baru untuk mengatur rute dalam aplikasi.

	// Menangani halaman utama.
	server.Router.HandleFunc("/", server.Home).Methods("GET")

	// Menangani halaman login.
	server.Router.HandleFunc("/login", server.Login).Methods("GET")
	// Memproses data login yang dikirimkan.
	server.Router.HandleFunc("/login", server.DoLogin).Methods("POST")

	// Menampilkan form registrasi pengguna.
	server.Router.HandleFunc("/register", server.Register).Methods("GET")
	// Memproses data registrasi pengguna.
	server.Router.HandleFunc("/register", server.DoRegister).Methods("POST")

	// Menangani proses logout pengguna.
	server.Router.HandleFunc("/logout", server.Logout).Methods("GET")

	// Menampilkan daftar produk.
	server.Router.HandleFunc("/products", server.Products).Methods("GET")
	// API untuk autocomplete search products
	server.Router.HandleFunc("/api/products/search", server.SearchProductsAPI).Methods("GET")
	// Menampilkan detail produk berdasarkan slug unik.
	server.Router.HandleFunc("/products/{slug}", server.GetProductBySlug).Methods("GET")

	// Menangani pemeriksaan status pengiriman berdasarkan nomor resi.
	server.Router.HandleFunc("/checkAWB", server.checkAWB).Methods("GET")
	// Memproses cek resi yang dikirimkan pengguna.
	server.Router.HandleFunc("/cek-resi", server.CekResiHandler).Methods("POST")

	// Menampilkan isi keranjang belanja (memerlukan login).
	server.Router.HandleFunc("/carts", middlewares.AuthMiddleware(server.GetCart)).Methods("GET")
	// Menambahkan item ke keranjang belanja (memerlukan login).
	server.Router.HandleFunc("/carts", middlewares.AuthMiddleware(server.AddItemToCart)).Methods("POST")
	// Memperbarui jumlah item dalam keranjang (memerlukan login).
	server.Router.HandleFunc("/carts/update", middlewares.AuthMiddleware(server.UpdateCart)).Methods("POST")

	// Menghitung biaya pengiriman menggunakan API Biteship (memerlukan login).
	server.Router.HandleFunc("/carts/calculate-shipping", middlewares.AuthMiddleware(server.CalculateShippingBiteship)).Methods("POST")
	// Menerapkan pilihan pengiriman ke keranjang (memerlukan login).
	server.Router.HandleFunc("/carts/apply-shipping", middlewares.AuthMiddleware(server.ApplyShipping)).Methods("POST")
	// Menghapus item dari keranjang berdasarkan ID (memerlukan login).
	server.Router.HandleFunc("/carts/remove/{id}", middlewares.AuthMiddleware(server.RemoveItemByID)).Methods("GET")

	// Menangani proses checkout dengan autentikasi pengguna.
	server.Router.HandleFunc("/orders/checkout", middlewares.AuthMiddleware(server.Checkout)).Methods("POST")
	// Menampilkan detail pesanan berdasarkan ID dengan autentikasi.
	server.Router.HandleFunc("/orders/{id}", middlewares.AuthMiddleware(server.ShowOrder)).Methods("GET")

	// Memproses notifikasi pembayaran dari Midtrans untuk update status pembayaran.
	server.Router.HandleFunc("/payment/notification", server.MidtransNotification).Methods("POST")

	server.Router.HandleFunc("/payment/test", server.PaymentTest).Methods("GET", "POST")

	server.Router.HandleFunc("/admin/dashboard", middlewares.AuthMiddleware(middlewares.RoleMiddleware(server.AdminDashboard, server.DB, consts.RoleAdmin))).Methods("GET")

	// Menyediakan file statis dari direktori './assets/' yang dapat diakses melalui '/public/'.
	staticFileDirectory := http.Dir("./assets/")
	staticFileHandler := http.StripPrefix("/public/", http.FileServer(staticFileDirectory))
	server.Router.PathPrefix("/public/").Handler(staticFileHandler).Methods("GET")
}

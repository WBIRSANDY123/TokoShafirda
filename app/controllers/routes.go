package controllers

import (
	"net/http"

	"github.com/gieart87/gotoko/app/consts"
	"github.com/gieart87/gotoko/app/middlewares"
	"github.com/gorilla/mux"
)

func (server *Server) initializeRoutes() {
	server.Router = mux.NewRouter()
	server.Router.HandleFunc("/", server.Home).Methods("GET")

	server.Router.HandleFunc("/login", server.Login).Methods("GET")
	server.Router.HandleFunc("/login", server.DoLogin).Methods("POST")

	server.Router.HandleFunc("/register", server.Register).Methods("GET")
	server.Router.HandleFunc("/register", server.DoRegister).Methods("POST")

	server.Router.HandleFunc("/logout", server.Logout).Methods("GET")

	server.Router.HandleFunc("/products", server.Products).Methods("GET")
	server.Router.HandleFunc("/api/products/search", server.SearchProductsAPI).Methods("GET")
	server.Router.HandleFunc("/products/{slug}", server.GetProductBySlug).Methods("GET")

	server.Router.HandleFunc("/checkAWB", server.checkAWB).Methods("GET")
	server.Router.HandleFunc("/cek-resi", server.CekResiHandler).Methods("POST")

	server.Router.HandleFunc("/carts", middlewares.AuthMiddleware(server.GetCart)).Methods("GET")
	server.Router.HandleFunc("/carts", middlewares.AuthMiddleware(server.AddItemToCart)).Methods("POST")
	server.Router.HandleFunc("/carts/update", middlewares.AuthMiddleware(server.UpdateCart)).Methods("POST")

	server.Router.HandleFunc("/carts/calculate-shipping", middlewares.AuthMiddleware(server.CalculateShippingBiteship)).Methods("POST")
	server.Router.HandleFunc("/carts/apply-shipping", middlewares.AuthMiddleware(server.ApplyShipping)).Methods("POST")
	server.Router.HandleFunc("/carts/remove/{id}", middlewares.AuthMiddleware(server.RemoveItemByID)).Methods("GET")

	server.Router.HandleFunc("/orders/checkout", middlewares.AuthMiddleware(server.Checkout)).Methods("POST")
	server.Router.HandleFunc("/orders/{id}", middlewares.AuthMiddleware(server.ShowOrder)).Methods("GET")

	server.Router.HandleFunc("/payment/notification", middlewares.CORSMiddleware(server.MidtransNotification)).Methods("POST", "OPTIONS")
	server.Router.HandleFunc("/payment/test", middlewares.CORSMiddleware(server.PaymentTest)).Methods("GET", "POST")
	server.Router.HandleFunc("/admin/dashboard", middlewares.AuthMiddleware(middlewares.RoleMiddleware(server.AdminDashboard, server.DB, consts.RoleAdmin))).Methods("GET")

	staticFileDirectory := http.Dir("./assets/")
	staticFileHandler := http.StripPrefix("/public/", http.FileServer(staticFileDirectory))
	server.Router.PathPrefix("/public/").Handler(staticFileHandler).Methods("GET")
}

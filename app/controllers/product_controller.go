package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/gieart87/gotoko/app/core/session/auth"
	"github.com/gieart87/gotoko/app/core/session/flash"
	"github.com/gieart87/gotoko/app/models"

	"github.com/unrolled/render"
)

func (server *Server) Products(w http.ResponseWriter, r *http.Request) {
	// Inisialisasi render dengan pengaturan layout dan ekstensi file
	render := render.New(render.Options{
		Layout:     "layout",
		Extensions: []string{".html", ".tmpl"},
	})

	// Mendapatkan query parameter dari URL
	q := r.URL.Query()
	searchQuery := q.Get("query")

	// Mendapatkan nilai "page" dari query parameter dan mengonversinya menjadi integer
	page, _ := strconv.Atoi(q.Get("page"))
	if page <= 0 {
		// Jika nilai page tidak valid, atur ke 1
		page = 1
	}

	// Menentukan jumlah produk per halaman
	perPage := 100

	// Membuat instance model produk
	productModel := models.Product{}

	var products *[]models.Product
	var totalRows int64
	var err error

	// Jika ada query search, gunakan SearchProducts, jika tidak gunakan GetProducts
	if searchQuery != "" {
		products, totalRows, err = productModel.SearchProducts(server.DB, searchQuery, perPage, page)
	} else {
		products, totalRows, err = productModel.GetProducts(server.DB, perPage, page)
	}

	if err != nil {
		// Mengembalikan jika terjadi kesalahan saat mengambil data
		return
	}

	// Membuat tautan paginasi berdasarkan data yang diperoleh
	pagination, _ := GetPaginationLinks(server.AppConfig, PaginationParams{
		Path:        "products",
		TotalRows:   int32(totalRows),
		PerPage:     int32(perPage),
		CurrentPage: int32(page),
	})

	// Merender halaman produk dengan data yang diperoleh
	_ = render.HTML(w, http.StatusOK, "products", map[string]interface{}{
		"products":    products,
		"pagination":  pagination,
		"user":        auth.CurrentUser(server.DB, w, r),
		"searchQuery": searchQuery,
	})
}

func (server *Server) GetProductBySlug(w http.ResponseWriter, r *http.Request) {
	// Inisialisasi render dengan pengaturan layout dan ekstensi file
	render := render.New(render.Options{
		Layout:     "layout",
		Extensions: []string{".html", ".tmpl"},
	})

	// Mendapatkan variabel dari URL
	vars := mux.Vars(r)

	// Memeriksa apakah slug tidak kosong
	if vars["slug"] == "" {
		// Jika slug kosong, hentikan eksekusi
		return
	}

	// Membuat instance model produk
	productModel := models.Product{}

	// Mencari produk berdasarkan slug di database
	product, err := productModel.FindBySlug(server.DB, vars["slug"])
	if err != nil {
		// Mengembalikan jika terjadi kesalahan saat mencari produk
		return
	}

	// Merender halaman produk dengan data yang diperoleh
	_ = render.HTML(w, http.StatusOK, "product", map[string]interface{}{
		"product": product,
		"success": flash.GetFlash(w, r, "success"),
		"error":   flash.GetFlash(w, r, "error"),
		"user":    auth.CurrentUser(server.DB, w, r),
	})
}

// SearchProductsAPI - API endpoint untuk autocomplete search products
func (server *Server) SearchProductsAPI(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	
	if query == "" || len(query) < 2 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	productModel := models.Product{}
	products, _, err := productModel.SearchProducts(server.DB, query, 10, 1) // Limit 10 hasil
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Search failed"})
		return
	}

	// Format response untuk autocomplete
	type ProductSuggestion struct {
		Name       string `json:"name"`
		Slug       string `json:"slug"`
		Categories string `json:"categories,omitempty"`
		SATUAN1    string `json:"satuan1,omitempty"`
		SATUAN2    string `json:"satuan2,omitempty"`
		SATUAN3    string `json:"satuan3,omitempty"`
		Price      string `json:"price"`
	}

	var suggestions []ProductSuggestion
	if products != nil {
		for _, product := range *products {
			suggestion := ProductSuggestion{
				Name:       product.Name,
				Slug:       product.Slug,
				Categories: product.Categories,
				SATUAN1:    product.SATUAN1,
				SATUAN2:    product.SATUAN2,
				SATUAN3:    product.SATUAN3,
				Price:      product.HJ1.String(),
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(suggestions)
}

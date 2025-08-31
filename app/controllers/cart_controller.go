package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/shopspring/decimal"

	"github.com/gorilla/mux"

	"github.com/unrolled/render"

	"gorm.io/gorm"

	"github.com/google/uuid"

	"github.com/gieart87/gotoko/app/core/session/auth"
	"github.com/gieart87/gotoko/app/core/session/flash"
	"github.com/gieart87/gotoko/app/models"
)

// GetShoppingCartID mengembalikan ID cart yang tersimpan dalam sesi atau membuat ID baru jika tidak ada.
func GetShoppingCartID(w http.ResponseWriter, r *http.Request) string {
	// Mengambil sesi cart dari store dengan nama "sessionShoppingCart".
	session, _ := store.Get(r, sessionShoppingCart)
	// Jika "cart-id" tidak ada dalam sesi, buat ID baru dan simpan ke dalam sesi.
	if session.Values["cart-id"] == nil {
		session.Values["cart-id"] = uuid.New().String() // Membuat ID baru menggunakan UUID.
		_ = session.Save(r, w)                          // Menyimpan perubahan sesi.
	}

	// Mengembalikan ID cart yang ada dalam sesi (atau ID yang baru dibuat).
	return fmt.Sprintf("%v", session.Values["cart-id"])
}

// ClearCart menghapus item dalam keranjang berdasarkan cartID.
func ClearCart(db *gorm.DB, cartID string) error {
	var cart models.Cart

	// Memanggil metode ClearCart dari model Cart untuk menghapus item dari keranjang.
	err := cart.ClearCart(db, cartID)
	if err != nil { // Jika terjadi error saat menghapus cart.
		return err
	}
	// Mengembalikan nil jika berhasil menghapus cart.
	return nil
}

// GetShoppingCart mengembalikan informasi tentang keranjang belanja berdasarkan cartID.
func GetShoppingCart(db *gorm.DB, cartID string) (*models.Cart, error) {
	var cart models.Cart

	// Mencari cart berdasarkan cartID di database.
	existCart, err := cart.GetCart(db, cartID)
	if err != nil { // Jika cart tidak ditemukan, buat cart baru.
		existCart, _ = cart.CreateCart(db, cartID)
	}

	// Menghitung total harga untuk cart yang ada.
	_, _ = existCart.CalculateCart(db, cartID)

	// Mendapatkan kembali cart yang sudah diperbarui.
	updatedCart, _ := cart.GetCart(db, cartID)

	// Menghitung total berat cart.
	totalWeight := 0
	productModel := models.Product{}
	for _, cartItem := range updatedCart.CartItems { // Mengiterasi setiap item dalam cart.
		// Mencari produk berdasarkan ProductID yang ada dalam cart item.
		product, _ := productModel.FindByID(db, cartItem.ProductID)

		// Mendapatkan berat produk dan menghitung berat total per item.
		productWeight, _ := product.Weight.Float64()
		ceilWeight := math.Ceil(productWeight) // Membulatkan berat produk ke atas.

		// Menghitung berat total untuk item tersebut berdasarkan jumlahnya.
		itemWeight := cartItem.Qty * int(ceilWeight)

		// Menambahkan berat item ke total berat cart.
		totalWeight += itemWeight
	}

	// Menyimpan total berat ke dalam cart yang diperbarui.
	updatedCart.TotalWeight = totalWeight

	// Mengembalikan cart yang sudah diperbarui dengan total berat.
	return updatedCart, nil
}

// Fungsi checkAWB untuk memeriksa dan menampilkan halaman cek resi berdasarkan cart yang ada.
func (server *Server) checkAWB(w http.ResponseWriter, r *http.Request) {
	// Membuat objek render baru untuk merender tampilan HTML menggunakan layout dan template.
	render := render.New(render.Options{
		Layout:     "layout",                   // Layout yang digunakan untuk tampilan.
		Extensions: []string{".html", ".tmpl"}, // Ekstensi template yang didukung.
	})

	// Mendeklarasikan variabel cart untuk menyimpan data keranjang belanja.
	var cart *models.Cart

	// Mendapatkan cartID dari sesi pengguna dan mengambil cart berdasarkan cartID.
	cartID := GetShoppingCartID(w, r)
	cart, _ = GetShoppingCart(server.DB, cartID)

	items, _ := GetCartItemsWithImages(server.DB, cartID)

	// Merender halaman HTML dengan template "cek_resi", dan memasukkan data ke dalam template.
	_ = render.HTML(w, http.StatusOK, "cek_resi", map[string]interface{}{
		"cart":    cart,                              // Menyertakan data cart ke dalam template.
		"items":   items,                             // Menyertakan data item dalam cart ke dalam template.
		"success": flash.GetFlash(w, r, "success"),   // Menyertakan pesan flash sukses jika ada.
		"error":   flash.GetFlash(w, r, "error"),     // Menyertakan pesan flash error jika ada.
		"user":    auth.CurrentUser(server.DB, w, r), // Menyertakan data pengguna yang sedang login.
	})
}

// Fungsi Track digunakan untuk merender halaman pelacakan (track) pengiriman berdasarkan keranjang belanja.
func (server *Server) Track(w http.ResponseWriter, r *http.Request) {
	// Membuat objek render baru dengan layout dan template yang ditentukan untuk tampilan HTML.
	render := render.New(render.Options{
		Layout:     "layout",                   // Layout yang digunakan untuk tampilan.
		Extensions: []string{".html", ".tmpl"}, // Ekstensi template yang didukung.
	})

	// Mendeklarasikan variabel cart untuk menyimpan data keranjang belanja.
	var cart *models.Cart

	// Mendapatkan cartID dari sesi pengguna dan mengambil cart berdasarkan cartID.
	cartID := GetShoppingCartID(w, r)
	cart, _ = GetShoppingCart(server.DB, cartID)

	items, _ := GetCartItemsWithImages(server.DB, cartID)

	// Merender halaman HTML dengan template "cek_resi", dan memasukkan data ke dalam template.
	_ = render.HTML(w, http.StatusOK, "cek_resi", map[string]interface{}{
		"cart":    cart,                              // Menyertakan data cart (keranjang belanja) ke dalam template.
		"items":   items,                             // Menyertakan data item dalam cart ke dalam template.
		"success": flash.GetFlash(w, r, "success"),   // Menyertakan pesan flash sukses jika ada.
		"error":   flash.GetFlash(w, r, "error"),     // Menyertakan pesan flash error jika ada.
		"user":    auth.CurrentUser(server.DB, w, r), // Menyertakan data pengguna yang sedang login.
	})
}

func (server *Server) CekResiHandler(w http.ResponseWriter, r *http.Request) {
	// Cek apakah metode HTTP adalah POST
	if r.Method == http.MethodPost {
		// Inisialisasi render untuk merender template HTML
		render := render.New(render.Options{
			Layout:     "layout",                   // Layout default yang digunakan
			Extensions: []string{".html", ".tmpl"}, // Ekstensi file template
		})

		// Parsing form untuk mendapatkan data dari request
		r.ParseForm()
		resiNumber := r.FormValue("resi_number") // Mengambil nilai dari input "resi_number"

		// Menyusun URL API untuk cek resi berdasarkan nomor resi yang diterima
		url := fmt.Sprintf("https://api.biteship.com/v1/trackings/%s", resiNumber)
		req, err := http.NewRequest("GET", url, nil) // Membuat request GET ke API
		if err != nil {
			// Menangani error jika request gagal dibuat
			http.Error(w, "Gagal membuat request", http.StatusInternalServerError)
			return
		}

		// Menambahkan header Authorization untuk autentikasi ke API
		req.Header.Set("Authorization", "Bearer "+os.Getenv("API_BITESHIP"))

		// Membuat client HTTP untuk mengirim request
		client := &http.Client{}
		resp, err := client.Do(req) // Mengirim request ke API
		if err != nil {
			// Menangani error jika API gagal diakses
			http.Error(w, "Gagal memanggil API", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close() // Menutup body response setelah selesai digunakan

		// Membaca isi body response dari API
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			// Menangani error jika membaca response gagal
			http.Error(w, "Gagal membaca response", http.StatusInternalServerError)
			return
		}

		// Mendeklarasikan variabel untuk menyimpan hasil parsing JSON
		var trackingData models.TrackingResponse
		// Mem-parsing body response menjadi objek JSON
		err = json.Unmarshal(body, &trackingData)
		if err != nil {
			// Menangani error jika parsing JSON gagal
			http.Error(w, "Gagal parsing data JSON", http.StatusInternalServerError)
			return
		}

		// Merender template cek_resi dengan data hasil tracking
		_ = render.HTML(w, http.StatusOK, "cek_resi", map[string]interface{}{
			"resi_result": trackingData,         // Data hasil tracking
			"success":     trackingData.Success, // Status sukses dari API
			"error":       nil,                  // Tidak ada error untuk saat ini
		})
	} else {
		// Jika metode HTTP bukan POST, redirect ke halaman utama
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// Fungsi ini menampilkan halaman keranjang belanja (cart).
func (server *Server) GetCart(w http.ResponseWriter, r *http.Request) {
	// Membuat instance render dengan opsi layout dan ekstensi file.
	render := render.New(render.Options{
		Layout:     "layout",                   // Layout halaman yang akan digunakan.
		Extensions: []string{".html", ".tmpl"}, // Ekstensi file yang dikenali.
	})

	// Mendeklarasikan variabel cart.
	var cart *models.Cart

	// Mendapatkan ID keranjang belanja dari sesi pengguna atau cookie.
	cartID := GetShoppingCartID(w, r)
	// Mengambil data keranjang belanja dari database berdasarkan cartID.
	cart, _ = GetShoppingCart(server.DB, cartID)

	items, _ := GetCartItemsWithImages(server.DB, cartID)

	// Mengirimkan data ke template "cart" dan merender halaman.
	_ = render.HTML(w, http.StatusOK, "cart", map[string]interface{}{
		"cart":    cart,                              // Menampilkan data keranjang.
		"items":   items,                             // Menampilkan item dalam keranjang.
		"success": flash.GetFlash(w, r, "success"),   // Menampilkan pesan sukses (jika ada).
		"error":   flash.GetFlash(w, r, "error"),     // Menampilkan pesan error (jika ada).
		"user":    auth.CurrentUser(server.DB, w, r), // Menampilkan data pengguna yang sedang login.
	})
}

// Fungsi ini menambahkan item ke dalam keranjang belanja.
func (server *Server) AddItemToCart(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan ID produk dan kuantitas dari form.
	productID := r.FormValue("product_id")
	qty, _ := strconv.Atoi(r.FormValue("qty"))
	unit := r.FormValue("unit")

	// Mencari produk berdasarkan ID.
	productModel := models.Product{}
	product, err := productModel.FindByID(server.DB, productID)
	if err != nil {
		// Jika produk tidak ditemukan, redirect ke halaman produk.
		http.Redirect(w, r, "/products/"+product.Slug, http.StatusSeeOther)
		return
	}
	// Mengecek apakah kuantitas yang diminta lebih besar dari stok yang tersedia.
	if qty > product.Stock {
		// Jika stok tidak mencukupi, set flash message error dan redirect ke halaman produk.
		flash.SetFlash(w, r, "error", "Stok tidak mencukupi")
		http.Redirect(w, r, "/products/"+product.Slug, http.StatusSeeOther)
		return
	}
	// Tentukan harga berdasarkan satuan dan kuantitas
	if qty > 2 {
		// Harga grosir untuk qty > 2
		if unit == product.SATUAN1 {
			product.Price = product.HJ1
		} else if unit == product.SATUAN2 {
			product.Price = product.HJ2
		} else if unit == product.SATUAN3 {
			product.Price = product.HJ3
		} else {
			http.Error(w, "Invalid unit", http.StatusBadRequest)
			return
		}
	} else {
		// Harga eceran untuk qty <= 2
		if unit == product.SATUAN1 {
			product.Price = product.HJ2_1
		} else if unit == product.SATUAN2 {
			product.Price = product.HJ2_2
		} else if unit == product.SATUAN3 {
			product.Price = product.HJ2_3
		} else {
			http.Error(w, "Invalid unit", http.StatusBadRequest)
			return
		}
	}

	// Mendapatkan cartID dan mengambil data keranjang belanja.
	var cart *models.Cart
	cartID := GetShoppingCartID(w, r)
	cart, _ = GetShoppingCart(server.DB, cartID)
	// Menambahkan item ke dalam keranjang.
	_, err = cart.AddItem(server.DB, models.CartItem{
		ProductID: productID,
		Qty:       qty,
		Unit:      unit,                         // Simpan unit yang dipilih
		Pricenew:  int(product.Price.IntPart()), // Gunakan harga yang sudah di-set
	})
	if err != nil {
		// Jika ada error, redirect ke halaman produk.
		http.Redirect(w, r, "/products/"+product.Slug, http.StatusSeeOther)
	}

	// Set flash message sukses dan redirect ke halaman keranjang belanja.
	flash.SetFlash(w, r, "success", "Item berhasil ditambahkan")
	http.Redirect(w, r, "/carts", http.StatusSeeOther)
}

// Fungsi ini untuk memperbarui kuantitas item dalam keranjang belanja.
func (server *Server) UpdateCart(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan ID keranjang belanja dari sesi pengguna.
	cartID := GetShoppingCartID(w, r)
	cart, _ := GetShoppingCart(server.DB, cartID)

	// Mengupdate kuantitas setiap item dalam keranjang berdasarkan form input.
	for _, item := range cart.CartItems {
		// Mengambil kuantitas dari form.
		qty, _ := strconv.Atoi(r.FormValue(item.ID))

		// Skip jika qty 0 atau negatif
		if qty <= 0 {
			continue
		}

		// Memperbarui kuantitas item dalam keranjang tanpa mengubah harga
		// Gunakan harga yang sudah tersimpan di item.Pricenew
		_, err := cart.UpdateItemQty(server.DB, item.ID, qty)
		if err != nil {
			http.Redirect(w, r, "/carts", http.StatusSeeOther)
			return
		}
	}

	// Setelah selesai mengupdate, redirect kembali ke halaman keranjang.
	http.Redirect(w, r, "/carts", http.StatusSeeOther)
}

// Fungsi ini untuk menghapus item dari keranjang belanja berdasarkan ID.
func (server *Server) RemoveItemByID(w http.ResponseWriter, r *http.Request) {
	// Mengambil parameter ID item yang ingin dihapus dari URL.
	vars := mux.Vars(r)
	// Jika ID tidak ditemukan, redirect ke halaman keranjang.
	if vars["id"] == "" {
		http.Redirect(w, r, "/carts", http.StatusSeeOther)
	}
	// Mendapatkan cartID dan data keranjang.
	cartID := GetShoppingCartID(w, r)
	cart, _ := GetShoppingCart(server.DB, cartID)
	// Menghapus item dari keranjang berdasarkan ID.
	err := cart.RemoveItemByID(server.DB, vars["id"])
	if err != nil {
		// Jika ada error, redirect kembali ke halaman keranjang.
		http.Redirect(w, r, "/carts", http.StatusSeeOther)
	}
	// Setelah item dihapus, redirect ke halaman keranjang.
	http.Redirect(w, r, "/carts", http.StatusSeeOther)
}

// perhitungan biteship
// Fungsi CalculateShippingBiteship digunakan untuk menghitung biaya pengiriman menggunakan API Biteship berdasarkan parameter yang diberikan oleh pengguna.
func (server *Server) CalculateShippingBiteship(w http.ResponseWriter, r *http.Request) {
	// Mengambil nilai-nilai dari form data yang dikirim oleh pengguna.
	destination := r.FormValue("city_id")                            // ID kota tujuan pengiriman.
	courier := r.FormValue("courier")                                // Nama kurir yang dipilih untuk pengiriman.
	cour_type := r.FormValue("cour_type")                            // Tipe kurir yang dipilih (misalnya, reguler atau instant).
	latitude := r.FormValue("latitude")                              // Latitude (garis lintang) dari lokasi asal pengiriman.
	longitude := r.FormValue("longitude")                            // Longitude (garis bujur) dari lokasi asal pengiriman.
	default_location := os.Getenv("API_BITESHIP_SAMARINDA_LOCATION") // Lokasi default yang diambil dari environment variable (misalnya, Samarinda).

	// DEBUG: Log semua form values yang diterima
	log.Printf("🔍 FORM VALUES RECEIVED:")
	log.Printf("   destination (city_id): '%s'", destination)
	log.Printf("   courier: '%s'", courier)
	log.Printf("   cour_type: '%s'", cour_type)
	log.Printf("   latitude: '%s'", latitude)
	log.Printf("   longitude: '%s'", longitude)
	log.Printf("   default_location: '%s'", default_location)

	// Validasi input yang diterima dari form, memastikan destination tidak kosong.
	if destination == "" {
		// Fallback ke default location jika city_id kosong
		destination = default_location
		if destination == "" {
			log.Printf("❌ ERROR: No destination provided and no default location")
			http.Error(w, "Invalid destination: no city_id provided and no default location configured", http.StatusBadRequest)
			return
		}
		log.Printf("⚠️  Using default location: %s", destination)
	}

	// Mengambil ID keranjang belanja dari sesi pengguna atau cookie.
	cartID := GetShoppingCartID(w, r)

	// Mengambil informasi keranjang belanja berdasarkan cartID.
	cart, err := GetShoppingCart(server.DB, cartID)
	if err != nil {
		// Jika gagal mengambil data keranjang belanja, kirimkan error dengan status 500 (Internal Server Error).
		http.Error(w, "Failed to retrieve shopping cart", http.StatusInternalServerError)
		return
	}

	// Mendeklarasikan variabel untuk menyimpan opsi biaya pengiriman
	var shippingFeeOptions []models.Pricing

	// Mengecek jenis kurir berdasarkan nilai 'cour_type'
	if cour_type == "instant" {
		// Untuk instant delivery, gunakan koordinat latitude/longitude
		if latitude == "" || longitude == "" {
			http.Error(w, "Latitude and longitude required for instant delivery", http.StatusBadRequest)
			return
		}

		// Jika jenis kurir adalah "instant", hitung biaya pengiriman menggunakan metode instant
		shippingFeeOptions, err = server.CalculateShippingFeeBiteshipInstant(ShippingFeeParams{
			Origin:      latitude,         // Gunakan latitude dari form
			Destination: longitude,        // Gunakan longitude dari form
			Weight:      cart.TotalWeight, // Berat total dari keranjang
			Couriers:    courier,          // Kurir yang digunakan
		})
		log.Printf("Instant delivery calculation with lat: %s, lng: %s", latitude, longitude)
	} else if cour_type == "pickup" {
		// Untuk pickup, tidak ada biaya pengiriman
		shippingFeeOptions = []models.Pricing{
			{
				CourierName: "Pickup",
				Price:       0,
			},
		}
	} else {
		// Untuk regular delivery
		log.Printf("Regular delivery calculation for: %s", cour_type)
		// Konversi city_id user ke Biteship area ID untuk regular delivery
		destinationAreaID := convertCityIDToBiteshipAreaID(destination)
		// Hitung biaya pengiriman menggunakan metode biasa
		shippingFeeOptions, err = server.CalculateShippingFeeBiteship(ShippingFeeParams{
			Origin:      default_location,  // Origin tetap default (toko)
			Destination: destinationAreaID, // Destination menggunakan area ID yang benar
			Weight:      cart.TotalWeight,  // Berat total dari keranjang
			Couriers:    courier,           // Kurir yang digunakan
		})
		log.Printf("Regular delivery calculation: %s -> %s", default_location, destinationAreaID)
	}

	// Mengecek apakah terdapat error dalam proses perhitungan biaya pengiriman
	if err != nil {
		// Mengembalikan error jika proses perhitungan gagal
		http.Error(w, fmt.Sprintf("Failed to calculate shipping fees: %v", err), http.StatusInternalServerError)
		return
	}

	// Membuat respons sukses dengan informasi area
	responseData := map[string]interface{}{
		"pricing": shippingFeeOptions,
	}

	// Selalu sertakan informasi lokasi, terlepas dari hasil API
	if cour_type == "instant" {
		responseData["origin"] = map[string]interface{}{
			"area_name": "Toko Shafirda, Samarinda",
			"latitude":  -0.5262810043373423,
			"longitude": 117.13669626404219,
		}

		// Untuk instant delivery, gunakan koordinat
		destinationAreaName := "Lokasi dari Peta"
		if latitude != "" && longitude != "" {
			destinationAreaName = fmt.Sprintf("Koordinat: %s, %s", latitude, longitude)
		}

		responseData["destination"] = map[string]interface{}{
			"area_name": destinationAreaName,
			"latitude":  latitude,
			"longitude": longitude,
		}
	} else if cour_type == "pickup" {
		responseData["origin"] = map[string]interface{}{
			"area_name": "Toko Shafirda, Samarinda",
		}
		responseData["destination"] = map[string]interface{}{
			"area_name": "Pickup di Toko",
		}
	} else {
		// Regular delivery - gunakan mapping lokal untuk nama area
		responseData["origin"] = map[string]interface{}{
			"area_name": "Toko Shafirda, Samarinda",
			"area_id":   default_location,
		}

		// Konversi city_id ke area_id dan dapatkan nama dari mapping lokal
		destinationAreaID := convertCityIDToBiteshipAreaID(destination)
		destinationAreaName := getAreaNameFromID(destinationAreaID)

		responseData["destination"] = map[string]interface{}{
			"area_name": destinationAreaName,
			"area_id":   destinationAreaID,
			"city_id":   destination, // Simpan city_id asli untuk checkout
		}
	}

	res := Result{
		Code:    200,          // Kode status sukses
		Data:    responseData, // Data dengan informasi area
		Message: "Success",    // Pesan sukses
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
func (server *Server) ApplyShipping(w http.ResponseWriter, r *http.Request) {
	// Mengambil lokasi default dari environment variable.
	default_location := os.Getenv("API_BITESHIP_SAMARINDA_LOCATION")

	// Mengambil nilai input dari form yang dikirim pengguna.
	destination := r.FormValue("city_id")
	courier := r.FormValue("courier")
	cour_type := r.FormValue("cour_type")
	shippingPackage := r.FormValue("shipping_package")

	// Mendapatkan ID keranjang belanja pengguna.
	cartID := GetShoppingCartID(w, r)

	// Mengambil data keranjang belanja dari database.
	cart, _ := GetShoppingCart(server.DB, cartID)

	// Validasi input tujuan pengiriman.
	if destination == "" {
		// Fallback ke default location jika city_id kosong
		destination = default_location
		if destination == "" {
			http.Error(w, "invalid destination: no city_id provided and no default location configured", http.StatusInternalServerError)
			return
		}
	}

	// Inisialisasi variabel untuk menyimpan opsi biaya pengiriman dan error.
	var shippingFeeOptions []models.Pricing
	var err error

	// Menghitung biaya pengiriman berdasarkan jenis layanan kurir.
	if cour_type == "instant" {
		// Untuk instant delivery, kita perlu koordinat latitude/longitude
		latitudeStr := r.FormValue("latitude")
		longitudeStr := r.FormValue("longitude")

		if latitudeStr == "" || longitudeStr == "" {
			http.Error(w, "Latitude and longitude required for instant delivery", http.StatusBadRequest)
			return
		}

		shippingFeeOptions, err = server.CalculateShippingFeeBiteshipInstant(ShippingFeeParams{
			Origin:      latitudeStr,  // Gunakan latitude dari form
			Destination: longitudeStr, // Gunakan longitude dari form
			Weight:      cart.TotalWeight,
			Couriers:    courier,
		})
		log.Printf("Apply instant delivery with lat: %s, lng: %s", latitudeStr, longitudeStr)
	} else if cour_type == "pickup" {
		// Untuk pickup, tidak ada biaya pengiriman
		shippingFeeOptions = []models.Pricing{
			{
				CourierName: "Pickup",
				Price:       0,
			},
		}
	} else {
		log.Printf("Apply regular delivery for: %s", cour_type)
		// Konversi city_id user ke Biteship area ID
		destinationAreaID := convertCityIDToBiteshipAreaID(destination)
		shippingFeeOptions, err = server.CalculateShippingFeeBiteship(ShippingFeeParams{
			Origin:      default_location,  // Origin tetap default (toko)
			Destination: destinationAreaID, // Destination menggunakan area ID yang benar
			Weight:      cart.TotalWeight,
			Couriers:    courier,
		})
		log.Printf("Apply regular delivery: %s -> %s", default_location, destinationAreaID)
	}

	// Menangani error jika terjadi kesalahan perhitungan biaya pengiriman.
	if err != nil {
		http.Error(w, "invalid shipping calculation", http.StatusInternalServerError)
		return
	}

	// Menemukan opsi pengiriman yang dipilih oleh pengguna.
	var selectedShipping models.Pricing

	for _, shippingOption := range shippingFeeOptions {
		if shippingOption.CourierName == shippingPackage {
			selectedShipping = shippingOption
			continue
		}
	}

	// Struktur respons untuk data pengiriman yang diterapkan.
	type ApplyShippingResponse struct {
		TotalOrder  decimal.Decimal        `json:"total_order"`
		ShippingFee decimal.Decimal        `json:"shipping_fee"`
		GrandTotal  decimal.Decimal        `json:"grand_total"`
		TotalWeight decimal.Decimal        `json:"total_weight"`
		Origin      map[string]interface{} `json:"origin"`
		Destination map[string]interface{} `json:"destination"`
		CourierInfo map[string]interface{} `json:"courier_info"`
	}

	// Menghitung total biaya termasuk biaya pengiriman.
	var grandTotal float64

	cartGrandTotal, _ := cart.GrandTotal.Float64()
	shippingFee := float64(selectedShipping.Price)
	grandTotal = cartGrandTotal + shippingFee

	// Siapkan informasi area untuk response - harus konsisten dengan CalculateShippingBiteship
	var originInfo, destinationInfo map[string]interface{}

	if cour_type == "instant" {
		originInfo = map[string]interface{}{
			"area_name": "Toko Shafirda, Samarinda",
			"latitude":  -0.5262810043373423,
			"longitude": 117.13669626404219,
		}

		destinationAreaName := "Lokasi dari Peta"
		lat := r.FormValue("latitude")
		lng := r.FormValue("longitude")
		if lat != "" && lng != "" {
			destinationAreaName = fmt.Sprintf("Koordinat: %s, %s", lat, lng)
		}

		destinationInfo = map[string]interface{}{
			"area_name": destinationAreaName,
			"latitude":  lat,
			"longitude": lng,
		}
	} else if cour_type == "pickup" {
		originInfo = map[string]interface{}{
			"area_name": "Toko Shafirda, Samarinda",
		}
		destinationInfo = map[string]interface{}{
			"area_name": "Pickup di Toko",
		}
	} else {
		// Regular delivery - gunakan mapping lokal yang sama
		destinationAreaID := convertCityIDToBiteshipAreaID(destination)
		destinationAreaName := getAreaNameFromID(destinationAreaID)

		originInfo = map[string]interface{}{
			"area_name": "Toko Shafirda, Samarinda",
			"area_id":   default_location,
		}
		destinationInfo = map[string]interface{}{
			"area_name": destinationAreaName,
			"area_id":   destinationAreaID,
			"city_id":   destination,
		}
	}

	// Menyiapkan respons dengan data yang dihitung.
	applyShippingResponse := ApplyShippingResponse{
		TotalOrder:  cart.GrandTotal,
		ShippingFee: decimal.NewFromInt(int64(selectedShipping.Price)),
		GrandTotal:  decimal.NewFromFloat(grandTotal),
		TotalWeight: decimal.NewFromInt(int64(cart.TotalWeight)),
		Origin:      originInfo,
		Destination: destinationInfo,
		CourierInfo: map[string]interface{}{
			"courier_name": selectedShipping.CourierName,
			"service_name": selectedShipping.CourierServiceName,
			"duration":     selectedShipping.Duration,
		},
	}

	// Menyiapkan data untuk dikirim dalam format JSON.
	res := Result{Code: 200, Data: applyShippingResponse, Message: "Success"}
	result, _ := json.Marshal(res)

	// Menulis respons HTTP dengan status 200 OK.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(result)
}

func GetCartItemsWithImages(db *gorm.DB, cartID string) ([]models.CartItem, error) {
	var cartItems []models.CartItem

	// Preload Product and ProductImages relationships to ensure images are loaded
	err := db.Preload("Product.ProductImages").Where("cart_id = ?", cartID).Find(&cartItems).Error
	if err != nil {
		return nil, err
	}

	// Debug logging to check if images are loaded
	for _, item := range cartItems {
		if len(item.Product.ProductImages) == 0 {
			log.Printf("Warning: Product %s (ID: %s) in cart has no images", item.Product.Name, item.ProductID)
		} else {
			log.Printf("Success: Product %s has %d images loaded", item.Product.Name, len(item.Product.ProductImages))
		}
	}

	return cartItems, nil
}

package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/gosimple/slug"
	"github.com/shopspring/decimal"

	"github.com/gieart87/gotoko/app/models"
	"github.com/gieart87/gotoko/database/seeders"
	"github.com/gorilla/mux"
	"github.com/urfave/cli"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Server struct digunakan untuk menyimpan konfigurasi utama aplikasi
type Server struct {
	DB        *gorm.DB    // Koneksi ke database menggunakan GORM
	Router    *mux.Router // Router untuk mengatur rute aplikasi
	AppConfig *AppConfig  // Konfigurasi aplikasi seperti nama, lingkungan, dan URL
}

// AppConfig struct digunakan untuk menyimpan konfigurasi aplikasi
type AppConfig struct {
	AppName string // Nama aplikasi
	AppEnv  string // Lingkungan aplikasi (misalnya: development, production)
	AppPort string // Port yang digunakan aplikasi
	AppURL  string // URL dasar aplikasi
}

// DBConfig struct digunakan untuk menyimpan konfigurasi database
type DBConfig struct {
	DBHost     string // Host database
	DBUser     string // Username untuk mengakses database
	DBPassword string // Password untuk mengakses database
	DBName     string // Nama database yang akan digunakan
	DBPort     string // Port koneksi database
	DBDriver   string // Driver database (misalnya: postgres, mysql)
}

// PageLink struct digunakan untuk membuat tautan halaman dalam pagination
type PageLink struct {
	Page          int32  // Nomor halaman yang terkait dengan tautan
	Url           string // URL untuk mengakses halaman tersebut
	IsCurrentPage bool   // Menandakan apakah halaman ini adalah halaman yang sedang dilihat
}

// PaginationLinks struct digunakan untuk menyimpan informasi pagination
type PaginationLinks struct {
	CurrentPage string     // Halaman saat ini dalam pagination
	NextPage    string     // Tautan ke halaman berikutnya (jika ada)
	PrevPage    string     // Tautan ke halaman sebelumnya (jika ada)
	TotalRows   int32      // Total jumlah baris data
	TotalPages  int32      // Total jumlah halaman yang tersedia
	Links       []PageLink // Daftar tautan ke setiap halaman
}

// PaginationParams struct digunakan untuk parameter pagination
type PaginationParams struct {
	Path        string // Path dasar untuk membuat URL pagination
	TotalRows   int32  // Total jumlah baris data
	PerPage     int32  // Jumlah baris per halaman
	CurrentPage int32  // Halaman saat ini yang sedang dilihat
}

// Result struct digunakan untuk menyimpan respons API dalam format JSON
type Result struct {
	Code    int         `json:"code"`    // Kode status (misalnya: 200 untuk sukses, 400 untuk error)
	Data    interface{} `json:"data"`    // Data yang akan dikembalikan dalam respons
	Message string      `json:"message"` // Pesan terkait hasil operasi
}

// Inisialisasi penyimpanan sesi menggunakan kunci dari variabel lingkungan
var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

// Nama sesi yang digunakan dalam aplikasi
var sessionShoppingCart = "shopping-cart-session" // Sesi untuk keranjang belanja
var sessionFlash = "flash-session"                // Sesi untuk pesan flash

// Initialize menginisialisasi server dengan konfigurasi aplikasi dan database
func (server *Server) Initialize(appConfig AppConfig, dbConfig DBConfig) {
	// Menampilkan pesan selamat datang dengan nama aplikasi
	fmt.Println("Welcome to " + appConfig.AppName)

	// Menginisialisasi koneksi database
	server.initializeDB(dbConfig)

	// Menyimpan konfigurasi aplikasi ke dalam server
	server.initializeAppConfig(appConfig)

	// Mengatur rute aplikasi
	server.initializeRoutes()
}

// Run menjalankan server pada alamat tertentu
func (server *Server) Run(addr string) {
	// Menampilkan pesan bahwa server mendengarkan pada port tertentu
	fmt.Printf("Listening to port %s", addr)

	// Menjalankan server HTTP dan mencatat error jika terjadi
	log.Fatal(http.ListenAndServe(addr, server.Router))
}

// initializeDB mengatur koneksi ke database sesuai dengan konfigurasi yang diberikan
func (server *Server) initializeDB(dbConfig DBConfig) {
	var err error

	// Membuat DSN untuk PostgreSQL jika driver adalah postgres
	if dbConfig.DBDriver == "postgres" {
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
			dbConfig.DBHost, dbConfig.DBUser, dbConfig.DBPassword, dbConfig.DBName, dbConfig.DBPort,
		)
		server.DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{}) // Membuka koneksi ke database PostgreSQL
	} else {
		// Membuat DSN untuk MySQL jika driver bukan postgres
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			dbConfig.DBUser, dbConfig.DBPassword, dbConfig.DBHost, dbConfig.DBPort, dbConfig.DBName,
		)
		server.DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{}) // Membuka koneksi ke database MySQL
	}
	// Jika ada error saat menghubungkan ke database, program akan dihentikan
	if err != nil {
		panic("Failed on connecting to the database server")
	}
}

// initializeAppConfig menyimpan konfigurasi aplikasi ke dalam server
func (server *Server) initializeAppConfig(appConfig AppConfig) {
	server.AppConfig = &appConfig
}

// dbMigrate melakukan migrasi database dengan semua model yang terdaftar
func (server *Server) dbMigrate() {
	// Iterasi setiap model yang terdaftar untuk migrasi
	for _, model := range models.RegisterModels() {
		// Melakukan migrasi model dengan mode debug untuk log detail
		err := server.DB.Debug().AutoMigrate(model.Model)
		// Jika terjadi error saat migrasi, catat dan hentikan program
		if err != nil {
			log.Fatal(err)
		}
	}
	// Menampilkan pesan sukses setelah migrasi selesai
	fmt.Println("Database migrated successfully.")
}

func (server *Server) InitCommands(config AppConfig, dbConfig DBConfig) {
	// Inisialisasi koneksi database dengan memanggil fungsi initializeDB
	server.initializeDB(dbConfig)

	// Membuat aplikasi CLI baru menggunakan paket urfave/cli
	cmdApp := cli.NewApp()

	// Menentukan perintah CLI yang tersedia
	cmdApp.Commands = []cli.Command{
		{
			// Nama perintah pertama adalah "db:migrate"
			Name: "db:migrate",
			Action: func(c *cli.Context) error {
				// Menjalankan migrasi database
				server.dbMigrate()
				return nil
			},
		},
		{
			// Nama perintah kedua adalah "db:seed"
			Name: "db:seed",
			Action: func(c *cli.Context) error {
				// Menjalankan seed data ke database
				err := seeders.DBSeed(server.DB)
				if err != nil {
					// Melaporkan kesalahan jika seed gagal
					log.Fatal(err)
				}
				return nil
			},
		},
		{
			// Nama perintah kedua adalah "db:seed"
			Name: "db:excel2",
			Action: func(c *cli.Context) error {
				err := server.importProductsFromExcel2("PRODUK.xlsx")
				if err != nil {
					log.Fatal(err)
				}
				return nil
			},
		},
		{
			Name: "db:excel",
			Action: func(c *cli.Context) error {
				err := server.importProductsFromExcel("PRODUK.xlsx")
				if err != nil {
					log.Fatal(err)
				}
				return nil
			},
		},
	}

	err := cmdApp.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func (server *Server) importProductsFromExcel(filePath string) error {
	// Buka file Excel menggunakan excelize
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("gagal membuka file Excel: %w", err)
	}

	// Ambil nama sheet pertama
	sheetName := f.GetSheetName(1)
	if sheetName == "" {
		return fmt.Errorf("sheet tidak ditemukan dalam file Excel")
	}

	// Ambil semua baris dari sheet
	rows := f.GetRows(sheetName)

	// Iterasi data dan masukkan ke database
	for i, row := range rows {
		if i == 0 {
			continue // Lewati header
		}

		// Pastikan jumlah kolom mencukupi
		if len(row) < 20 {
			log.Printf("Baris %d dilewati karena kolom tidak mencukupi", i+1)
			continue
		}

		// Parsing data dari Excel ke Product
		product := models.Product{
			ID:          row[0],
			Name:        row[1],
			SATUAN1:     row[3],
			SATUAN2:     row[4],
			SATUAN3:     row[5],
			KONVERSI1:   toInt(row[6]),
			KONVERSI2:   toInt(row[7]),
			KONVERSI3:   toInt(row[8]),
			HARGAPOKOK1: toDecimal(row[9]),
			HARGAPOKOK2: toDecimal(row[10]),
			HARGAPOKOK3: toDecimal(row[11]),
			HJ1:         toDecimal(row[12]),
			HJ2:         toDecimal(row[13]),
			HJ3:         toDecimal(row[14]),
			HJ2_1:       toDecimal(row[15]),
			HJ2_2:       toDecimal(row[16]),
			HJ2_3:       toDecimal(row[17]),
			Stock:       toInt(row[18]),
			Supplier:    row[19],
			Categories:  row[2],
			Sku:         slug.Make(fmt.Sprintf("%s-%s", row[2], row[1])),
			Slug:        slug.Make(fmt.Sprintf("%s-%s", row[2], row[1])),
			Status:      1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Simpan ke database
		if err := server.DB.Create(&product).Error; err != nil {
			log.Printf("Gagal menyimpan produk ID %s: %v", product.ID, err)
		}
	}

	log.Println("Import data dari Excel berhasil")
	return nil
}

// Fungsi bantu untuk konversi string ke integer
func toInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func (server *Server) importProductsFromExcel2(filePath string) error {
	// Buka file Excel menggunakan excelize
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("gagal membuka file Excel: %w", err)
	}

	// Ambil nama sheet pertama
	sheetName := f.GetSheetName(1)
	if sheetName == "" {
		return fmt.Errorf("sheet tidak ditemukan dalam file Excel")
	}

	// Ambil semua baris dari sheet
	rows := f.GetRows(sheetName)

	// Iterasi data dan masukkan ke database
	for i, row := range rows {
		if i == 0 {
			continue // Lewati header
		}

		// Pastikan jumlah kolom mencukupi
		if len(row) < 20 {
			log.Printf("Baris %d dilewati karena kolom tidak mencukupi", i+1)
			continue
		}

		// Parsing data dari Excel ke Product
		productimage := models.ProductImage{
			ID:        uuid.New().String(),
			ProductID: row[0],
		}

		// Simpan ke database
		if err := server.DB.Create(&productimage).Error; err != nil {
			log.Printf("Gagal menyimpan produk ID %s: %v", productimage.ProductID, err)
		}
	}

	log.Println("Import data dari Excel berhasil")
	return nil
}

// Fungsi bantu untuk konversi string ke decimal.Decimal
func toDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

// convertCityIDToBiteshipAreaID mengkonversi City ID Indonesia ke Biteship Area ID
func convertCityIDToBiteshipAreaID(cityID string) string {
	log.Printf("Converting city ID: %s", cityID)

	// Check if we're in testing mode based on API key
	apiKey := os.Getenv("API_BITESHIP")
	isTestingMode := strings.Contains(apiKey, "biteship_test")

	if isTestingMode {
		log.Printf("🧪 TESTING MODE DETECTED - Using simplified mapping")
		// Mapping untuk TESTING MODE - lebih sederhana karena data sandbox terbatas
		testingMapping := map[string]string{
			// Semua Samarinda districts map ke city-level untuk compatibility
			"IDNC383": "IDNC383", // Samarinda
			"6472010": "IDNC383", // Kec. Sungai Kunjang
			"6472020": "IDNC383", // Kec. Sambutan
			"6472030": "IDNC383", // Kec. Palaran
			"6472040": "IDNC383", // Kec. Samarinda Seberang
			"6472050": "IDNC383", // Kec. Samarinda Ulu
			"6472060": "IDNC383", // Kec. Samarinda Ilir
			"6472070": "IDNC383", // Kec. Samarinda Utara
			"6472080": "IDNC383", // Kec. Loa Janan Ilir
			"6472090": "IDNC383", // Kec. Sungai Pinang
			"6472100": "IDNC383", // Kec. Samarinda Kota

			// Kutai Kartanegara
			"IDNC386": "IDNC210",
			"6403010": "IDNC210", // Kec. Samboja
			"6403020": "IDNC210", // Kec. Muara Jawa
			"6403030": "IDNC210", // Kec. Sanga-Sanga
			"6403040": "IDNC210", // Kec. Tenggarong
			"6403050": "IDNC210", // Kec. Sebulu
			"6403060": "IDNC210", // Kec. Kota Bangun
			"6403070": "IDNC210", // Kec. Kenohan
		}

		if areaID, exists := testingMapping[cityID]; exists {
			log.Printf("✅ Testing mapping found: %s -> %s", cityID, areaID)
			return areaID
		}
		log.Printf("❌ Testing mapping NOT found for: %s, using original ID", cityID)
		return cityID
	}

	// Mapping untuk PRODUCTION MODE - menggunakan district-level IDs yang spesifik
	log.Printf("🚀 PRODUCTION MODE - Using detailed mapping")
	cityMapping := map[string]string{
		// Kalimantan Timur - Samarinda (DISTRICT-LEVEL SPECIFIC IDs)
		"IDNC383": "IDNC383",               // Samarinda default (city level)
		"6472010": "IDNP15IDNC383IDND4551", // Kec. Sungai Kunjang
		"6472020": "IDNP15IDNC383IDND4549", // Kec. Sambutan
		"6472030": "IDNP15IDNC383IDND4548", // Kec. Palaran - SPESIFIK!
		"6472040": "IDNP15IDNC383IDND4554", // Kec. Samarinda Seberang - SPESIFIK!
		"6472050": "IDNP15IDNC383IDND4553", // Kec. Samarinda Ulu
		"6472060": "IDNP15IDNC383IDND4552", // Kec. Samarinda Ilir
		"6472070": "IDNP15IDNC383IDND4555", // Kec. Samarinda Utara
		"6472080": "IDNP15IDNC383IDND4550", // Kec. Loa Janan Ilir
		"6472090": "IDNP15IDNC383IDND4556", // Kec. Sungai Pinang
		"6472100": "IDNP15IDNC383IDND4557", // Kec. Samarinda Kota

		// Kalimantan Timur - Balikpapan
		"IDNC384": "IDNC384", // Balikpapan
		"6471010": "IDNC384", // Kec. Balikpapan Kota
		"6471020": "IDNC384", // Kec. Balikpapan Selatan
		"6471030": "IDNC384", // Kec. Balikpapan Timur
		"6471040": "IDNC384", // Kec. Balikpapan Utara
		"6471050": "IDNC384", // Kec. Balikpapan Tengah
		"6471060": "IDNC384", // Kec. Balikpapan Barat

		// Kalimantan Timur - Bontang
		"IDNC385": "IDNC385", // Bontang
		"6472210": "IDNC385", // Kec. Bontang Utara
		"6472220": "IDNC385", // Kec. Bontang Selatan
		"6472230": "IDNC385", // Kec. Bontang Barat

		// Kutai Kartanegara (DISTRICT-LEVEL SPECIFIC IDs)
		"IDNC386": "IDNC210",               // Kutai Kartanegara default (city level)
		"6403010": "IDNP15IDNC210IDND1881", // Kec. Samboja
		"6403020": "IDNP15IDNC210IDND1879", // Kec. Muara Jawa
		"6403030": "IDNP15IDNC210IDND1878", // Kec. Sanga-Sanga - SPESIFIK & BERBEDA!
		"6403040": "IDNP15IDNC210IDND1882", // Kec. Tenggarong
		"6403050": "IDNP15IDNC210IDND1883", // Kec. Sebulu
		"6403060": "IDNP15IDNC210IDND1884", // Kec. Kota Bangun
		"6403070": "IDNP15IDNC210IDND1885", // Kec. Kenohan

		// Jakarta
		"IDNC001": "IDNC001", // Jakarta Pusat
		"3171010": "IDNC001", // Gambir
		"3171020": "IDNC001", // Sawah Besar
		"3171030": "IDNC001", // Kemayoran
		"3171040": "IDNC001", // Senen
		"3171050": "IDNC001", // Cempaka Putih
		"3171060": "IDNC001", // Menteng
		"3171070": "IDNC001", // Tanah Abang
		"3171080": "IDNC001", // Johar Baru

		"IDNC002": "IDNC002", // Jakarta Utara
		"3174010": "IDNC002", // Penjaringan
		"3174020": "IDNC002", // Pademangan
		"3174030": "IDNC002", // Tanjung Priok
		"3174040": "IDNC002", // Koja
		"3174050": "IDNC002", // Kelapa Gading
		"3174060": "IDNC002", // Cilincing

		"IDNC003": "IDNC003", // Jakarta Selatan
		"3173010": "IDNC003", // Kebayoran Baru
		"3173020": "IDNC003", // Kebayoran Lama
		"3173030": "IDNC003", // Pesanggrahan
		"3173040": "IDNC003", // Cilandak
		"3173050": "IDNC003", // Pasar Minggu
		"3173060": "IDNC003", // Jagakarsa
		"3173070": "IDNC003", // Mampang Prapatan
		"3173080": "IDNC003", // Pancoran
		"3173090": "IDNC003", // Tebet
		"3173100": "IDNC003", // Setiabudi

		"IDNC004": "IDNC004", // Jakarta Barat
		"3172010": "IDNC004", // Tambora
		"3172020": "IDNC004", // Taman Sari
		"3172030": "IDNC004", // Cengkareng
		"3172040": "IDNC004", // Grogol Petamburan
		"3172050": "IDNC004", // Kebon Jeruk
		"3172060": "IDNC004", // Kembangan
		"3172070": "IDNC004", // Palmerah
		"3172080": "IDNC004", // Kali Deres

		"IDNC005": "IDNC005", // Jakarta Timur
		"3175010": "IDNC005", // Pasar Rebo
		"3175020": "IDNC005", // Ciracas
		"3175030": "IDNC005", // Cipayung
		"3175040": "IDNC005", // Makasar
		"3175050": "IDNC005", // Kramat Jati
		"3175060": "IDNC005", // Jatinegara
		"3175070": "IDNC005", // Duren Sawit
		"3175080": "IDNC005", // Cakung
		"3175090": "IDNC005", // Pulo Gadung
		"3175100": "IDNC005", // Matraman

		// Surabaya
		"IDNC100": "IDNC100", // Surabaya
		"3578010": "IDNC100", // Karang Pilang
		"3578020": "IDNC100", // Wonocolo
		"3578030": "IDNC100", // Rungkut
		"3578040": "IDNC100", // Wonokromo
		"3578050": "IDNC100", // Tegalsari
		"3578060": "IDNC100", // Sawahan
		"3578070": "IDNC100", // Genteng
		"3578080": "IDNC100", // Bubutan
		"3578090": "IDNC100", // Simokerto
		"3578100": "IDNC100", // Pabean Cantian
		"3578110": "IDNC100", // Semampir
		"3578120": "IDNC100", // Krembangan
		"3578130": "IDNC100", // Kenjeran
		"3578140": "IDNC100", // Lakarsantri
		"3578150": "IDNC100", // Benowo
		"3578160": "IDNC100", // Wiyung
		"3578170": "IDNC100", // Dukuh Pakis
		"3578180": "IDNC100", // Gayungan
		"3578190": "IDNC100", // Jambangan
		"3578200": "IDNC100", // Tenggilis Mejoyo
		"3578210": "IDNC100", // Gunung Anyar
		"3578220": "IDNC100", // Sukolilo
		"3578230": "IDNC100", // Bulak
		"3578240": "IDNC100", // Mulyorejo
		"3578250": "IDNC100", // Gubeng
		"3578260": "IDNC100", // Sukomanunggal
		"3578270": "IDNC100", // Tandes
		"3578280": "IDNC100", // Asemrowo
		"3578290": "IDNC100", // Pakal
		"3578300": "IDNC100", // Sambikerep

		// Sidoarjo
		"IDNC101": "IDNC101", // Sidoarjo
		"3515010": "IDNC101", // Kec. Sidoarjo
		"3515020": "IDNC101", // Kec. Buduran
		"3515030": "IDNC101", // Kec. Candi
		"3515040": "IDNC101", // Kec. Porong
		"3515050": "IDNC101", // Kec. Krembung
		"3515060": "IDNC101", // Kec. Tulangan
		"3515070": "IDNC101", // Kec. Tanggulangin
		"3515080": "IDNC101", // Kec. Jabon
		"3515090": "IDNC101", // Kec. Krian
		"3515100": "IDNC101", // Kec. Balongbendo
		"3515110": "IDNC101", // Kec. Wonoayu
		"3515120": "IDNC101", // Kec. Tarik
		"3515130": "IDNC101", // Kec. Prambon
		"3515140": "IDNC101", // Kec. Taman
		"3515150": "IDNC101", // Kec. Sukodono
		"3515160": "IDNC101", // Kec. Gedangan
		"3515170": "IDNC101", // Kec. Sedati
		"3515180": "IDNC101", // Kec. Waru
	}

	if areaID, exists := cityMapping[cityID]; exists {
		log.Printf("✅ City mapping found: %s -> %s", cityID, areaID)

		// DEBUGGING: Log specific mappings
		if cityID == "6472030" {
			log.Printf("🎯 PALARAN MAPPED TO: %s", areaID)
		}
		if cityID == "6403030" {
			log.Printf("🎯 SANGA-SANGA MAPPED TO: %s", areaID)
		}

		return areaID
	}

	// Jika tidak ada mapping, gunakan city ID asli dan log untuk debugging
	log.Printf("❌ City mapping NOT found for: %s, using original ID", cityID)
	return cityID
}

// getAreaNameFromID mengkonversi Area ID ke nama area yang user-friendly
func getAreaNameFromID(areaID string) string {
	// Mapping dari area_id ke nama yang user-friendly - SESUAI DENGAN DROPDOWN DI CART.HTML
	areaNameMapping := map[string]string{
		// Kalimantan Timur - Samarinda
		"IDNC383": "Samarinda",
		"6472010": "Kec. Sungai Kunjang, Samarinda",
		"6472020": "Kec. Sambutan, Samarinda",
		"6472030": "Kec. Palaran, Samarinda",
		"6472040": "Kec. Samarinda Seberang, Samarinda",
		"6472050": "Kec. Samarinda Ulu, Samarinda",
		"6472060": "Kec. Samarinda Ilir, Samarinda",
		"6472070": "Kec. Samarinda Utara, Samarinda",
		"6472080": "Kec. Loa Janan Ilir, Samarinda",
		"6472090": "Kec. Sungai Pinang, Samarinda",
		"6472100": "Kec. Samarinda Kota, Samarinda",
		// Kalimantan Timur - Balikpapan
		"IDNC384": "Balikpapan",
		"6471010": "Kec. Balikpapan Kota, Balikpapan",
		"6471020": "Kec. Balikpapan Selatan, Balikpapan",
		"6471030": "Kec. Balikpapan Timur, Balikpapan",
		"6471040": "Kec. Balikpapan Utara, Balikpapan",
		"6471050": "Kec. Balikpapan Tengah, Balikpapan",
		"6471060": "Kec. Balikpapan Barat, Balikpapan",

		// Kalimantan Timur - Bontang
		"IDNC385": "Bontang",
		"6472210": "Kec. Bontang Utara, Bontang",
		"6472220": "Kec. Bontang Selatan, Bontang",
		"6472230": "Kec. Bontang Barat, Bontang",

		// Kutai Kartanegara
		"IDNC386": "Kutai Kartanegara",
		"6403010": "Kec. Samboja, Kutai Kartanegara",
		"6403020": "Kec. Muara Jawa, Kutai Kartanegara",
		"6403030": "Kec. Sanga-Sanga, Kutai Kartanegara",
		"6403040": "Kec. Tenggarong, Kutai Kartanegara",
		"6403050": "Kec. Sebulu, Kutai Kartanegara",
		"6403060": "Kec. Kota Bangun, Kutai Kartanegara",
		"6403070": "Kec. Kenohan, Kutai Kartanegara",

		// Jakarta
		"IDNC001": "Jakarta Pusat",
		"3171010": "Kec. Gambir, Jakarta Pusat",
		"3171020": "Kec. Sawah Besar, Jakarta Pusat",
		"3171030": "Kec. Kemayoran, Jakarta Pusat",
		"3171040": "Kec. Senen, Jakarta Pusat",
		"3171050": "Kec. Cempaka Putih, Jakarta Pusat",
		"3171060": "Kec. Menteng, Jakarta Pusat",
		"3171070": "Kec. Tanah Abang, Jakarta Pusat",
		"3171080": "Kec. Johar Baru, Jakarta Pusat",

		"IDNC002": "Jakarta Utara",
		"3174010": "Kec. Penjaringan, Jakarta Utara",
		"3174020": "Kec. Pademangan, Jakarta Utara",
		"3174030": "Kec. Tanjung Priok, Jakarta Utara",
		"3174040": "Kec. Koja, Jakarta Utara",
		"3174050": "Kec. Kelapa Gading, Jakarta Utara",
		"3174060": "Kec. Cilincing, Jakarta Utara",

		"IDNC003": "Jakarta Selatan",
		"3173010": "Kec. Kebayoran Baru, Jakarta Selatan",
		"3173020": "Kec. Kebayoran Lama, Jakarta Selatan",
		"3173030": "Kec. Pesanggrahan, Jakarta Selatan",
		"3173040": "Kec. Cilandak, Jakarta Selatan",
		"3173050": "Kec. Pasar Minggu, Jakarta Selatan",
		"3173060": "Kec. Jagakarsa, Jakarta Selatan",
		"3173070": "Kec. Mampang Prapatan, Jakarta Selatan",
		"3173080": "Kec. Pancoran, Jakarta Selatan",
		"3173090": "Kec. Tebet, Jakarta Selatan",
		"3173100": "Kec. Setiabudi, Jakarta Selatan",

		"IDNC004": "Jakarta Barat",
		"3172010": "Kec. Tambora, Jakarta Barat",
		"3172020": "Kec. Taman Sari, Jakarta Barat",
		"3172030": "Kec. Cengkareng, Jakarta Barat",
		"3172040": "Kec. Grogol Petamburan, Jakarta Barat",
		"3172050": "Kec. Kebon Jeruk, Jakarta Barat",
		"3172060": "Kec. Kembangan, Jakarta Barat",
		"3172070": "Kec. Palmerah, Jakarta Barat",
		"3172080": "Kec. Kali Deres, Jakarta Barat",

		"IDNC005": "Jakarta Timur",
		"3175010": "Kec. Pasar Rebo, Jakarta Timur",
		"3175020": "Kec. Ciracas, Jakarta Timur",
		"3175030": "Kec. Cipayung, Jakarta Timur",
		"3175040": "Kec. Makasar, Jakarta Timur",
		"3175050": "Kec. Kramat Jati, Jakarta Timur",
		"3175060": "Kec. Jatinegara, Jakarta Timur",
		"3175070": "Kec. Duren Sawit, Jakarta Timur",
		"3175080": "Kec. Cakung, Jakarta Timur",
		"3175090": "Kec. Pulo Gadung, Jakarta Timur",
		"3175100": "Kec. Matraman, Jakarta Timur",

		// Surabaya
		"IDNC100": "Surabaya",
		"3578010": "Kec. Karang Pilang, Surabaya",
		"3578020": "Kec. Wonocolo, Surabaya",
		"3578030": "Kec. Rungkut, Surabaya",
		"3578040": "Kec. Wonokromo, Surabaya",
		"3578050": "Kec. Tegalsari, Surabaya",
		"3578060": "Kec. Sawahan, Surabaya",
		"3578070": "Kec. Genteng, Surabaya",
		"3578080": "Kec. Bubutan, Surabaya",
		"3578090": "Kec. Simokerto, Surabaya",
		"3578100": "Kec. Pabean Cantian, Surabaya",
		"3578110": "Kec. Semampir, Surabaya",
		"3578120": "Kec. Krembangan, Surabaya",
		"3578130": "Kec. Kenjeran, Surabaya",
		"3578140": "Kec. Lakarsantri, Surabaya",
		"3578150": "Kec. Benowo, Surabaya",
		"3578160": "Kec. Wiyung, Surabaya",
		"3578170": "Kec. Dukuh Pakis, Surabaya",
		"3578180": "Kec. Gayungan, Surabaya",
		"3578190": "Kec. Jambangan, Surabaya",
		"3578200": "Kec. Tenggilis Mejoyo, Surabaya",
		"3578210": "Kec. Gunung Anyar, Surabaya",
		"3578220": "Kec. Sukolilo, Surabaya",
		"3578230": "Kec. Bulak, Surabaya",
		"3578240": "Kec. Mulyorejo, Surabaya",
		"3578250": "Kec. Gubeng, Surabaya",
		"3578260": "Kec. Sukomanunggal, Surabaya",
		"3578270": "Kec. Tandes, Surabaya",
		"3578280": "Kec. Asemrowo, Surabaya",
		"3578290": "Kec. Pakal, Surabaya",
		"3578300": "Kec. Sambikerep, Surabaya",

		// Sidoarjo
		"IDNC101": "Sidoarjo",
		"3515010": "Kec. Sidoarjo, Sidoarjo",
		"3515020": "Kec. Buduran, Sidoarjo",
		"3515030": "Kec. Candi, Sidoarjo",
		"3515040": "Kec. Porong, Sidoarjo",
		"3515050": "Kec. Krembung, Sidoarjo",
		"3515060": "Kec. Tulangan, Sidoarjo",
		"3515070": "Kec. Tanggulangin, Sidoarjo",
		"3515080": "Kec. Jabon, Sidoarjo",
		"3515090": "Kec. Krian, Sidoarjo",
		"3515100": "Kec. Balongbendo, Sidoarjo",
		"3515110": "Kec. Wonoayu, Sidoarjo",
		"3515120": "Kec. Tarik, Sidoarjo",
		"3515130": "Kec. Prambon, Sidoarjo",
		"3515140": "Kec. Taman, Sidoarjo",
		"3515150": "Kec. Sukodono, Sidoarjo",
		"3515160": "Kec. Gedangan, Sidoarjo",
		"3515170": "Kec. Sedati, Sidoarjo",
		"3515180": "Kec. Waru, Sidoarjo",
	}

	if areaName, exists := areaNameMapping[areaID]; exists {
		log.Printf("Area name found: %s -> %s", areaID, areaName)
		return areaName
	}

	// Jika tidak ada mapping, kembalikan area ID sebagai fallback
	log.Printf("Area name NOT found for: %s, using area ID", areaID)
	return areaID
}

func GetPaginationLinks(config *AppConfig, params PaginationParams) (PaginationLinks, error) {
	// Inisialisasi slice untuk menampung daftar tautan halaman
	var links []PageLink

	// Hitung total halaman dengan membulatkan ke atas hasil pembagian total baris dengan baris per halaman
	totalPages := int32(math.Ceil(float64(params.TotalRows) / float64(params.PerPage)))

	// Loop untuk membuat daftar tautan halaman
	for i := 1; int32(i) <= totalPages; i++ {
		links = append(links, PageLink{
			// Nomor halaman
			Page: int32(i),

			// Tautan halaman menggunakan format URL dari konfigurasi aplikasi
			Url: fmt.Sprintf("%s/%s?page=%s", config.AppURL, params.Path, fmt.Sprint(i)),

			// Tandai apakah ini adalah halaman saat ini
			IsCurrentPage: int32(i) == params.CurrentPage,
		})
	}

	// Inisialisasi variabel untuk halaman berikutnya dan sebelumnya
	var nextPage int32
	var prevPage int32

	// Set halaman sebelumnya ke 1 sebagai default
	prevPage = 1

	// Set halaman berikutnya ke halaman terakhir
	nextPage = totalPages

	// Jika halaman saat ini lebih dari 2, halaman sebelumnya diset ke halaman sebelum halaman saat ini
	if params.CurrentPage > 2 {
		prevPage = params.CurrentPage - 1
	}

	// Jika halaman saat ini kurang dari total halaman, set halaman berikutnya ke halaman setelahnya
	if params.CurrentPage < totalPages {
		nextPage = params.CurrentPage + 1
	}

	// Kembalikan struktur PaginationLinks yang terisi lengkap
	return PaginationLinks{
		// Tautan halaman saat ini
		CurrentPage: fmt.Sprintf("%s/%s?page=%s", config.AppURL, params.Path, fmt.Sprint(params.CurrentPage)),

		// Tautan halaman berikutnya
		NextPage: fmt.Sprintf("%s/%s?page=%s", config.AppURL, params.Path, fmt.Sprint(nextPage)),

		// Tautan halaman sebelumnya
		PrevPage: fmt.Sprintf("%s/%s?page=%s", config.AppURL, params.Path, fmt.Sprint(prevPage)),
		// Total baris data
		TotalRows: params.TotalRows,
		// Total halaman yang dihitung
		TotalPages: totalPages,
		// Daftar tautan yang dihasilkan
		Links: links,
	}, nil
}

type ShippingFeeParams struct {
	Origin      string
	Destination string
	Weight      int
	Couriers    string
}

// CalculateShippingFeeBiteship mengirim permintaan POST ke API Biteship untuk menghitung biaya pengiriman
func (server *Server) CalculateShippingFeeBiteship(params ShippingFeeParams) ([]models.Pricing, error) {
	// DEBUG: Log what we're sending to Biteship
	apiKey := os.Getenv("API_BITESHIP")
	isTestingMode := strings.Contains(apiKey, "biteship_test")
	modeLabel := "🚀 PRODUCTION"
	if isTestingMode {
		modeLabel = "🧪 TESTING"
	}

	log.Printf("🚚 BITESHIP API REQUEST (%s MODE):", modeLabel)
	log.Printf("   Origin Area ID: %s", params.Origin)
	log.Printf("   Destination Area ID: %s", params.Destination)
	log.Printf("   Couriers: %s", params.Couriers)
	log.Printf("   Weight: %d grams", params.Weight)

	// Validate input parameters
	if params.Origin == "" {
		log.Printf("❌ ERROR: Origin area ID is empty")
		return nil, fmt.Errorf("origin area ID cannot be empty")
	}
	if params.Destination == "" {
		log.Printf("❌ ERROR: Destination area ID is empty")
		return nil, fmt.Errorf("destination area ID cannot be empty")
	}
	if params.Couriers == "" {
		log.Printf("❌ ERROR: Couriers is empty")
		return nil, fmt.Errorf("couriers cannot be empty")
	}

	// Membuat payload data permintaan sesuai format API Biteship
	payload := models.CourierRequest{
		OriginAreaID:      params.Origin,
		DestinationAreaID: params.Destination,
		Couriers:          params.Couriers,
		Items: []models.Item{
			{
				Name:        "Cart Items",                        // Nama item default
				Description: "Combined items from shopping cart", // Deskripsi item default
				Value:       100000,                              // Nilai item contoh
				Length:      10,                                  // Panjang item dalam cm
				Width:       10,                                  // Lebar item dalam cm
				Height:      10,                                  // Tinggi item dalam cm
				Weight:      params.Weight,                       // Berat item dari parameter
				Quantity:    1,                                   // Jumlah item default
			},
		},
	}

	// Mengonversi payload ke format JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err // Mengembalikan kesalahan jika proses marshaling gagal
	}

	// Menentukan URL endpoint API Biteship
	url := "https://api.biteship.com/v1/rates/couriers"

	// Membuat permintaan HTTP POST
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err // Mengembalikan kesalahan jika pembuatan permintaan gagal
	}

	// Menetapkan header permintaan
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("API_BITESHIP"))
	log.Printf("Authorization Header: %s", req.Header.Get("Authorization"))

	// Mengirim permintaan HTTP ke server
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err // Mengembalikan kesalahan jika permintaan gagal
	}
	defer resp.Body.Close() // Menutup body respons setelah selesai

	// Membaca isi body respons
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err // Mengembalikan kesalahan jika pembacaan body gagal
	}

	// Memeriksa status HTTP dari respons
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", string(body)) // Mengembalikan kesalahan jika status tidak OK
	}

	// DEBUG: Log response from Biteship
	log.Printf("📦 BITESHIP API RESPONSE: %s", string(body))

	// Mengurai data JSON dari respons
	var response models.CourierResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err // Mengembalikan kesalahan jika penguraian JSON gagal
	}

	// DEBUG: Log parsed response
	log.Printf("✅ BITESHIP PARSED DATA:")
	log.Printf("   Success: %v", response.Success)
	log.Printf("   Origin: %+v", response.Origin)
	log.Printf("   Destination: %+v", response.Destination)
	log.Printf("   Pricing options: %d", len(response.Pricing))

	// SPECIFIC DEBUG: Check if destination matches what we expect
	log.Printf("🔍 DESTINATION CHECK:")
	log.Printf("   Expected Destination Area ID: %s", params.Destination)
	log.Printf("   Actual Destination from Biteship: %+v", response.Destination)

	// Alert if there's a mismatch (Palaran should NOT return Sanga-sanga)
	if params.Destination == "IDNP15IDNC383IDND4548" {
		log.Printf("🎯 PALARAN REQUEST - Checking response...")
		log.Printf("   Destination City: %s", response.Destination.City)
		if response.Destination.City == "Sanga-sanga" {
			log.Printf("🚨 CRITICAL: Palaran request returned Sanga-sanga!")
		}
	}

	// Memeriksa keberhasilan respons dari API
	if !response.Success {
		return nil, errors.New(response.Message) // Mengembalikan pesan kesalahan dari API jika gagal
	}

	return response.Pricing, nil // Mengembalikan data harga pengiriman jika berhasil
}

// parseLtlng mengonversi string menjadi float64
func parseLtlng(latitudeStr string) (float64, error) {
	// Konversi string ke float64
	latitude, err := strconv.ParseFloat(latitudeStr, 64)
	if err != nil {
		return 0, fmt.Errorf("gagal mengonversi latitude: %v", err) // Mengembalikan kesalahan jika konversi gagal
	}

	return latitude, nil // Mengembalikan nilai latitude jika berhasil
}

// CalculateShippingFeeBiteshipInstant menghitung biaya pengiriman instan menggunakan API Biteship
// dengan parameter lokasi asal dan tujuan yang diberikan.
func (server *Server) CalculateShippingFeeBiteshipInstant(params ShippingFeeParams) ([]models.Pricing, error) {
	// Mengonversi koordinat asal ke latitude, jika terjadi kesalahan set nilai default
	latitudeDestination, err := parseLtlng(params.Origin)
	if err != nil {
		// Updated coordinates for Toko Shafirda - Jl. KH. Harun Nafsi No.106, Loa Janan Ilir
		latitudeDestination = -0.5262810043373423
	}

	// Mengonversi koordinat tujuan ke longitude, jika terjadi kesalahan set nilai default
	longitudeDestination, err := parseLtlng(params.Destination)
	if err != nil {
		// Updated coordinates for Toko Shafirda - Jl. KH. Harun Nafsi No.106, Loa Janan Ilir
		longitudeDestination = 117.13669626404219
	}

	// Membuat payload untuk permintaan API yang berisi data pengiriman
	payload := models.CourierInstantRequest{
		OriginLatitude:       -0.526313085327813,   // Latitude lokasi asal
		OriginLongitude:      117.13666900992393,   // Longitude lokasi asal
		DestinationLatitude:  latitudeDestination,  // Latitude lokasi tujuan
		DestinationLongitude: longitudeDestination, // Longitude lokasi tujuan
		Couriers:             "grab,gojek",         // Kurir yang digunakan
		Items: []models.Item{
			{
				Name:        "Shoes",                 // Nama item
				Description: "Black colored size 45", // Deskripsi item
				Value:       199000,                  // Nilai item dalam rupiah
				Length:      30,                      // Panjang item dalam cm
				Width:       15,                      // Lebar item dalam cm
				Height:      20,                      // Tinggi item dalam cm
				Weight:      200,                     // Berat item dalam gram
				Quantity:    2,                       // Jumlah item
			},
		},
	}

	// Mengubah payload menjadi format JSON untuk dikirim melalui API
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err // Mengembalikan error jika gagal membuat JSON
	}

	// URL endpoint API Biteship untuk menghitung biaya pengiriman
	url := "https://api.biteship.com/v1/rates/couriers"

	// Membuat permintaan HTTP POST ke API
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err // Mengembalikan error jika gagal membuat permintaan
	}

	// Menambahkan header ke permintaan HTTP
	req.Header.Set("Content-Type", "application/json") // Menentukan tipe konten sebagai JSON
	req.Header.Set("Authorization", "Bearer "+os.Getenv("API_BITESHIP"))

	// Membuat klien HTTP dan mengirim permintaan
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err // Mengembalikan error jika permintaan gagal
	}
	defer resp.Body.Close() // Menutup body respons setelah selesai

	// Membaca respons dari API
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err // Mengembalikan error jika gagal membaca respons
	}

	// Mengecek status kode HTTP untuk memastikan respons sukses
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", string(body)) // Mengembalikan error jika API mengembalikan status selain 200 OK
	}

	// Mengurai respons JSON ke dalam struktur data `CourierResponse`
	var response models.CourierResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err // Mengembalikan error jika gagal mengurai JSON
	}

	// Mengecek apakah API mengembalikan status sukses
	if !response.Success {
		return nil, fmt.Errorf("API returned error: %s", response.Message) // Mengembalikan error jika API mengembalikan pesan error
	}

	// Mengembalikan data harga pengiriman yang berhasil dihitung
	return response.Pricing, nil
}

// perubahan API create biteship
// CreateBiteshipOrder membuat pesanan baru menggunakan API Biteship
func (server *Server) CreateBiteshipOrder(params models.OrderParams) (*models.OrderResponse, error) {
	// Mengubah parameter pesanan menjadi format JSON untuk dikirimkan ke API
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order params: %w", err) // Mengembalikan error jika gagal melakukan serialisasi JSON
	}

	// URL endpoint API untuk membuat pesanan
	url := "https://api.biteship.com/v1/orders"

	// Membuat permintaan HTTP POST dengan data JSON
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err) // Mengembalikan error jika gagal membuat permintaan
	}

	// Menambahkan header untuk tipe konten dan otorisasi
	req.Header.Set("Content-Type", "application/json")                   // Menentukan bahwa data yang dikirim adalah JSON
	req.Header.Set("Authorization", "Bearer "+os.Getenv("API_BITESHIP")) // Token API untuk autentikasi

	// Membuat klien HTTP dan mengirim permintaan
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err) // Mengembalikan error jika permintaan gagal
	}
	defer resp.Body.Close() // Menutup body respons setelah selesai untuk menghindari kebocoran sumber daya

	// Membaca respons dari API
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err) // Mengembalikan error jika gagal membaca body respons
	}

	// Memeriksa kode status HTTP untuk memastikan bahwa permintaan berhasil
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", string(body)) // Mengembalikan error jika API mengembalikan kode status selain 200 OK
	}

	// Mengurai data JSON dari respons ke dalam struktur data `OrderResponse`
	var response models.OrderResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err) // Mengembalikan error jika gagal mengurai JSON
	}

	// Memeriksa apakah respons dari API menunjukkan kesuksesan
	if !response.Success {
		return nil, errors.New(response.Message) // Mengembalikan error jika API mengembalikan status gagal
	}

	// Mengembalikan respons yang berhasil berupa data `OrderResponse`
	return &response, nil
}

package controllers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/unrolled/render"

	"github.com/google/uuid"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"

	"github.com/gieart87/gotoko/app/consts"
	"github.com/gieart87/gotoko/app/core/session/auth"
	"github.com/gieart87/gotoko/app/core/session/flash"

	"github.com/gieart87/gotoko/app/models"
	"github.com/shopspring/decimal"
)

type CheckoutRequest struct {
	Cart            *models.Cart
	ShippingFee     *ShippingFee
	ShippingAddress *ShippingAddress
}

type ShippingFee struct {
	Courier     string
	PackageName string
	Fee         float64
}

type ShippingAddress struct {
	FirstName  string
	LastName   string
	CityID     string
	ProvinceID string
	Address1   string
	Address2   string
	Phone      string
	Email      string
	PostCode   string
}

func (server *Server) Checkout(w http.ResponseWriter, r *http.Request) {
	user := auth.CurrentUser(server.DB, w, r)
	log.Printf("Checkout started for user: %v", user)

	// Debug form values
	log.Printf("Form values: courier=%s, shipping_fee=%s, city_id=%s, cour_type=%s",
		r.FormValue("courier"), r.FormValue("shipping_fee"), r.FormValue("city_id"), r.FormValue("cour_type"))

	shippingCost, err := server.getSelectedShippingCost(w, r)
	if err != nil {
		log.Printf("Shipping cost calculation failed: %v", err)
		flash.SetFlash(w, r, "error", "Proses checkout gagal: "+err.Error())
		http.Redirect(w, r, "/carts", http.StatusSeeOther)
		return
	}
	log.Printf("Shipping cost calculated: %f", shippingCost)

	cartID := GetShoppingCartID(w, r)
	cart, _ := GetShoppingCart(server.DB, cartID)

	checkoutRequest := &CheckoutRequest{
		Cart: cart,
		ShippingFee: &ShippingFee{
			Courier:     r.FormValue("courier"),
			PackageName: r.FormValue("shipping_fee"),
			Fee:         shippingCost,
		},
		ShippingAddress: &ShippingAddress{
			FirstName:  r.FormValue("first_name"),
			LastName:   r.FormValue("last_name"),
			CityID:     r.FormValue("city_id"),
			ProvinceID: r.FormValue("province_id"),
			Address1:   r.FormValue("address1"),
			Address2:   r.FormValue("address2"),
			Phone:      r.FormValue("phone"),
			Email:      r.FormValue("email"),
			PostCode:   r.FormValue("post_code"),
		},
	}
	order, err := server.SaveOrder(user, checkoutRequest)
	if err != nil {
		flash.SetFlash(w, r, "error", "Proses checkout gagal")
		http.Redirect(w, r, "/carts", http.StatusSeeOther)
		return
	}

	var orderItems []models.OrderItemTestimonials

	if len(checkoutRequest.Cart.CartItems) > 0 {
		for _, cartItem := range checkoutRequest.Cart.CartItems {
			orderItems = append(orderItems, models.OrderItemTestimonials{
				Name:        cartItem.Product.Name,
				Description: cartItem.Product.ShortDescription,
				Value:       int(cartItem.Product.Price.IntPart()),
				Quantity:    cartItem.Qty,
				Weight:      int(cartItem.Product.Weight.IntPart()),
			})
		}
	}

	cour_type := r.FormValue("cour_type")
	var response *models.OrderResponse
	if cour_type == "instant" {
		latitudeStr := r.FormValue("latitude")
		longitudeStr := r.FormValue("longitude")
		if latitudeStr == "" || longitudeStr == "" {
			flash.SetFlash(w, r, "error", "Pilih Latitude Longitude!")
			http.Redirect(w, r, "/carts", http.StatusSeeOther)
			return
		}

		latitude, err := strconv.ParseFloat(latitudeStr, 64)
		if err != nil {
			flash.SetFlash(w, r, "error", "Error Konversi Latitude")
			http.Redirect(w, r, "/carts", http.StatusSeeOther)
			return
		}

		longitude, err := strconv.ParseFloat(longitudeStr, 64)
		if err != nil {
			flash.SetFlash(w, r, "error", "Error Konversi Longitude")
			http.Redirect(w, r, "/carts", http.StatusSeeOther)
			return
		}

		//parameter untuk membuat order  biteship
		params := models.OrderParams{
			ShipperContactName:      checkoutRequest.ShippingFee.Courier,
			ShipperContactPhone:     checkoutRequest.ShippingFee.Courier,
			ShipperContactEmail:     checkoutRequest.ShippingFee.Courier,
			ShipperOrganization:     checkoutRequest.ShippingFee.Courier,
			OriginContactName:       "Wahyu Bahri Irsandy",
			OriginContactPhone:      "08115992185",
			OriginAddress:           "Jl. KH. Harun Nafsi No.106, RT.22, Rapak Dalam, Kec. Loa Janan Ilir, Kota Samarinda, Kalimantan Timur",
			OriginNote:              "Toko Shafirda",
			OriginCoordinate:        models.Coordinate{-0.526313085327813, 117.13666900992393},
			DestinationContactName:  checkoutRequest.ShippingAddress.FirstName + checkoutRequest.ShippingAddress.LastName,
			DestinationContactPhone: checkoutRequest.ShippingAddress.Phone,
			DestinationContactEmail: checkoutRequest.ShippingAddress.Email,
			DestinationAddress:      checkoutRequest.ShippingAddress.Address1,
			DestinationNote:         checkoutRequest.ShippingAddress.Address2,
			DestinationCoordinate:   models.Coordinate{latitude, longitude},
			CourierCompany:          "grab",
			CourierType:             "instant",
			CourierInsurance:        50000,
			DeliveryType:            "now",
			OrderNote:               "Please be careful",
			Items:                   orderItems,
		}

		//memanggil handler create biteship
		response, err = server.CreateBiteshipOrder(params)
		if err != nil {
			log.Fatalf("Failed to create order: %v", err)
		}

		log.Printf("Order created successfully: %+v", response)
	} else if cour_type == "pickup" {

	} else {
		params := models.OrderParams{
			ShipperContactName:      checkoutRequest.ShippingFee.Courier,
			ShipperContactPhone:     checkoutRequest.ShippingFee.Courier,
			ShipperContactEmail:     checkoutRequest.ShippingFee.Courier,
			ShipperOrganization:     checkoutRequest.ShippingFee.Courier,
			OriginContactName:       "Wahyu Bahri Irsandy",
			OriginContactPhone:      "08115992185",
			OriginPostalCode:        75131,
			OriginAddress:           "Jl. KH. Harun Nafsi No.106, RT.22, Rapak Dalam, Kec. Loa Janan Ilir, Kota Samarinda, Kalimantan Timur ",
			OriginNote:              "Toko Shafirda",
			DestinationContactName:  checkoutRequest.ShippingAddress.FirstName + checkoutRequest.ShippingAddress.LastName,
			DestinationContactPhone: checkoutRequest.ShippingAddress.Phone,
			DestinationContactEmail: checkoutRequest.ShippingAddress.Email,
			DestinationAddress:      checkoutRequest.ShippingAddress.Address1,
			DestinationNote:         checkoutRequest.ShippingAddress.Address2,
			DestinationPostalCode:   checkoutRequest.ShippingAddress.PostCode,
			CourierCompany:          r.FormValue("courier"),
			CourierType:             "reg",
			CourierInsurance:        50000,
			DeliveryType:            "now",
			OrderNote:               "Please be careful",
			Items:                   orderItems,
		}

		//memanggil handler create biteship
		response, err = server.CreateBiteshipOrder(params)
		if err != nil {
			log.Fatalf("Failed to create order: %v", err)
		}

		log.Printf("Order created successfully: %+v", response)
	}

	ClearCart(server.DB, cartID)

	var trackingID = ""
	if response != nil {
		trackingID = response.Courier.TrackingID
	}
	message := fmt.Sprintf("Data order berhasil disimpan %s", trackingID)

	flash.SetFlash(w, r, "success", message)
	http.Redirect(w, r, "/orders/"+order.ID, http.StatusSeeOther)
}

func (server *Server) ShowOrder(w http.ResponseWriter, r *http.Request) {
	render := render.New(render.Options{
		Layout:     "layout",
		Extensions: []string{".html", ".tmpl"},
	})

	vars := mux.Vars(r)

	if vars["id"] == "" {
		http.Redirect(w, r, "/products", http.StatusSeeOther)
		return
	}

	orderModel := models.Order{}
	order, err := orderModel.FindByID(server.DB, vars["id"])
	if err != nil {
		http.Redirect(w, r, "/products", http.StatusSeeOther)
		return
	}

	_ = render.HTML(w, http.StatusOK, "show_order", map[string]interface{}{
		"order":   order,
		"success": flash.GetFlash(w, r, "success"),
		"user":    auth.CurrentUser(server.DB, w, r),
	})
}

func (server *Server) getSelectedShippingCost(w http.ResponseWriter, r *http.Request) (float64, error) {
	default_location := os.Getenv("API_BITESHIP_SAMARINDA_LOCATION")

	destination := r.FormValue("city_id") // Gunakan city_id dari user input
	cour_type := r.FormValue("cour_type")
	courier := r.FormValue("courier")
	shippingFeeSelected := r.FormValue("shipping_fee")

	log.Printf("Checkout shipping params: city_id=%s, cour_type=%s, courier=%s, shipping_fee=%s",
		destination, cour_type, courier, shippingFeeSelected)

	cartID := GetShoppingCartID(w, r)
	cart, _ := GetShoppingCart(server.DB, cartID)

	// Handle pickup - no shipping cost
	if cour_type == "pickup" || courier == "pickup" {
		log.Printf("Pickup selected - no shipping cost")
		return 0, nil
	}

	if destination == "" {
		// Fallback ke default location jika city_id kosong
		destination = default_location
		if destination == "" {
			return 0, errors.New("invalid destination: no city_id provided and no default location configured")
		}
	}

	var shippingFeeOptions []models.Pricing
	var err error

	if cour_type == "instant" {
		// Untuk instant delivery, gunakan koordinat latitude/longitude
		latitudeStr := r.FormValue("latitude")
		longitudeStr := r.FormValue("longitude")

		if latitudeStr == "" || longitudeStr == "" {
			return 0, errors.New("latitude and longitude required for instant delivery")
		}

		shippingFeeOptions, err = server.CalculateShippingFeeBiteshipInstant(ShippingFeeParams{
			Origin:      latitudeStr,  // Gunakan latitude dari form
			Destination: longitudeStr, // Gunakan longitude dari form
			Weight:      cart.TotalWeight,
			Couriers:    courier,
		})
		log.Printf("Instant delivery calculation with lat: %s, lng: %s", latitudeStr, longitudeStr)
	} else if cour_type == "pickup" {
		// Untuk pickup, tidak ada shipping cost
		return 0, nil
	} else {
		// Untuk regular delivery, gunakan area ID yang sudah dikonversi
		log.Printf("Regular delivery calculation for: %s", cour_type)
		destinationAreaID := convertCityIDToBiteshipAreaID(destination)
		shippingFeeOptions, err = server.CalculateShippingFeeBiteship(ShippingFeeParams{
			Origin:      default_location,  // Origin tetap default (toko)
			Destination: destinationAreaID, // Destination menggunakan area ID yang benar
			Weight:      cart.TotalWeight,
			Couriers:    courier,
		})
		log.Printf("Regular delivery calculation: %s -> %s", default_location, destinationAreaID)
	}

	if err != nil {
		log.Printf("Shipping calculation error: %v", err)
		return 0, errors.New("failed shipping calculation")
	}

	var selectedShipping models.Pricing
	var found bool

	// Mencari opsi shipping yang dipilih dengan lebih akurat
	for _, shippingOption := range shippingFeeOptions {
		log.Printf("Comparing: '%s' with '%s'", shippingOption.CourierName, shippingFeeSelected)
		if shippingOption.CourierName == shippingFeeSelected {
			selectedShipping = shippingOption
			found = true
			break // Gunakan break, bukan continue
		}
	}

	if !found {
		log.Printf("Selected shipping option not found: %s", shippingFeeSelected)
		return 0, errors.New("selected shipping option not found")
	}

	log.Printf("Selected shipping cost: %d", selectedShipping.Price)
	return float64(selectedShipping.Price), nil
}

func (server *Server) SaveOrder(user *models.User, r *CheckoutRequest) (*models.Order, error) {
	var orderItems []models.OrderItem

	orderID := uuid.New().String()
	var paymentURL string
	var err error
	// if r.ShippingFee.Courier != "pickup" {
	paymentURL, err = server.createPaymentURL(user, r, orderID)
	if err != nil {
		return nil, err
	}
	// }

	if len(r.Cart.CartItems) > 0 {
		for _, cartItem := range r.Cart.CartItems {
			orderItems = append(orderItems, models.OrderItem{
				ProductID:       cartItem.ProductID,
				Qty:             cartItem.Qty,
				Unit:            cartItem.Unit, // Include unit dari cart item
				BasePrice:       cartItem.BasePrice,
				BaseTotal:       cartItem.BaseTotal,
				TaxAmount:       cartItem.TaxAmount,
				TaxPercent:      cartItem.TaxPercent,
				DiscountAmount:  cartItem.DiscountAmount,
				DiscountPercent: cartItem.DiscountPercent,
				SubTotal:        cartItem.SubTotal,
				Sku:             cartItem.Product.Sku,
				Name:            cartItem.Product.Name,
				Weight:          cartItem.Product.Weight,
			})
		}
	}

	orderCustomer := &models.OrderCustomer{
		UserID:     user.ID,
		FirstName:  r.ShippingAddress.FirstName,
		LastName:   r.ShippingAddress.LastName,
		CityID:     r.ShippingAddress.CityID,
		ProvinceID: r.ShippingAddress.ProvinceID,
		Address1:   r.ShippingAddress.Address1,
		Address2:   r.ShippingAddress.Address2,
		Phone:      r.ShippingAddress.Phone,
		Email:      r.ShippingAddress.Email,
		PostCode:   r.ShippingAddress.PostCode,
	}

	// Hitung grand total termasuk shipping cost
	shippingCostDecimal := decimal.NewFromFloat(r.ShippingFee.Fee)
	grandTotalWithShipping := r.Cart.GrandTotal.Add(shippingCostDecimal)

	orderData := &models.Order{
		ID:                  orderID,
		UserID:              user.ID,
		OrderItems:          orderItems,
		OrderCustomer:       orderCustomer,
		Status:              0,
		OrderDate:           time.Now(),
		PaymentDue:          time.Now().AddDate(0, 0, 7),
		PaymentStatus:       consts.OrderPaymentStatusUnpaid,
		BaseTotalPrice:      r.Cart.BaseTotalPrice,
		TaxAmount:           r.Cart.TaxAmount,
		TaxPercent:          r.Cart.TaxPercent,
		DiscountAmount:      r.Cart.DiscountAmount,
		DiscountPercent:     r.Cart.DiscountPercent,
		ShippingCost:        shippingCostDecimal,
		GrandTotal:          grandTotalWithShipping,
		ShippingCourier:     r.ShippingFee.Courier,
		ShippingServiceName: r.ShippingFee.PackageName,
		PaymentToken:        sql.NullString{String: paymentURL, Valid: true},
	}

	orderModel := models.Order{}
	order, err := orderModel.CreateOrder(server.DB, orderData)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (server *Server) createPaymentURL(user *models.User, r *CheckoutRequest, orderID string) (string, error) {
	midtransServerKey := os.Getenv("API_MIDTRANS_SERVER_KEY")

	midtrans.ServerKey = midtransServerKey

	var enabledPaymentTypes []snap.SnapPaymentType

	enabledPaymentTypes = append(enabledPaymentTypes, snap.AllSnapPaymentType...)

	// Hitung grand total termasuk shipping cost untuk payment
	shippingCostDecimal := decimal.NewFromFloat(r.ShippingFee.Fee)
	grandTotalWithShipping := r.Cart.GrandTotal.Add(shippingCostDecimal)

	snapRequest := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderID,
			GrossAmt: int64(grandTotalWithShipping.IntPart()),
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: user.FirstName,
			LName: user.LastName,
			Email: user.Email,
			Phone: user.Phone,
		},
		EnabledPayments: enabledPaymentTypes,
	}

	snapResponse, err := snap.CreateTransaction(snapRequest)
	if err != nil {
		return "", err
	}

	return snapResponse.RedirectURL, nil
}

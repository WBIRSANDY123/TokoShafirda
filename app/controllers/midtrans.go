package controllers

import (
	"encoding/json"
	"github.com/gieart87/gotoko/app/models"
	"github.com/shopspring/decimal"
	"net/http"
)

func (server *Server) Midtrans(w http.ResponseWriter, r *http.Request) {
	var paymentNotification models.MidtransNotification

	// Mendekode payload JSON dari permintaan masuk.
	err := json.NewDecoder(r.Body).Decode(&paymentNotification)
	if err != nil {
		// Jika terjadi kesalahan saat mendekode, kembalikan respons HTTP 400.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		res := Result{Code: http.StatusBadRequest, Message: err.Error()}
		response, _ := json.Marshal(res)
		w.Write(response)
		return
	}
	defer r.Body.Close()

	// Memvalidasi signature key dari payload Midtrans.
	err = validateSignatureKey(&paymentNotification)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		res := Result{Code: http.StatusForbidden, Message: err.Error()}
		response, _ := json.Marshal(res)
		w.Write(response)
		return
	}

	// Mencari pesanan berdasarkan ID.
	orderModel := models.Order{}
	order, err := orderModel.FindByID(server.DB, paymentNotification.OrderID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		res := Result{Code: http.StatusForbidden, Message: err.Error()}
		response, _ := json.Marshal(res)
		w.Write(response)
		return
	}

	// Memeriksa apakah pesanan sudah dibayar.
	if order.IsPaid() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		res := Result{Code: http.StatusForbidden, Message: "Already paid before."}
		response, _ := json.Marshal(res)
		w.Write(response)
		return
	}

	// Membuat pembayaran baru.
	paymentModel := models.Payment{}
	amount, _ := decimal.NewFromString(paymentNotification.GrossAmount)
	jsonPayload, _ := json.Marshal(paymentNotification)
	payload := (*json.RawMessage)(&jsonPayload)

	_, err = paymentModel.CreatePayment(server.DB, &models.Payment{
		OrderID:           order.ID,
		Amount:            amount,
		TransactionID:     paymentNotification.TransactionID,
		TransactionStatus: paymentNotification.TransactionStatus,
		Payload:           payload,
		PaymentType:       paymentNotification.PaymentType,
	})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		res := Result{Code: http.StatusBadRequest, Message: "Could not process the payment."}
		response, _ := json.Marshal(res)
		w.Write(response)
		return
	}

	// Memeriksa apakah pembayaran berhasil.
	if isPaymentSuccess(&paymentNotification) {
		err = order.MarkAsPaid(server.DB)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			res := Result{Code: http.StatusBadRequest, Message: "Could not process the payment."}
			response, _ := json.Marshal(res)
			w.Write(response)
			return
		}
	}

	// Mengembalikan respons sukses.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	res := Result{Code: http.StatusOK, Message: "Payment saved."}
	response, _ := json.Marshal(res)
	w.Write(response)
}

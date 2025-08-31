package controllers

import (
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/shopspring/decimal"

	"github.com/midtrans/midtrans-go/snap"

	"github.com/gieart87/gotoko/app/consts"
	"github.com/gieart87/gotoko/app/models"
)

func (server *Server) MidtransNotification(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[DEBUG] MidtransNotification endpoint called")
	fmt.Printf("[DEBUG] Request method: %s\n", r.Method)
	fmt.Printf("[DEBUG] Request URL: %s\n", r.URL.String())
	fmt.Printf("[DEBUG] Request headers: %+v\n", r.Header)
	fmt.Printf("[DEBUG] Content-Type: %s\n", r.Header.Get("Content-Type"))
	fmt.Printf("[DEBUG] Content-Length: %s\n", r.Header.Get("Content-Length"))

	var paymentNotification models.MidtransNotification

	err := json.NewDecoder(r.Body).Decode(&paymentNotification)
	if err != nil {
		fmt.Printf("[ERROR] Failed to decode JSON payload: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		res := Result{Code: http.StatusBadRequest, Message: err.Error()}
		response, _ := json.Marshal(res)
		w.Write(response)
		return
	}
	defer r.Body.Close()

	fmt.Printf("[DEBUG] Decoded payload: %+v\n", paymentNotification)

	err = validateSignatureKey(&paymentNotification)
	if err != nil {
		fmt.Printf("[ERROR] Signature validation failed: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		res := Result{Code: http.StatusForbidden, Message: err.Error()}
		response, _ := json.Marshal(res)
		w.Write(response)
		return
	}
	fmt.Println("[DEBUG] Signature validation passed")

	orderModel := models.Order{}
	order, err := orderModel.FindByID(server.DB, paymentNotification.OrderID)
	if err != nil {
		fmt.Printf("[ERROR] Order lookup failed for ID %s: %v\n", paymentNotification.OrderID, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		res := Result{Code: http.StatusForbidden, Message: err.Error()}
		response, _ := json.Marshal(res)
		w.Write(response)
		return
	}
	fmt.Printf("[DEBUG] Order found: ID=%d, Status=%s\n", order.ID, order.Status)

	if order.IsPaid() {
		fmt.Printf("[DEBUG] Order %d is already paid\n", order.ID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		res := Result{Code: http.StatusForbidden, Message: "Already paid before."}
		response, _ := json.Marshal(res)
		w.Write(response)
		return
	}

	paymentModel := models.Payment{}
	amount, _ := decimal.NewFromString(paymentNotification.GrossAmount)
	jsonPayload, _ := json.Marshal(paymentNotification)
	payload := (*json.RawMessage)(&jsonPayload)

	payment, err := paymentModel.CreatePayment(server.DB, &models.Payment{
		OrderID:           order.ID,
		Amount:            amount,
		TransactionID:     paymentNotification.TransactionID,
		TransactionStatus: paymentNotification.TransactionStatus,
		Payload:           payload,
		PaymentType:       paymentNotification.PaymentType,
	})
	if err != nil {
		fmt.Printf("[ERROR] Payment creation failed: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		res := Result{Code: http.StatusBadRequest, Message: "Could not process the payment."}
		response, _ := json.Marshal(res)
		w.Write(response)
		return
	}
	fmt.Printf("[DEBUG] Payment created: ID=%d, Amount=%s, Status=%s\n", payment.ID, amount.String(), paymentNotification.TransactionStatus)

	if isPaymentSuccess(&paymentNotification) {
		fmt.Printf("[DEBUG] Payment is successful, marking order as paid\n")
		err = order.MarkAsPaid(server.DB)
		if err != nil {
			fmt.Printf("[ERROR] Failed to mark order as paid: %v\n", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			res := Result{Code: http.StatusBadRequest, Message: "Could not process the payment."}
			response, _ := json.Marshal(res)
			w.Write(response)
			return
		}
		fmt.Printf("[DEBUG] Order %d successfully marked as paid\n", order.ID)
	} else {
		fmt.Printf("[DEBUG] Payment not successful: Status=%s, FraudStatus=%s, PaymentType=%s\n",
			paymentNotification.TransactionStatus, paymentNotification.FraudStatus, paymentNotification.PaymentType)
	}

	fmt.Println("[DEBUG] Sending success response")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	res := Result{Code: http.StatusOK, Message: "Payment saved."}
	response, _ := json.Marshal(res)
	w.Write(response)
}

func isPaymentSuccess(payload *models.MidtransNotification) bool {
	paymentStatus := false
	if payload.PaymentType == string(snap.PaymentTypeCreditCard) {
		paymentStatus = (payload.TransactionStatus == consts.PaymentStatusCapture) &&
			(payload.FraudStatus == consts.FraudStatusAccept)
	} else {
		paymentStatus = (payload.TransactionStatus == consts.PaymentStatusSettlement) &&
			(payload.FraudStatus == consts.FraudStatusAccept)
	}
	return paymentStatus
}

func validateSignatureKey(payload *models.MidtransNotification) error {
	environment := os.Getenv("APP_ENV")
	if environment == "development" {
		return nil
	}

	signaturePayload := payload.OrderID + payload.StatusCode +
		payload.GrossAmount + os.Getenv("API_MIDTRANS_SERVER_KEY")
	sha512Value := sha512.New()
	sha512Value.Write([]byte(signaturePayload))

	signatureKey := fmt.Sprintf("%x", sha512Value.Sum(nil))
	if signatureKey != payload.SignatureKey {
		return errors.New("invalid signature key")
	}
	return nil
}

func (server *Server) PaymentTest(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[DEBUG] PaymentTest endpoint called - Method: %s, URL: %s\n", r.Method, r.URL.String())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status":    "success",
		"message":   "Test endpoint working",
		"method":    r.Method,
		"timestamp": fmt.Sprintf("%v", r.Header.Get("Date")),
	}
	json.NewEncoder(w).Encode(response)
}

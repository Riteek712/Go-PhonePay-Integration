package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

var (
	PayEndpoint = "/pg/v1/pay"
	PhonePeURL  = os.Getenv("PHONEPE_TEST_HOST_URL")
	MerchantId  = os.Getenv("PHONEPE_MERCHANT_ID")
	SaltIndex   = os.Getenv("PHONEPE_KEY_API_INDEX")
	SaltKey     = os.Getenv("PHONEPE_KEY_API_VALUE")
)

func (s *Server) RegisterRoutes() http.Handler {
	r := httprouter.New()
	r.HandlerFunc(http.MethodGet, "/", s.HelloWorldHandler)

	r.HandlerFunc(http.MethodGet, "/health", s.healthHandler)

	r.HandlerFunc(http.MethodGet, "/pay", s.payHandler)

	r.HandlerFunc(http.MethodGet, "/redirect-url/", s.RedirectHandler)

	return r
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World from PhonePe!"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResp, err := json.Marshal(s.db.Health())

	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

type RequestData struct {
	MerchantID            string `json:"merchantId"`
	MerchantTransactionID string `json:"merchantTransactionId"`
	MerchantUserID        string `json:"merchantUserId"`
	Amount                int64  `json:"amount"`
	RedirectURL           string `json:"redirectUrl"`
	RedirectMode          string `json:"redirectMode"`
	CallbackURL           string `json:"callbackUrl"`
	MobileNumber          string `json:"mobileNumber"`
	PaymentInstrument     struct {
		Type string `json:"type"`
	} `json:"paymentInstrument"`
}

// GenerateXVerify generates the X-Verify header value
func GenerateXVerify(base64Payload string) string {
	data := base64Payload + PayEndpoint + SaltKey
	checksum := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x###%s", checksum, SaltIndex)
}

type ResponseData struct {
	Message string `json:"message"`
}
type PaymentResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		MerchantID            string `json:"merchantId"`
		MerchantTransactionID string `json:"merchantTransactionId"`
		InstrumentResponse    struct {
			Type         string `json:"type"`
			RedirectInfo struct {
				URL    string `json:"url"`
				Method string `json:"method"`
			} `json:"redirectInfo"`
		} `json:"instrumentResponse"`
	} `json:"data"`
}

// http://localhost:8080/pay
func (s *Server) payHandler(w http.ResponseWriter, r *http.Request) {
	// if r.Method != http.MethodPost {
	// 	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	// 	return
	// }
	// var reqBody RequestData
	// err := json.NewDecoder(r.Body).Decode(&reqBody)
	// if err != nil {
	// 	http.Error(w, "Invalid request body", http.StatusBadRequest)
	// 	return
	// }
	merchantTransactionID := uuid.New().String()
	userID := "123242"
	fmt.Println("merchantTransactionID")
	fmt.Println(merchantTransactionID)
	reqBody := RequestData{
		MerchantID:            MerchantId,
		MerchantTransactionID: merchantTransactionID,
		MerchantUserID:        userID,
		Amount:                3000,                            // Amount in Paise
		RedirectURL:           "http://localhost:5174/phonepe", //fmt.Sprintf("http://localhost:8080/redirect-url/txId = %s", merchantTransactionID), // Provide a valid redirect URL
		RedirectMode:          "REDIRECT",
		CallbackURL:           "http://localhost:8080/callback-url", // Provide a valid callback URL
		MobileNumber:          "9999999999",
		PaymentInstrument: struct {
			Type string `json:"type"`
		}{
			Type: "PAY_PAGE",
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		http.Error(w, "failed to marshal payment request", http.StatusBadRequest)
		return
	}
	fmt.Println("payload")
	fmt.Println(payload)

	// Encode the JSON payload to Base64
	base64Payload := base64.StdEncoding.EncodeToString(payload)
	fmt.Println("base64")
	fmt.Println(base64Payload)
	// Generate X-Verify header
	xVerify := GenerateXVerify(base64Payload)
	fmt.Println("xVerify")
	fmt.Println(xVerify)
	// Prepare the HTTP request
	requestBody := fmt.Sprintf(`{"request":"%s"}`, base64Payload)
	req, err := http.NewRequest("POST", PhonePeURL+PayEndpoint, strings.NewReader(requestBody))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create request: %v", err), http.StatusBadRequest)
		return

	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-VERIFY", xVerify)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to send request: %v", err), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read response: %v", err), http.StatusBadRequest)
		return
	}

	// Unmarshal the response
	var paymentResp PaymentResponse
	if err := json.Unmarshal(body, &paymentResp); err != nil {
		http.Error(w, fmt.Sprintf("failed to unmarshal response %v", err), http.StatusBadRequest)
		return
	}
	jsonResponse, err := json.Marshal(paymentResp)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	fmt.Println(paymentResp)
	fmt.Println()
	fmt.Println(jsonResponse)
	// w.WriteHeader(http.StatusOK)
	// w.Write(jsonResponse)
	http.Redirect(w, r, paymentResp.Data.InstrumentResponse.RedirectInfo.URL, http.StatusFound)
	return
}
func (s *Server) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	userID := parts[2]

	// For example, return user info as a plain text response (replace this with actual logic)
	userInfo := fmt.Sprintf("Merchant Transaction ID: %s", userID)

	// Send the response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(userInfo))
}

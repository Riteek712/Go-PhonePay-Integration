package tests

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mock database for unit testing
type MockDB struct {
	HealthFunc func() map[string]string
}

func (db *MockDB) Health() map[string]string {
	return db.HealthFunc()
}

// Mock server for unit testing
type MockServer struct {
	*Server
}

func TestHelloWorldHandler(t *testing.T) {
	// Initialize the server
	s := &MockServer{
		Server: &Server{},
	}

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HelloWorldHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	expected := `{"message":"Hello World from PhonePe!"}`
	assert.JSONEq(t, expected, rr.Body.String())
}

func TestHealthHandler(t *testing.T) {
	// Mock db.Health() response
	mockDB := &MockDB{
		HealthFunc: func() map[string]string {
			return map[string]string{"status": "healthy"}
		},
	}

	// Initialize the server
	s := &MockServer{
		Server: &Server{
			db: mockDB,
		},
	}

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.healthHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	expected := `{"status":"healthy"}`
	assert.JSONEq(t, expected, rr.Body.String())
}

func TestPayHandler(t *testing.T) {
	// Initialize the server with mock dependencies
	s := &MockServer{
		Server: &Server{
			db: &MockDB{
				HealthFunc: func() map[string]string {
					return map[string]string{"status": "healthy"}
				},
			},
		},
	}

	// Mock HTTP Client and server response
	mockClient := &http.Client{}
	mockClient.Do = func(req *http.Request) (*http.Response, error) {
		response := `{
			"success": true,
			"code": "200",
			"message": "Success",
			"data": {
				"merchantId": "mockMerchantId",
				"merchantTransactionId": "mockTransactionId",
				"instrumentResponse": {
					"type": "PAY_PAGE",
					"redirectInfo": {
						"url": "http://example.com/redirect",
						"method": "POST"
					}
				}
			}
		}`

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte(response))),
			Header:     make(http.Header),
		}, nil
	}

	// Replace the client with our mock
	// Ensure you have a way to inject the mockClient into your Server instance
	// This typically involves adding a Client field to the Server struct and setting it here

	req, _ := http.NewRequest(http.MethodGet, "/pay", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.payHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)
	assert.Equal(t, "http://example.com/redirect", rr.Header().Get("Location"))
}

func TestRedirectHandler(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/redirect-url/12345", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.RedirectHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	expected := "Merchant Transaction ID: 12345"
	assert.Equal(t, expected, rr.Body.String())
}

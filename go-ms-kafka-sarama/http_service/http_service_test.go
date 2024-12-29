package http_service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sing3demons/logger-kp/logger"
)

func TestExecuteRequestSuccess(t *testing.T) {
	// Create a test server with a successful response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "success"})
	}))
	defer ts.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	attr := RequestAttributes{
		Method: GET,
		URL:    ts.URL,
	}

	req, err := http.NewRequest(string(attr.Method), attr.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	response := executeRequest(client, req, attr)
	if response.Status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, response.Status)
	}
	if response.Err != nil {
		t.Errorf("Expected no error, got %v", response.Err)
	}
	if response.Body.(map[string]interface{})["message"] != "success" {
		t.Errorf("Expected body message 'success', got %v", response.Body)
	}
}

func TestExecuteRequestError(t *testing.T) {
	// Create a test server with an error response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	attr := RequestAttributes{
		Method: GET,
		URL:    ts.URL,
	}

	req, err := http.NewRequest(string(attr.Method), attr.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	response := executeRequest(client, req, attr)
	if response.Status != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, response.Status)
	}
	if response.Err != nil {
		t.Errorf("Expected no error, got %v", response.Err)
	}
}

func TestExecuteRequestTimeout(t *testing.T) {
	// Create a test server that delays the response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := &http.Client{Timeout: 1 * time.Second}
	attr := RequestAttributes{
		Method: GET,
		URL:    ts.URL,
	}

	req, err := http.NewRequest(string(attr.Method), attr.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	response := executeRequest(client, req, attr)
	if response.Status != 500 {
		t.Errorf("Expected status 500, got %d", response.Status)
	}
	if response.Err == nil {
		t.Errorf("Expected timeout error, got nil")
	}
}

func TestExecuteRequestWithBody(t *testing.T) {
	// Create a test server that echoes the request body
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write(bodyBytes)
	}))
	defer ts.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	attr := RequestAttributes{
		Method: POST,
		URL:    ts.URL,
		Body:   TMap{"key": "value"},
	}

	bodyBytes, _ := json.Marshal(attr.Body)
	req, err := http.NewRequest(string(attr.Method), attr.URL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	response := executeRequest(client, req, attr)
	if response.Status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, response.Status)
	}
	if response.Err != nil {
		t.Errorf("Expected no error, got %v", response.Err)
	}
	if response.Body.(map[string]interface{})["key"] != "value" {
		t.Errorf("Expected body key 'value', got %v", response.Body)
	}
}
func TestCreateRequest(t *testing.T) {
	detailLog := logger.NewDetailLog("", "", "")
	summaryLog := logger.NewSummaryLog("", "", "")

	mockURL := "http://example.com"

	tests := []struct {
		name       string
		attr       RequestAttributes
		wantErr    bool
		wantMethod string
		wantURL    string
		wantBody   string
	}{
		{
			name: "GET request without body",
			attr: RequestAttributes{
				Method: GET,
				URL:    mockURL,
			},
			wantErr:    false,
			wantMethod: "GET",
			wantURL:    mockURL,
			wantBody:   "",
		},
		{
			name: "POST request with body",
			attr: RequestAttributes{
				Method: POST,
				URL:    mockURL,
				Body:   TMap{"key": "value"},
			},
			wantErr:    false,
			wantMethod: "POST",
			wantURL:    mockURL,
			wantBody:   `{"key":"value"}`,
		},
		{
			name: "Request with headers",
			attr: RequestAttributes{
				Method: GET,
				URL:    mockURL,
				Headers: TMap{
					"Content-Type": "application/json",
				},
			},
			wantErr:    false,
			wantMethod: "GET",
			wantURL:    mockURL,
			wantBody:   "",
		},
		{
			name: "Request with query parameters",
			attr: RequestAttributes{
				Method: GET,
				URL:    mockURL,
				Query: TMap{
					"param1": "value1",
					"param2": "value2",
				},
			},
			wantErr:    false,
			wantMethod: "GET",
			wantURL:    "http://example.com?param1=value1&param2=value2",
			wantBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req, err := createRequest(ctx, tt.attr, detailLog, summaryLog)
			if (err != nil) != tt.wantErr {
				t.Errorf("createRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if req.Method != tt.wantMethod {
				t.Errorf("Expected method %s, got %s", tt.wantMethod, req.Method)
			}
			if req.URL.String() != tt.wantURL {
				t.Errorf("Expected URL %s, got %s", tt.wantURL, req.URL.String())
			}
			if tt.wantBody != "" {
				bodyBytes, _ := io.ReadAll(req.Body)
				if string(bodyBytes) != tt.wantBody {
					t.Errorf("Expected body %s, got %s", tt.wantBody, string(bodyBytes))
				}
			}
		})
	}
}
func TestExecuteRequest(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		attr           RequestAttributes
		expectedStatus int
		expectedBody   map[string]interface{}
		expectError    bool
	}{
		{
			name: "Successful GET request",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"message": "success"})
			},
			attr: RequestAttributes{
				Method: GET,
				URL:    "",
			},
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]interface{}{"message": "success"},
			expectError:    false,
		},
		{
			name: "Internal Server Error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			attr: RequestAttributes{
				Method: GET,
				URL:    "",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
			expectError:    false,
		},
		{
			name: "Timeout Error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(2 * time.Second)
				w.WriteHeader(http.StatusOK)
			},
			attr: RequestAttributes{
				Method: GET,
				URL:    "",
			},
			expectedStatus: 500,
			expectedBody:   nil,
			expectError:    true,
		},
		{
			name: "POST request with body",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				bodyBytes, _ := io.ReadAll(r.Body)
				w.WriteHeader(http.StatusOK)
				w.Write(bodyBytes)
			},
			attr: RequestAttributes{
				Method: POST,
				URL:    "",
				Body:   TMap{"key": "value"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]interface{}{"key": "value"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer ts.Close()

			client := &http.Client{Timeout: 1 * time.Second}
			tt.attr.URL = ts.URL

			var body io.Reader
			if len(tt.attr.Body) > 0 {
				bodyBytes, _ := json.Marshal(tt.attr.Body)
				body = bytes.NewBuffer(bodyBytes)
			}

			req, err := http.NewRequest(string(tt.attr.Method), tt.attr.URL, body)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			response := executeRequest(client, req, tt.attr)
			if response.Status != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, response.Status)
			}
			if (response.Err != nil) != tt.expectError {
				t.Errorf("Expected error %v, got %v", tt.expectError, response.Err)
			}
			if tt.expectedBody != nil && !equalMaps(response.Body.(map[string]interface{}), tt.expectedBody) {
				t.Errorf("Expected body %v, got %v", tt.expectedBody, response.Body)
			}
		})
	}
}

func equalResponses(a, b interface{}) bool {
	switch a := a.(type) {
	case map[string]interface{}:
		b, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		return equalMaps(a, b)
	case []interface{}:
		b, ok := b.([]interface{})
		if !ok {
			return false
		}
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if !equalResponses(a[i], b[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
func equalMaps(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

package pdfclient_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	pdfclient "github.com/mhpenta/pypdftotext-client"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		options     []pdfclient.ClientOption
		wantErr     bool
		wantBaseURL string
	}{
		{
			name:        "valid URL",
			baseURL:     "http://localhost:8000",
			options:     nil,
			wantErr:     false,
			wantBaseURL: "http://localhost:8000",
		},
		{
			name:        "missing scheme",
			baseURL:     "localhost:8000",
			options:     nil,
			wantErr:     false,
			wantBaseURL: "http://localhost:8000",
		},
		{
			name:        "with options",
			baseURL:     "http://localhost:8000",
			options:     []pdfclient.ClientOption{pdfclient.WithDebug(true), pdfclient.WithTimeout(5 * time.Second)},
			wantErr:     false,
			wantBaseURL: "http://localhost:8000",
		},
		{
			name:    "invalid URL",
			baseURL: ":%invalid:",
			options: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := pdfclient.NewClient(tt.baseURL, tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && client.BaseURL != tt.wantBaseURL {
				t.Errorf("NewClient() baseURL = %v, want %v", client.BaseURL, tt.wantBaseURL)
			}
		})
	}
}

func TestHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected request to '/health', got: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","version":"1.0.0"}`)) 
	}))
	defer server.Close()

	client, err := pdfclient.NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	health, err := client.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	if health.Status != "ok" {
		t.Errorf("HealthCheck() status = %v, want %v", health.Status, "ok")
	}

	if health.Version != "1.0.0" {
		t.Errorf("HealthCheck() version = %v, want %v", health.Version, "1.0.0")
	}
}

func TestExtractTextMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/extract" {
			t.Errorf("Expected request to '/extract', got: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Check for multipart form data
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("Failed to parse multipart form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Check if file was provided
		if _, _, err := r.FormFile("file"); err != nil {
			t.Errorf("No file provided: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"pages": [{"page": 1, "text": "Sample extracted text from PDF."}],
			"page_count": 1,
			"file_name": "test.pdf",
			"file_size": 1024
		}`))
	}))
	defer server.Close()

	client, err := pdfclient.NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create a temporary PDF file for testing
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.pdf")

	if err := os.WriteFile(tempFile, []byte("fake PDF content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test ExtractTextFromFile
	result, err := client.ExtractTextFromFile(context.Background(), tempFile)
	if err != nil {
		t.Fatalf("ExtractTextFromFile() error = %v", err)
	}

	if result.GetFullText() != "Sample extracted text from PDF." {
		t.Errorf("ExtractTextFromFile() text = %v, want %v", result.GetFullText(), "Sample extracted text from PDF.")
	}

	if result.PageCount != 1 {
		t.Errorf("ExtractTextFromFile() pageCount = %v, want %v", result.PageCount, 1)
	}

	if len(result.Pages) != 1 {
		t.Errorf("ExtractTextFromFile() pages length = %v, want %v", len(result.Pages), 1)
	}

	// Test ExtractTextFromBytes
	bytes := []byte("fake PDF content")
	result, err = client.ExtractTextFromBytes(context.Background(), bytes, "test.pdf")
	if err != nil {
		t.Fatalf("ExtractTextFromBytes() error = %v", err)
	}

	if result.GetFullText() != "Sample extracted text from PDF." {
		t.Errorf("ExtractTextFromBytes() text = %v, want %v", result.GetFullText(), "Sample extracted text from PDF.")
	}
}

func TestExtractTextFromGCS(t *testing.T) {
	outputLocation := "gs://test-bucket/output/test.txt"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/extract-from-gcs" {
			t.Errorf("Expected request to '/extract-from-gcs', got: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Check Content-Type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got: %s", r.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{
			"pages": [
				{"page": 1, "text": "Sample extracted text from GCS PDF page 1."},
				{"page": 2, "text": "Sample extracted text from GCS PDF page 2."}
			],
			"page_count": 2,
			"file_name": "test.pdf",
			"file_size": 2048,
			"method": "pdfplumber",
			"output_location": "%s"
		}`, outputLocation)))
	}))
	defer server.Close()

	client, err := pdfclient.NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test with output location
	request := pdfclient.GCSExtractionRequest{
		InputGCSURL:  "gs://test-bucket/input/test.pdf",
		OutputGCSURL: &outputLocation,
		Method:       "auto",
	}

	result, err := client.ExtractTextFromGCS(context.Background(), request)
	if err != nil {
		t.Fatalf("ExtractTextFromGCS() error = %v", err)
	}

	expectedText := "Sample extracted text from GCS PDF page 1.\n\nSample extracted text from GCS PDF page 2."
	if result.GetFullText() != expectedText {
		t.Errorf("ExtractTextFromGCS() text = %v, want %v", result.GetFullText(), expectedText)
	}

	if result.PageCount != 2 {
		t.Errorf("ExtractTextFromGCS() pageCount = %v, want %v", result.PageCount, 2)
	}

	if len(result.Pages) != 2 {
		t.Errorf("ExtractTextFromGCS() pages length = %v, want %v", len(result.Pages), 2)
	}

	if result.Method != "pdfplumber" {
		t.Errorf("ExtractTextFromGCS() method = %v, want %v", result.Method, "pdfplumber")
	}

	if result.OutputLocation == nil || *result.OutputLocation != outputLocation {
		t.Errorf("ExtractTextFromGCS() outputLocation = %v, want %v", result.OutputLocation, outputLocation)
	}
}

func TestExtractTextFromGCS_NoOutputLocation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"text": "Sample text",
			"page_count": 1,
			"file_name": "test.pdf",
			"file_size": 1024,
			"method": "pypdf2"
		}`))
	}))
	defer server.Close()

	client, err := pdfclient.NewClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test without output location
	request := pdfclient.GCSExtractionRequest{
		InputGCSURL: "gs://test-bucket/input/test.pdf",
		Method:      "pypdf2",
	}

	result, err := client.ExtractTextFromGCS(context.Background(), request)
	if err != nil {
		t.Fatalf("ExtractTextFromGCS() error = %v", err)
	}

	if result.OutputLocation != nil {
		t.Errorf("ExtractTextFromGCS() outputLocation = %v, want nil", result.OutputLocation)
	}
}

func TestClientError_GCSErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        pdfclient.ClientError
		wantPerm   bool
		wantNotFound bool
	}{
		{
			name: "GCS Permission Error",
			err: pdfclient.ClientError{
				StatusCode: http.StatusForbidden,
				Detail:     "Permission denied accessing gs://bucket/file.pdf",
			},
			wantPerm:     true,
			wantNotFound: false,
		},
		{
			name: "GCS Not Found Error",
			err: pdfclient.ClientError{
				StatusCode: http.StatusNotFound,
				Detail:     "Blob does not exist: gs://bucket/file.pdf",
			},
			wantPerm:     false,
			wantNotFound: true,
		},
		{
			name: "Other Error",
			err: pdfclient.ClientError{
				StatusCode: http.StatusInternalServerError,
				Detail:     "Internal server error",
			},
			wantPerm:     false,
			wantNotFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.IsGCSPermissionError(); got != tt.wantPerm {
				t.Errorf("IsGCSPermissionError() = %v, want %v", got, tt.wantPerm)
			}
			if got := tt.err.IsGCSNotFoundError(); got != tt.wantNotFound {
				t.Errorf("IsGCSNotFoundError() = %v, want %v", got, tt.wantNotFound)
			}
		})
	}
}

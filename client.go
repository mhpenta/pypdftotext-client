package pdfclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
	Debug      bool
	Timeout    time.Duration
	APIKey     string
}

type ClientOption func(*Client)

func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.HTTPClient = client
	}
}

func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) {
		c.UserAgent = userAgent
	}
}

func WithDebug(debug bool) ClientOption {
	return func(c *Client) {
		c.Debug = debug
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.Timeout = timeout
	}
}

func WithAPIKey(apiKey string) ClientOption {
	return func(c *Client) {
		c.APIKey = apiKey
	}
}

func NewClient(baseURL string, options ...ClientOption) (*Client, error) {
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")

	client := &Client{
		BaseURL: parsedURL.String(),
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		UserAgent: "Go-PyPDFToTextClient/1.0",
		Debug:     false,
		Timeout:   120 * time.Second,
	}

	for _, option := range options {
		option(client)
	}

	return client, nil
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type TextExtractionResponse struct {
	Text      string `json:"text"`
	PageCount int    `json:"page_count"`
	FileName  string `json:"file_name"`
	FileSize  int    `json:"file_size"`
}

type GCSExtractionRequest struct {
	InputGCSURL  string  `json:"input_gcs_url"`
	OutputGCSURL *string `json:"output_gcs_url,omitempty"`
	Method       string  `json:"method,omitempty"`
	ProjectID    *string `json:"project_id,omitempty"`
}

type GCSExtractionResponse struct {
	Text           string  `json:"text"`
	PageCount      int     `json:"page_count"`
	FileName       string  `json:"file_name"`
	FileSize       int     `json:"file_size"`
	Method         string  `json:"method"`
	OutputLocation *string `json:"output_location,omitempty"`
}

type ClientError struct {
	StatusCode int
	Message    string
	Detail     string
}

func (e ClientError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("API error (HTTP %d): %s - %s", e.StatusCode, e.Message, e.Detail)
	}
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Message)
}

// IsInvalidPDFError returns true if the error is related to an invalid PDF
func (e ClientError) IsInvalidPDFError() bool {
	return e.StatusCode == http.StatusBadRequest && 
		(strings.Contains(e.Detail, "PDF file has syntax errors") || 
		 strings.Contains(e.Detail, "PDF file appears to be corrupted") || 
		 strings.Contains(e.Detail, "PDF file is incomplete or truncated") ||
		 strings.Contains(e.Detail, "Invalid PDF format") ||
		 strings.Contains(e.Detail, "file does not appear to be a valid PDF"))
}

// IsTimeoutError returns true if the error is a timeout error
func (e ClientError) IsTimeoutError() bool {
	return e.StatusCode == http.StatusRequestTimeout || 
		strings.Contains(e.Detail, "timed out")
}

// IsFileSizeError returns true if the error is related to a file size limit
func (e ClientError) IsFileSizeError() bool {
	return e.StatusCode == http.StatusRequestEntityTooLarge ||
		strings.Contains(e.Detail, "File too large")
}

// IsGCSPermissionError returns true if the error is related to GCS permissions
func (e ClientError) IsGCSPermissionError() bool {
	return e.StatusCode == http.StatusForbidden &&
		strings.Contains(e.Detail, "Permission denied")
}

// IsGCSNotFoundError returns true if the error is related to GCS resource not found
func (e ClientError) IsGCSNotFoundError() bool {
	return e.StatusCode == http.StatusNotFound &&
		(strings.Contains(e.Detail, "not found") || strings.Contains(e.Detail, "does not exist"))
}

func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	reqURL := fmt.Sprintf("%s/health", c.BaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	if c.Debug {
		fmt.Printf("DEBUG: Making request to %s\n", reqURL)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			slog.Error("Failed to close response body in remote PyPDFToText health check", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		
		// Try to parse the error response as JSON
		var apiError struct {
			Detail string `json:"detail"`
		}
		
		detail := ""
		if err := json.Unmarshal(body, &apiError); err == nil && apiError.Detail != "" {
			detail = apiError.Detail
		}
		
		return nil, ClientError{
			StatusCode: resp.StatusCode,
			Message:    bodyStr,
			Detail:     detail,
		}
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &health, nil
}

func (c *Client) ExtractTextFromFile(ctx context.Context, filePath string) (*TextExtractionResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			slog.Error("Failed to close file", "error", err)
		}
	}(file)

	return c.ExtractTextFromReader(ctx, file, filepath.Base(filePath))
}

func (c *Client) ExtractTextFromBytes(ctx context.Context, fileContent []byte, fileName string) (*TextExtractionResponse, error) {
	reader := bytes.NewReader(fileContent)
	return c.ExtractTextFromReader(ctx, reader, fileName)
}

func (c *Client) ExtractTextFromReader(ctx context.Context, reader io.Reader, fileName string) (*TextExtractionResponse, error) {
	reqURL := fmt.Sprintf("%s/extract", c.BaseURL)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("error creating form file: %w", err)
	}

	if _, err := io.Copy(part, reader); err != nil {
		return nil, fmt.Errorf("error copying file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("error closing multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	if c.Debug {
		fmt.Printf("DEBUG: Making request to %s with file %s\n", reqURL, fileName)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		innerErr := Body.Close()
		if innerErr != nil {
			slog.Error("Failed to close response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		
		// Try to parse the error response as JSON
		var apiError struct {
			Detail string `json:"detail"`
		}
		
		detail := ""
		if err := json.Unmarshal(body, &apiError); err == nil && apiError.Detail != "" {
			detail = apiError.Detail
		}
		
		return nil, ClientError{
			StatusCode: resp.StatusCode,
			Message:    bodyStr,
			Detail:     detail,
		}
	}

	var result TextExtractionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &result, nil
}

func (c *Client) ExtractTextFromGCS(ctx context.Context, request GCSExtractionRequest) (*GCSExtractionResponse, error) {
	reqURL := fmt.Sprintf("%s/extract-from-gcs", c.BaseURL)

	// Set default method if not provided
	if request.Method == "" {
		request.Method = "auto"
	}

	// Marshal request body
	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	if c.Debug {
		fmt.Printf("DEBUG: Making request to %s with GCS URL %s\n", reqURL, request.InputGCSURL)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		innerErr := Body.Close()
		if innerErr != nil {
			slog.Error("Failed to close response body", "error", innerErr)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Try to parse the error response as JSON
		var apiError struct {
			Detail string `json:"detail"`
		}

		detail := ""
		if err := json.Unmarshal(body, &apiError); err == nil && apiError.Detail != "" {
			detail = apiError.Detail
		}

		return nil, ClientError{
			StatusCode: resp.StatusCode,
			Message:    bodyStr,
			Detail:     detail,
		}
	}

	var result GCSExtractionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &result, nil
}

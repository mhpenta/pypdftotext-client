package pdfclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
			Timeout: 30 * time.Second,
		},
		UserAgent: "Go-PDFClient/1.0",
		Debug:     false,
		Timeout:   30 * time.Second,
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

type ClientError struct {
	StatusCode int
	Message    string
}

func (e ClientError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Message)
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, ClientError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
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
	defer file.Close()

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, ClientError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	var result TextExtractionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &result, nil
}
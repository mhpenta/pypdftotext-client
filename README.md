# PDF Text Extraction Client for Go

A Go client library for interacting with our PDF Text Extraction API service. This client provides a simple interface for extracting text from PDF files via the API. This service and client exist because of limitations within the go ecosystem around pdf to text extraction.

## Features

- Simple client initialization with configurable options
- Health check endpoint support
- Multiple methods for extracting text from PDFs (file path, bytes, io.Reader)
- API key authentication support
- Proper error handling and context support
- Configurable timeouts and HTTP client settings
- Comprehensive test coverage

## Installation

```bash
go get github.com/mhpenta/pypdftotext-client
```

## Usage

### Basic Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	pdfclient "github.com/mhpenta/pypdftotext-client"
)

func main() {
	// Initialize the client
	client, err := pdfclient.NewClient("http://localhost:8000")
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	// Extract text from a PDF file
	result, err := client.ExtractTextFromFile(context.Background(), "/path/to/document.pdf")
	if err != nil {
		log.Fatalf("Text extraction failed: %v", err)
	}

	// Use the extracted text
	fmt.Printf("Extracted %d pages of text from %s (%d bytes)\n", 
		result.PageCount, result.FileName, result.FileSize)
	fmt.Println(result.Text)
}
```

### Advanced Configuration

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	pdfclient "github.com/mhpenta/pypdftotext-client"
)

func main() {
	// Create a custom HTTP client
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
		// Add other HTTP client configurations as needed
	}

	// Initialize the client with options
	client, err := pdfclient.NewClient(
		"http://localhost:8000",
		pdfclient.WithHTTPClient(httpClient),
		pdfclient.WithDebug(true),
		pdfclient.WithUserAgent("MyApp/1.0"),
		pdfclient.WithTimeout(30*time.Second),
		pdfclient.WithAPIKey("your-secret-api-key"),  // Add API key for authentication
	)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	// Check API health
	health, err := client.HealthCheck(context.Background())
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}

	fmt.Printf("API Status: %s, Version: %s\n", health.Status, health.Version)

	// Extract text from PDF bytes
	pdfBytes := []byte{/* PDF content */}
	result, err := client.ExtractTextFromBytes(context.Background(), pdfBytes, "document.pdf")
	if err != nil {
		log.Fatalf("Text extraction failed: %v", err)
	}

	fmt.Println(result.Text)
}
```

## API Reference

### Client Initialization

```go
func NewClient(baseURL string, options ...ClientOption) (*Client, error)
```

Create a new client with the provided base URL and optional configuration options.

### Client Options

- `WithHTTPClient(client *http.Client)`: Use a custom HTTP client
- `WithUserAgent(userAgent string)`: Set a custom User-Agent header
- `WithDebug(debug bool)`: Enable/disable debug logging
- `WithTimeout(timeout time.Duration)`: Set a request timeout
- `WithAPIKey(apiKey string)`: Set API key for authentication

### Methods

#### Health Check

```go
func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error)
```

Check the health of the API server.

#### Extract Text

```go
func (c *Client) ExtractTextFromFile(ctx context.Context, filePath string) (*TextExtractionResponse, error)
```

Extract text from a PDF file at the given file path.

```go
func (c *Client) ExtractTextFromBytes(ctx context.Context, fileContent []byte, fileName string) (*TextExtractionResponse, error)
```

Extract text from PDF content provided as a byte slice.

```go
func (c *Client) ExtractTextFromReader(ctx context.Context, reader io.Reader, fileName string) (*TextExtractionResponse, error)
```

Extract text from PDF content provided as an io.Reader.

### Response Types

#### HealthResponse

```go
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}
```

#### TextExtractionResponse

```go
type TextExtractionResponse struct {
	Text      string `json:"text"`
	PageCount int    `json:"page_count"`
	FileName  string `json:"file_name"`
	FileSize  int    `json:"file_size"`
}
```

## Error Handling

The client provides detailed error information when API calls fail. The `ClientError` type includes the HTTP status code and error message from the API.

```go
result, err := client.ExtractTextFromFile(context.Background(), "/path/to/document.pdf")
if err != nil {
	if clientErr, ok := err.(pdfclient.ClientError); ok {
		fmt.Printf("API error (HTTP %d): %s\n", clientErr.StatusCode, clientErr.Message)
	} else {
		fmt.Printf("Error: %v\n", err)
	}
	return
}
```

## Example

See the `example` directory for a complete example of using the client.

## License

MIT

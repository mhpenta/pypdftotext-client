# PDF Text Extraction Client for Go

Go client for the PDF Text Extraction API. Supports local files and Google Cloud Storage URLs.

## Features

- Local file extraction (file path, bytes, io.Reader)
- GCS URL extraction with optional output to GCS
- API key authentication
- Multiple extraction methods (PyPDF2, pdfplumber, auto)
- Context support and configurable timeouts
- Typed error handling

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

### With Options

```go
client, err := pdfclient.NewClient(
	"http://localhost:8000",
	pdfclient.WithAPIKey("your-api-key"),
	pdfclient.WithTimeout(30*time.Second),
	pdfclient.WithDebug(true),
)
```

### GCS Extraction

```go
request := pdfclient.GCSExtractionRequest{
	InputGCSURL:  "gs://bucket/file.pdf",
	OutputGCSURL: &outputURL,  // Optional
	Method:       "auto",       // "auto", "pypdf2", or "pdfplumber"
}

result, err := client.ExtractTextFromGCS(ctx, request)
```

## Client Options

- `WithAPIKey(string)` - API key authentication
- `WithTimeout(time.Duration)` - Request timeout
- `WithDebug(bool)` - Debug logging
- `WithHTTPClient(*http.Client)` - Custom HTTP client
- `WithUserAgent(string)` - Custom User-Agent

## Methods

- `HealthCheck(ctx)` - Check API health
- `ExtractTextFromFile(ctx, filePath)` - Extract from local file
- `ExtractTextFromBytes(ctx, data, fileName)` - Extract from bytes
- `ExtractTextFromReader(ctx, reader, fileName)` - Extract from io.Reader
- `ExtractTextFromGCS(ctx, request)` - Extract from GCS URL

## Error Handling

```go
if err != nil {
	if clientErr, ok := err.(pdfclient.ClientError); ok {
		if clientErr.IsInvalidPDFError() { /* corrupted PDF */ }
		if clientErr.IsTimeoutError() { /* timeout */ }
		if clientErr.IsFileSizeError() { /* too large */ }
		if clientErr.IsGCSPermissionError() { /* GCS permission denied */ }
		if clientErr.IsGCSNotFoundError() { /* GCS not found */ }
	}
}
```

## Examples

See the `example/` directory for complete usage examples.

## License

MIT

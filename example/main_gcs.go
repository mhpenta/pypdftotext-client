package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	pdfclient "github.com/mhpenta/pypdftotext-client"
)

func main() {
	s := time.Now()

	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main_gcs.go <server-url> <input-gcs-url> [output-gcs-url] [api-key]")
		fmt.Println("Example: go run main_gcs.go http://localhost:8000 gs://bucket/input/file.pdf gs://bucket/output/file.txt my-secret-key")
		fmt.Println("Example (no output): go run main_gcs.go http://localhost:8000 gs://bucket/input/file.pdf")
		os.Exit(1)
	}

	serverURL := os.Args[1]
	inputGCSURL := os.Args[2]

	// Optional output GCS URL
	var outputGCSURL *string
	if len(os.Args) >= 4 && os.Args[3] != "" && os.Args[3] != "-" {
		outputGCSURL = &os.Args[3]
	}

	// Optional API key from command line
	apiKey := ""
	if len(os.Args) >= 5 {
		apiKey = os.Args[4]
	}

	// Create a client with custom options
	clientOptions := []pdfclient.ClientOption{
		pdfclient.WithDebug(true),
		pdfclient.WithTimeout(120 * time.Second), // GCS operations may take longer
		pdfclient.WithUserAgent("PDFClient-GCS-Example/1.0"),
	}

	// Add API key if provided
	if apiKey != "" {
		clientOptions = append(clientOptions, pdfclient.WithAPIKey(apiKey))
		fmt.Println("Using API key authentication")
	}

	client, err := pdfclient.NewClient(serverURL, clientOptions...)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	// Check API health
	fmt.Println("Checking API health...")
	health, err := client.HealthCheck(context.Background())
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}

	fmt.Printf("API Status: %s, Version: %s\n\n", health.Status, health.Version)

	// Extract text from GCS
	fmt.Printf("Extracting text from GCS: %s...\n", inputGCSURL)

	request := pdfclient.GCSExtractionRequest{
		InputGCSURL:  inputGCSURL,
		OutputGCSURL: outputGCSURL,
		Method:       "auto", // or "pypdf2" or "pdfplumber"
	}

	result, err := client.ExtractTextFromGCS(context.Background(), request)
	if err != nil {
		// Check for specific error types
		if clientErr, ok := err.(pdfclient.ClientError); ok {
			if clientErr.IsGCSPermissionError() {
				log.Fatalf("GCS Permission Error: %v\nMake sure the service account has proper GCS permissions.", err)
			} else if clientErr.IsGCSNotFoundError() {
				log.Fatalf("GCS Not Found Error: %v\nCheck that the GCS URL is correct and the file exists.", err)
			}
		}
		log.Fatalf("Text extraction failed: %v", err)
	}

	fmt.Printf("Extraction successful!\n")
	fmt.Printf("File: %s\n", result.FileName)
	fmt.Printf("Size: %d bytes\n", result.FileSize)
	fmt.Printf("Pages: %d\n", result.PageCount)
	fmt.Printf("Method: %s\n", result.Method)

	if result.OutputLocation != nil {
		fmt.Printf("Saved to: %s\n", *result.OutputLocation)
	}

	fmt.Printf("\nExtracted Text Preview (first 500 chars):\n")

	fullText := result.GetFullText()
	previewText := fullText
	if len(fullText) > 500 {
		previewText = fullText[:500] + "..."
	}

	fmt.Println(previewText)
	fmt.Printf("\nTotal time: %v\n", time.Since(s))
}

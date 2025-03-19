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
		fmt.Println("Usage: go run main.go <server-url> <pdf-file-path> [api-key]")
		fmt.Println("Example: go run main.go http://localhost:8000 /path/to/document.pdf my-secret-key")
		os.Exit(1)
	}

	serverURL := os.Args[1]
	pdfPath := os.Args[2]

	// Optional API key from command line
	apiKey := ""
	if len(os.Args) >= 4 {
		apiKey = os.Args[3]
	}

	// Create a client with custom options
	clientOptions := []pdfclient.ClientOption{
		pdfclient.WithDebug(true),
		pdfclient.WithTimeout(30 * time.Second),
		pdfclient.WithUserAgent("PDFClient-Example/1.0"),
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

	// Extract text from the provided PDF file
	fmt.Printf("Extracting text from %s...\n", pdfPath)
	result, err := client.ExtractTextFromFile(context.Background(), pdfPath)
	if err != nil {
		log.Fatalf("Text extraction failed: %v", err)
	}

	fmt.Printf("Extraction successful!\n")
	fmt.Printf("File: %s\n", result.FileName)
	fmt.Printf("Size: %d bytes\n", result.FileSize)
	fmt.Printf("Pages: %d\n", result.PageCount)
	fmt.Printf("\nExtracted Text Preview (first 500 chars):\n")

	previewText := result.Text
	if len(result.Text) > 5000 {
		previewText = result.Text[:5000] + "..."
	}

	fmt.Println(previewText)
	fmt.Printf("\nTotal time: %v\n", time.Since(s))
}

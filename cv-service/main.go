package main

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"pizza-nz/cv-service/tracing"

	"github.com/gin-gonic/gin"
	"github.com/h2non/filetype"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
)

type Score struct {
	Scenario string `json:"scenario"`
	Score    int    `json:"score"`
}

var (
	allowedTypes map[string]bool = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
	}
)

func main() {
	tp, err := tracing.InitTracerProvider("cv-service")
	if err != nil {
		fmt.Println("failed to initialize tracer provider:", err)
		os.Exit(1)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			fmt.Printf("Error shutting down tracer provider: %v\n", err)
		}
	}()

	router := gin.Default()
	router.Use(otelgin.Middleware("cv-service"))

	router.POST("/process", handleProcessImage)

	fmt.Println("CV service running on :8081")
	router.Run(":8081")
}

func handleProcessImage(c *gin.Context) {
	tracer := otel.Tracer("cv-handler")
	_, span := tracer.Start(c.Request.Context(), "handleProcessImage")
	defer span.End()

	img, head, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image file not found"})
		return
	}

	mockScores, err := processImage(img, head)
	if err != nil {
		c.JSON(http.StatusBadRequest, "failed to process image")
	}

	c.JSON(http.StatusOK, gin.H{"scores": mockScores})
}

func processImage(file multipart.File, handler *multipart.FileHeader) ([]Score, error) {
	defer file.Close()

	// Copied from pizza-nz/file-uploader
	// Read the first 261 bytes to determine the file type
	head := make([]byte, 261)
	if _, err := file.Read(head); err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file header: %w", err)
	}

	// Reset the file reader so the full file can be read again later
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to reset file reader: %w", err)
	}

	// Use filetype.Match to determine the file type based on magic numbers
	kind, err := filetype.Match(head)
	if err != nil {
		return nil, fmt.Errorf("failed to match file type: %w", err)
	}

	// Check if the detected file type is allowed
	if kind == filetype.Unknown || !allowedTypes[kind.MIME.Value] {
		return nil, fmt.Errorf("file type %s is not allowed", kind.MIME.Value)
	}

	// TODO: LOGIC HERE
	fmt.Println("Received image for processing. Returning mock data.")
	mockScores := []Score{
		{Scenario: "VT Adjustshot VALORANT", Score: 805},
		{Scenario: "VT Flickspeed VALORANT", Score: 825},
		{Scenario: "VT Angleshot VALORANT", Score: 677},
	}

	return mockScores, nil
}

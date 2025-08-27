package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"pizza-nz/gateway/internal/database"
	"pizza-nz/gateway/internal/tracing"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
)

type CVServiceResponse struct {
	Scores []database.Score `json:"scores"`
}

type UpdateScorePayload struct {
	Scenario string `json:"scenario"`
	Score    int    `json:"score"`
}

func main() {
	tp, err := tracing.InitTracerProvider("gateway-service")
	if err != nil {
		fmt.Printf("failed to initialise tracer provider: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			fmt.Printf("Error shutting down tracer provider: %v\n", err)
		}
	}()

	db, err := database.InitDB("/data/scores.db")
	if err != nil {
		fmt.Printf("failed to initialise db: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	router := gin.Default()

	router.Use(otelgin.Middleware("gateway-service"))

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.POST("/upload", handleUpload(db))
	router.GET("/scores", handleGetScores(db))
	router.PUT("/scores/:id", handleUpdateScore(db))

	router.Run(":8080")
}

func handleUpload(db *database.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tracer := otel.Tracer("gateway-handler")
		ctx, span := tracer.Start(c.Request.Context(), "handleUpload")
		defer span.End()

		file, header, err := c.Request.FormFile("screenshot")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Could not get file from form"})
			return
		}
		defer file.Close()

		cvServiceURL := os.Getenv("CV_SERVICE_URL") + "/process"
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", header.Filename)
		io.Copy(part, file)
		writer.Close()

		req, _ := http.NewRequestWithContext(ctx, "POST", cvServiceURL, body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process image with CV service"})
			return
		}
		defer resp.Body.Close()

		var cvResponse CVServiceResponse
		if err := json.NewDecoder(resp.Body).Decode(&cvResponse); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode CV service response"})
			return
		}

		for _, score := range cvResponse.Scores {
			err := db.CreateScore(ctx, score.Scenario, score.Score)
			if err != nil {
				fmt.Printf("Error saving score for %s: %v\n", score.Scenario, err)
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "File processed successfully", "scores_found": len(cvResponse.Scores)})
	}
}

func handleGetScores(db *database.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tracer := otel.Tracer("gateway-handler")
		ctx, span := tracer.Start(c.Request.Context(), "handleGetScores")
		defer span.End()

		scores, err := db.GetAllScores(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve scores"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"scores": scores})
	}
}

func handleUpdateScore(db *database.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tracer := otel.Tracer("gateway-handler")
		ctx, span := tracer.Start(c.Request.Context(), "handleUpdateScore")
		defer span.End()

		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid score ID"})
			return
		}

		var payload UpdateScorePayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		err = db.UpdateScore(ctx, id, payload.Scenario, payload.Score)
		if err != nil {
			fmt.Printf("Error updating score ID %d: %v\n", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update score"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Score updated successfully"})
	}
}

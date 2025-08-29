package ai

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	stripehandlers "pocketbase/internal/stripe"
)

// TextProcessingRequest represents a request for text-based AI processing
type TextProcessingRequest struct {
	SystemPrompt string                 `json:"system_prompt"`
	UserPrompt   string                 `json:"user_prompt"`
	Model        string                 `json:"model"`
	TaskType     string                 `json:"task_type"` // "suggest_highlights", "reorder", "improve_silences", "chat"
	Context      map[string]interface{} `json:"context,omitempty"`
}

// TextProcessingResult represents the result of text processing
type TextProcessingResult struct {
	Content    string      `json:"content"`
	TaskType   string      `json:"task_type"`
	Structured interface{} `json:"structured,omitempty"`
	TokensUsed int         `json:"tokens_used,omitempty"`
}

// OpenRouterRequest represents the request format for OpenRouter API
type OpenRouterRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterResponse represents the response from OpenRouter API
type OpenRouterResponse struct {
	Choices []Choice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Choice represents a response choice
type Choice struct {
	Message Message `json:"message"`
}

// AudioProcessingRequest is no longer used - audio streaming uses multipart form data directly

// AudioProcessingResult represents the result of audio processing
type AudioProcessingResult struct {
	Transcript string    `json:"transcript"`
	Duration   float64   `json:"duration,omitempty"`
	Language   string    `json:"language,omitempty"`
	Words      []Word    `json:"words,omitempty"`
	Segments   []Segment `json:"segments,omitempty"`
}

// Word represents a word with timestamps
type Word struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// Segment represents a segment with timestamps
type Segment struct {
	ID               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	Tokens           []int   `json:"tokens"`
	Temperature      float64 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
	Words            []Word  `json:"words"`
}

// OpenAITranscriptionResponse represents the response from OpenAI transcription API
type OpenAITranscriptionResponse struct {
	Task     string    `json:"task"`
	Language string    `json:"language"`
	Duration float64   `json:"duration"`
	Text     string    `json:"text"`
	Segments []Segment `json:"segments"`
	Words    []Word    `json:"words"`
}

// ProcessTextHandler handles text processing requests
func ProcessTextHandler(e *core.RequestEvent, app core.App) error {
	startTime := time.Now()
	clientIP := getClientIP(e)
	userAgent := e.Request.Header.Get("User-Agent")
	
	log.Printf("🤖 [AI TEXT REQUEST] IP: %s | User-Agent: %s | Method: %s", 
		clientIP, userAgent, e.Request.Method)

	// Validate API key
	apiKey := extractBearerToken(e.Request.Header.Get("Authorization"))
	if apiKey == "" {
		log.Printf("❌ [AI TEXT REQUEST] FAILED: Missing API key | IP: %s", clientIP)
		return e.JSON(401, map[string]string{"error": "Missing or invalid API key"})
	}

	// Mask API key for logging (show first 8 chars)
	maskedKey := apiKey[:8] + "..."
	log.Printf("🔐 [AI TEXT REQUEST] API Key: %s | IP: %s", maskedKey, clientIP)

	// Check API key validity and get user
	user, err := validateAPIKey(app, apiKey)
	if err != nil {
		log.Printf("❌ [AI TEXT REQUEST] FAILED: Invalid API key %s | IP: %s | Error: %v", 
			maskedKey, clientIP, err)
		return e.JSON(401, map[string]string{"error": "Invalid API key"})
	}

	userEmail := user.GetString("email")
	userID := user.Id
	log.Printf("👤 [AI TEXT REQUEST] User: %s (%s) | API Key: %s | IP: %s", 
		userEmail, userID, maskedKey, clientIP)

	// Check user's subscription status
	if !isUserSubscribed(app, userID) {
		log.Printf("❌ [AI TEXT REQUEST] FAILED: No active subscription | User: %s | IP: %s", 
			userEmail, clientIP)
		return e.JSON(403, map[string]string{"error": "Active subscription required"})
	}

	// Parse request body
	var request TextProcessingRequest
	if err := e.BindBody(&request); err != nil {
		log.Printf("❌ [AI TEXT REQUEST] FAILED: Invalid request format | User: %s | IP: %s | Error: %v", 
			userEmail, clientIP, err)
		return e.JSON(400, map[string]string{"error": "Invalid request format"})
	}

	// Validate required fields
	if request.UserPrompt == "" {
		log.Printf("❌ [AI TEXT REQUEST] FAILED: Missing user_prompt | User: %s | IP: %s", 
			userEmail, clientIP)
		return e.JSON(400, map[string]string{"error": "user_prompt is required"})
	}

	// Set default model if not provided
	if request.Model == "" {
		request.Model = "anthropic/claude-3.5-sonnet"
	}

	// Log request details
	log.Printf("📝 [AI TEXT REQUEST] Processing | User: %s | Task: %s | Model: %s | Prompt Length: %d chars | System Prompt Length: %d chars | IP: %s", 
		userEmail, request.TaskType, request.Model, len(request.UserPrompt), len(request.SystemPrompt), clientIP)

	// Proxy request to OpenRouter
	result, err := proxyToOpenRouter(&request)
	if err != nil {
		elapsed := time.Since(startTime)
		log.Printf("❌ [AI TEXT REQUEST] FAILED: OpenRouter error | User: %s | Task: %s | Model: %s | Duration: %v | IP: %s | Error: %v", 
			userEmail, request.TaskType, request.Model, elapsed, clientIP, err)
		return e.JSON(500, map[string]string{"error": fmt.Sprintf("AI processing failed: %v", err)})
	}

	elapsed := time.Since(startTime)
	responseLength := len(result.Choices[0].Message.Content)
	
	// Log usage and success
	logAIUsage(app, userID, userEmail, request.TaskType, request.Model, 0, len(request.UserPrompt), responseLength, elapsed, clientIP)
	
	log.Printf("✅ [AI TEXT REQUEST] SUCCESS | User: %s | Task: %s | Model: %s | Response Length: %d chars | Duration: %v | IP: %s", 
		userEmail, request.TaskType, request.Model, responseLength, elapsed, clientIP)

	return e.JSON(200, result)
}

// GenerateAPIKeyHandler generates a new API key for authenticated users
func GenerateAPIKeyHandler(e *core.RequestEvent, app core.App) error {
	clientIP := getClientIP(e)
	userAgent := e.Request.Header.Get("User-Agent")
	
	log.Printf("🔑 [API KEY REQUEST] IP: %s | User-Agent: %s", clientIP, userAgent)

	// Get authenticated user
	user := e.Auth
	if user == nil {
		log.Printf("❌ [API KEY REQUEST] FAILED: No authentication | IP: %s", clientIP)
		return e.JSON(401, map[string]string{"error": "Authentication required"})
	}

	userEmail := user.GetString("email")
	userID := user.Id
	log.Printf("👤 [API KEY REQUEST] User: %s (%s) | IP: %s", userEmail, userID, clientIP)

	// Generate API key
	apiKey := generateAPIKey()
	keyHash := hashAPIKey(apiKey)

	// Create API key record
	apiKeyCollection, err := app.FindCollectionByNameOrId("api_keys")
	if err != nil {
		log.Printf("❌ [API KEY REQUEST] FAILED: Cannot find api_keys collection | User: %s | IP: %s | Error: %v", 
			userEmail, clientIP, err)
		return e.JSON(500, map[string]string{"error": "Failed to find API keys collection"})
	}

	record := core.NewRecord(apiKeyCollection)
	record.Set("key_hash", keyHash)
	record.Set("user_id", user.Id)
	record.Set("active", true)
	record.Set("name", fmt.Sprintf("API Key - %s", time.Now().Format("2006-01-02 15:04")))

	if err := app.Save(record); err != nil {
		log.Printf("❌ [API KEY REQUEST] FAILED: Cannot save API key | User: %s | IP: %s | Error: %v", 
			userEmail, clientIP, err)
		return e.JSON(500, map[string]string{"error": "Failed to save API key"})
	}

	maskedKey := apiKey[:8] + "..."
	log.Printf("✅ [API KEY REQUEST] SUCCESS: Generated API key %s | User: %s | IP: %s", 
		maskedKey, userEmail, clientIP)

	return e.JSON(200, map[string]string{
		"api_key": apiKey,
		"message": "API key generated successfully",
	})
}

// Helper functions

func extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}

func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

func generateAPIKey() string {
	// Generate a secure random API key (simplified for demo)
	hash := sha256.Sum256([]byte(fmt.Sprintf("ramble-ai-%d", time.Now().UnixNano())))
	return "ra-" + hex.EncodeToString(hash[:])[:32]
}

func validateAPIKey(app core.App, apiKey string) (*core.Record, error) {
	keyHash := hashAPIKey(apiKey)
	
	// Find API key record
	apiKeyRecord, err := app.FindFirstRecordByFilter("api_keys", "key_hash = {:hash} && active = true", map[string]interface{}{
		"hash": keyHash,
	})
	if err != nil {
		return nil, fmt.Errorf("API key not found or inactive")
	}

	// Get user record
	userRecord, err := app.FindRecordById("users", apiKeyRecord.GetString("user_id"))
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return userRecord, nil
}

func isUserSubscribed(app core.App, userID string) bool {
	// Check if user has an active subscription using our new system
	subscription, err := stripehandlers.GetUserSubscription(app, userID)
	if err != nil {
		log.Printf("No subscription found for user %s: %v", userID, err)
		return false
	}

	status := subscription.GetString("status")
	return status == "active" || status == "trialing"
}

func proxyToOpenRouter(request *TextProcessingRequest) (*OpenRouterResponse, error) {
	// Build messages array
	messages := []Message{}

	// Add system message if provided
	if request.SystemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: request.SystemPrompt,
		})
	}

	// Add user message
	messages = append(messages, Message{
		Role:    "user",
		Content: request.UserPrompt,
	})

	// Create OpenRouter request
	openRouterReq := OpenRouterRequest{
		Model:    request.Model,
		Messages: messages,
	}

	// Marshal request
	jsonData, err := json.Marshal(openRouterReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// TODO: Get OpenRouter API key from environment or settings
	// For now, this would need to be configured
	openRouterAPIKey := getOpenRouterAPIKey()
	if openRouterAPIKey == "" {
		return nil, fmt.Errorf("OpenRouter API key not configured")
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+openRouterAPIKey)
	req.Header.Set("Content-Type", "application/json")

	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenRouter API error: %s", string(body))
	}

	// Parse response
	var openRouterResp OpenRouterResponse
	err = json.Unmarshal(body, &openRouterResp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if openRouterResp.Error != nil {
		return nil, fmt.Errorf("OpenRouter API error: %s", openRouterResp.Error.Message)
	}

	if len(openRouterResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenRouter API")
	}

	return &openRouterResp, nil
}

func getOpenRouterAPIKey() string {
	// Get OpenRouter API key from environment
	return os.Getenv("OPENROUTER_API_KEY")
}

func logAIUsage(app core.App, userID, userEmail, taskType, model string, tokensUsed, inputSize, outputSize int, duration time.Duration, clientIP string) {
	// Enhanced logging for AI usage analytics and billing
	log.Printf("📊 [AI USAGE] User: %s (%s) | Task: %s | Model: %s | Input: %d | Output: %d | Duration: %v | IP: %s", 
		userEmail, userID, taskType, model, inputSize, outputSize, duration, clientIP)
	
	// TODO: Optionally save to database for analytics/billing
	// This could create records in an "ai_usage_logs" collection:
	/*
	usageCollection, err := app.FindCollectionByNameOrId("ai_usage_logs")
	if err == nil {
		record := core.NewRecord(usageCollection)
		record.Set("user_id", userID)
		record.Set("task_type", taskType)
		record.Set("model", model)
		record.Set("tokens_used", tokensUsed)
		record.Set("input_size", inputSize)
		record.Set("output_size", outputSize)
		record.Set("duration_ms", int(duration.Milliseconds()))
		record.Set("client_ip", clientIP)
		record.Set("timestamp", time.Now())
		app.Save(record)
	}
	*/
}

func getClientIP(e *core.RequestEvent) string {
	// Try to get real IP from common proxy headers
	if ip := e.Request.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip // Cloudflare
	}
	if ip := e.Request.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := e.Request.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For can contain multiple IPs, get the first one
		if ips := strings.Split(ip, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	// Fallback to RemoteAddr
	return e.Request.RemoteAddr
}

// ProcessAudioHandler handles audio transcription requests using PocketBase native file uploads
func ProcessAudioHandler(e *core.RequestEvent, app core.App) error {
	startTime := time.Now()
	clientIP := getClientIP(e)
	userAgent := e.Request.Header.Get("User-Agent")
	
	log.Printf("🎵 [AI AUDIO REQUEST] IP: %s | User-Agent: %s | Method: %s", 
		clientIP, userAgent, e.Request.Method)

	// Validate API key
	apiKey := extractBearerToken(e.Request.Header.Get("Authorization"))
	if apiKey == "" {
		log.Printf("❌ [AI AUDIO REQUEST] FAILED: Missing API key | IP: %s", clientIP)
		return e.JSON(401, map[string]string{"error": "Missing or invalid API key"})
	}

	// Mask API key for logging (show first 8 chars)
	maskedKey := apiKey[:8] + "..."
	log.Printf("🔐 [AI AUDIO REQUEST] API Key: %s | IP: %s", maskedKey, clientIP)

	// Check API key validity and get user
	user, err := validateAPIKey(app, apiKey)
	if err != nil {
		log.Printf("❌ [AI AUDIO REQUEST] FAILED: Invalid API key %s | IP: %s | Error: %v", 
			maskedKey, clientIP, err)
		return e.JSON(401, map[string]string{"error": "Invalid API key"})
	}

	userEmail := user.GetString("email")
	userID := user.Id
	log.Printf("👤 [AI AUDIO REQUEST] User: %s (%s) | API Key: %s | IP: %s", 
		userEmail, userID, maskedKey, clientIP)

	// Check user's subscription status and usage limits
	if !isUserSubscribed(app, userID) {
		log.Printf("❌ [AI AUDIO REQUEST] FAILED: No active subscription | User: %s | IP: %s", 
			userEmail, clientIP)
		return e.JSON(403, map[string]string{"error": "Active subscription required"})
	}

	// Parse multipart form data using PocketBase's capabilities (handles large files)
	err = e.Request.ParseMultipartForm(500 << 20) // 500MB max memory for large audio files, rest goes to disk
	if err != nil {
		log.Printf("❌ [AI AUDIO REQUEST] FAILED: Invalid multipart form | User: %s | IP: %s | Error: %v", 
			userEmail, clientIP, err)
		return e.JSON(400, map[string]string{"error": "Invalid multipart form data"})
	}

	// Get the audio file from form data
	file, header, err := e.Request.FormFile("audio")
	if err != nil {
		log.Printf("❌ [AI AUDIO REQUEST] FAILED: Missing audio file | User: %s | IP: %s | Error: %v", 
			userEmail, clientIP, err)
		return e.JSON(400, map[string]string{"error": "Audio file is required"})
	}
	defer file.Close()

	filename := header.Filename
	fileSize := header.Size
	fileSizeKB := fileSize / 1024
	
	// Check for chunk metadata from form data
	baseFilename := e.Request.FormValue("base_filename")
	isChunk := e.Request.FormValue("is_chunk") == "true"
	isLastChunk := e.Request.FormValue("is_last_chunk") == "true"
	chunkIndex := 0
	if chunkStr := e.Request.FormValue("chunk_index"); chunkStr != "" {
		fmt.Sscanf(chunkStr, "%d", &chunkIndex)
	}
	var originalFileSize int64
	if sizeStr := e.Request.FormValue("original_file_size_bytes"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &originalFileSize)
	}
	var originalDuration float64
	if durStr := e.Request.FormValue("original_duration_seconds"); durStr != "" {
		fmt.Sscanf(durStr, "%f", &originalDuration)
	}
	
	// If not a chunk, use the current filename as base
	if baseFilename == "" {
		baseFilename = filename
	}
	
	if isChunk {
		log.Printf("🎵 [AI AUDIO REQUEST] Processing Chunk | User: %s | Base: %s | Chunk: %d | Size: %d KB | Last: %v | IP: %s", 
			userEmail, baseFilename, chunkIndex, fileSizeKB, isLastChunk, clientIP)
	} else {
		log.Printf("🎵 [AI AUDIO REQUEST] Processing | User: %s | Filename: %s | Audio Size: %d KB | IP: %s", 
			userEmail, filename, fileSizeKB, clientIP)
	}

	// For non-chunks, do preliminary usage validation (estimate duration from file size)
	// This is a rough check - final validation and tracking happens after getting actual duration
	if !isChunk {
		// Rough estimate: 1MB ≈ 1 minute for typical audio files
		estimatedDurationSeconds := float64(fileSize) / 1048576.0 * 60.0
		
		// Check if user can process this estimated duration
		if err := stripehandlers.ValidateUsageLimits(app, userID, estimatedDurationSeconds/3600.0); err != nil {
			log.Printf("❌ [AI AUDIO REQUEST] FAILED: Usage limit exceeded | User: %s | Estimated hours: %.2f | IP: %s | Error: %v", 
				userEmail, estimatedDurationSeconds/3600.0, clientIP, err)
			return e.JSON(403, map[string]string{
				"error": err.Error(),
				"code":  "USAGE_LIMIT_EXCEEDED",
			})
		}
	}

	// Create initial processed_files record with chunk metadata
	processedFileRecord, err := createProcessedFileRecordWithChunkInfo(app, userID, filename, fileSize, clientIP, 
		baseFilename, isChunk, isLastChunk, chunkIndex, originalFileSize, originalDuration)
	if err != nil {
		log.Printf("⚠️  [AI AUDIO REQUEST] Warning: Failed to create processed_files record | User: %s | Error: %v", 
			userEmail, err)
		// Continue processing even if logging fails
	}

	// Process audio using OpenAI Whisper API
	result, err := streamToOpenAIWhisper(file, filename)
	if err != nil {
		elapsed := time.Since(startTime)
		
		// Update processed_files record with failure
		if processedFileRecord != nil {
			updateProcessedFileRecord(app, processedFileRecord, "failed", 0, 0, 0, elapsed.Milliseconds())
		}
		
		log.Printf("❌ [AI AUDIO REQUEST] FAILED: Transcription error | User: %s | Filename: %s | Duration: %v | IP: %s | Error: %v", 
			userEmail, filename, elapsed, clientIP, err)
		return e.JSON(500, map[string]string{"error": fmt.Sprintf("Transcription failed: %v", err)})
	}

	elapsed := time.Since(startTime)
	transcriptLength := len(result.Transcript)
	wordCount := len(result.Words)
	
	// Update processed_files record with success
	if processedFileRecord != nil {
		updateProcessedFileRecord(app, processedFileRecord, "completed", result.Duration, transcriptLength, wordCount, elapsed.Milliseconds())
		
		// If this is the last chunk, flatten all chunks into a single record
		if isLastChunk {
			if err := flattenChunkedRecords(app, userID, baseFilename, originalFileSize, originalDuration); err != nil {
				log.Printf("⚠️  [AI AUDIO REQUEST] Warning: Failed to flatten chunk records | User: %s | Base: %s | Error: %v", 
					userEmail, baseFilename, err)
				// Don't fail the request, just log the warning
			} else {
				log.Printf("✅ [AI AUDIO REQUEST] Flattened chunks | User: %s | Base: %s", userEmail, baseFilename)
			}
		}
	}

	// Update usage tracking for non-chunks (for chunks, usage is tracked when flattened)
	if !isChunk {
		if err := stripehandlers.UpdateUsageAfterProcessing(app, userID, result.Duration); err != nil {
			log.Printf("⚠️  [AI AUDIO REQUEST] Warning: Failed to update usage tracking | User: %s | Duration: %.2fs | Error: %v", 
				userEmail, result.Duration, err)
			// Don't fail the request if usage tracking fails
		} else {
			log.Printf("📊 [AI AUDIO REQUEST] Usage updated | User: %s | Duration: %.2fs (%.3f hours)", 
				userEmail, result.Duration, result.Duration/3600.0)
		}
	}
	
	// Log usage and success
	logAIUsage(app, userID, userEmail, "transcription", "whisper-1", 0, int(fileSizeKB), transcriptLength, elapsed, clientIP)
	
	if isChunk {
		log.Printf("✅ [AI AUDIO REQUEST] CHUNK SUCCESS | User: %s | Base: %s | Chunk: %d | Transcript: %d chars | Duration: %v | IP: %s", 
			userEmail, baseFilename, chunkIndex, transcriptLength, elapsed, clientIP)
	} else {
		log.Printf("✅ [AI AUDIO REQUEST] SUCCESS | User: %s | Filename: %s | Audio: %d KB | Transcript: %d chars | Words: %d | Duration: %v | IP: %s", 
			userEmail, filename, fileSizeKB, transcriptLength, wordCount, elapsed, clientIP)
	}

	return e.JSON(200, result)
}

// streamToOpenAIWhisper streams audio directly to OpenAI's Whisper API without temp files
func streamToOpenAIWhisper(audioFile multipart.File, filename string) (*AudioProcessingResult, error) {
	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured")
	}

	// Create a pipe for streaming multipart data to OpenAI
	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)

	// Start goroutine to write multipart data
	go func() {
		defer pipeWriter.Close()
		defer multipartWriter.Close()

		// Add file field - stream directly from input
		fileWriter, err := multipartWriter.CreateFormFile("file", filepath.Base(filename))
		if err != nil {
			pipeWriter.CloseWithError(fmt.Errorf("failed to create form file: %w", err))
			return
		}

		// Stream file contents directly from input to OpenAI
		_, err = io.Copy(fileWriter, audioFile)
		if err != nil {
			pipeWriter.CloseWithError(fmt.Errorf("failed to copy file: %w", err))
			return
		}

		// Add model field
		if err := multipartWriter.WriteField("model", "whisper-1"); err != nil {
			pipeWriter.CloseWithError(fmt.Errorf("failed to write model field: %w", err))
			return
		}

		// Add response format for verbose JSON with timestamps
		if err := multipartWriter.WriteField("response_format", "verbose_json"); err != nil {
			pipeWriter.CloseWithError(fmt.Errorf("failed to write response_format field: %w", err))
			return
		}

		// Add timestamp granularities for word-level timestamps
		if err := multipartWriter.WriteField("timestamp_granularities[]", "word"); err != nil {
			pipeWriter.CloseWithError(fmt.Errorf("failed to write timestamp_granularities field: %w", err))
			return
		}
	}()

	// Create request with streaming body
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", pipeReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	// Make request
	client := &http.Client{Timeout: 120 * time.Second} // Longer timeout for large files
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var transcriptionResp OpenAITranscriptionResponse
	if err := json.Unmarshal(body, &transcriptionResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &AudioProcessingResult{
		Transcript: transcriptionResp.Text,
		Duration:   transcriptionResp.Duration,
		Language:   transcriptionResp.Language,
		Words:      transcriptionResp.Words,
		Segments:   transcriptionResp.Segments,
	}, nil
}

// createProcessedFileRecordWithChunkInfo creates a new record in processed_files collection with chunk metadata
func createProcessedFileRecordWithChunkInfo(app core.App, userID, filename string, fileSizeBytes int64, clientIP string,
	baseFilename string, isChunk, isLastChunk bool, chunkIndex int, originalFileSize int64, originalDuration float64) (*core.Record, error) {
	
	collection, err := app.FindCollectionByNameOrId("processed_files")
	if err != nil {
		return nil, fmt.Errorf("failed to find processed_files collection: %w", err)
	}

	// For non-chunks, check existing processing count
	if !isChunk {
		existingRecords, err := app.FindRecordsByFilter("processed_files", 
			fmt.Sprintf("user_id = '%s' && filename = '%s' && is_chunk = false", userID, filename), 
			"", 0, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to query existing processed files: %w", err)
		}

		processingCount := len(existingRecords) + 1
		if processingCount > 2 {
			return nil, fmt.Errorf("maximum processing limit reached for file '%s' (limit: 2 attempts)", filename)
		}

		log.Printf("📊 [PROCESSING COUNT] User: %s | Filename: %s | Attempt: %d/2 | IP: %s", 
			userID, filename, processingCount, clientIP)
	}

	record := core.NewRecord(collection)
	record.Set("user_id", userID)
	record.Set("filename", filename)
	record.Set("file_size_bytes", fileSizeBytes)
	record.Set("status", "processing")
	record.Set("model_used", "whisper-1")
	record.Set("client_ip", clientIP)
	
	// Set chunk metadata
	record.Set("base_filename", baseFilename)
	record.Set("is_chunk", isChunk)
	record.Set("is_last_chunk", isLastChunk)
	if isChunk {
		record.Set("chunk_index", chunkIndex)
		record.Set("processing_count", 1) // Chunks always count as 1
	}
	if isLastChunk && originalFileSize > 0 {
		record.Set("original_file_size_bytes", originalFileSize)
		record.Set("original_duration_seconds", originalDuration)
	}

	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save processed_files record: %w", err)
	}

	return record, nil
}

// updateProcessedFileRecord updates an existing processed_files record with completion data
func updateProcessedFileRecord(app core.App, record *core.Record, status string, durationSeconds float64, transcriptLength, wordsCount int, processingTimeMs int64) error {
	record.Set("status", status)
	record.Set("duration_seconds", durationSeconds)
	record.Set("transcript_length", transcriptLength)
	record.Set("words_count", wordsCount)
	record.Set("processing_time_ms", processingTimeMs)

	if err := app.Save(record); err != nil {
		return fmt.Errorf("failed to update processed_files record: %w", err)
	}

	return nil
}

// flattenChunkedRecords consolidates all chunk records into a single record after last chunk is processed
func flattenChunkedRecords(app core.App, userID, baseFilename string, originalFileSize int64, originalDuration float64) error {
	// Find all chunk records for this base filename
	filter := fmt.Sprintf("user_id = '%s' && base_filename = '%s' && is_chunk = true && status = 'completed'", userID, baseFilename)
	chunkRecords, err := app.FindRecordsByFilter("processed_files", filter, "chunk_index ASC", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to find chunk records: %w", err)
	}

	if len(chunkRecords) == 0 {
		return fmt.Errorf("no completed chunks found for base file: %s", baseFilename)
	}

	log.Printf("📊 [FLATTEN CHUNKS] Found %d chunks for file: %s | User: %s", len(chunkRecords), baseFilename, userID)

	// Aggregate data from all chunks
	var totalTranscriptLength int64
	var totalWordsCount int64
	var totalProcessingTimeMs int64
	var clientIP string

	for _, chunk := range chunkRecords {
		totalTranscriptLength += int64(chunk.GetInt("transcript_length"))
		totalWordsCount += int64(chunk.GetInt("words_count"))
		totalProcessingTimeMs += int64(chunk.GetInt("processing_time_ms"))
		if clientIP == "" {
			clientIP = chunk.GetString("client_ip")
		}
	}

	// Create the consolidated record
	collection, err := app.FindCollectionByNameOrId("processed_files")
	if err != nil {
		return fmt.Errorf("failed to find processed_files collection: %w", err)
	}

	consolidatedRecord := core.NewRecord(collection)
	consolidatedRecord.Set("user_id", userID)
	consolidatedRecord.Set("filename", baseFilename)
	consolidatedRecord.Set("file_size_bytes", originalFileSize)
	consolidatedRecord.Set("duration_seconds", originalDuration)
	consolidatedRecord.Set("processing_time_ms", totalProcessingTimeMs)
	consolidatedRecord.Set("status", "completed")
	consolidatedRecord.Set("transcript_length", totalTranscriptLength)
	consolidatedRecord.Set("words_count", totalWordsCount)
	consolidatedRecord.Set("model_used", "whisper-1")
	consolidatedRecord.Set("client_ip", clientIP)
	consolidatedRecord.Set("base_filename", baseFilename)
	consolidatedRecord.Set("is_chunk", false)
	consolidatedRecord.Set("chunk_index", len(chunkRecords)) // Store total chunk count for reference
	consolidatedRecord.Set("processing_count", 1)

	if err := app.Save(consolidatedRecord); err != nil {
		return fmt.Errorf("failed to save consolidated record: %w", err)
	}

	log.Printf("✅ [FLATTEN CHUNKS] Created consolidated record | File: %s | Chunks: %d | Total Duration: %.1fs | Total Words: %d", 
		baseFilename, len(chunkRecords), originalDuration, totalWordsCount)

	// Delete the individual chunk records
	for _, chunk := range chunkRecords {
		if err := app.Delete(chunk); err != nil {
			log.Printf("⚠️  [FLATTEN CHUNKS] Failed to delete chunk record %s: %v", chunk.Id, err)
			// Continue deleting other chunks even if one fails
		}
	}

	log.Printf("🗑️  [FLATTEN CHUNKS] Deleted %d chunk records for file: %s", len(chunkRecords), baseFilename)

	return nil
}

// UsageSummaryHandler provides aggregated usage statistics for authenticated users via API key
func UsageSummaryHandler(e *core.RequestEvent, app core.App) error {
	clientIP := getClientIP(e)
	userAgent := e.Request.Header.Get("User-Agent")
	
	log.Printf("📊 [USAGE SUMMARY REQUEST] IP: %s | User-Agent: %s", clientIP, userAgent)

	// Validate API key
	apiKey := extractBearerToken(e.Request.Header.Get("Authorization"))
	if apiKey == "" {
		log.Printf("❌ [USAGE SUMMARY REQUEST] FAILED: Missing API key | IP: %s", clientIP)
		return e.JSON(401, map[string]string{"error": "Missing or invalid API key"})
	}

	user, err := validateAPIKey(app, apiKey)
	if err != nil {
		maskedKey := apiKey[:8] + "..."
		log.Printf("❌ [USAGE SUMMARY REQUEST] FAILED: Invalid API key %s | IP: %s", maskedKey, clientIP)
		return e.JSON(401, map[string]string{"error": "Invalid API key"})
	}

	userEmail := user.GetString("email")
	userID := user.Id

	// Get month parameter (optional, defaults to current month)
	month := e.Request.URL.Query().Get("month") // Format: YYYY-MM

	// Query processed files for user (exclude chunk records)
	filter := fmt.Sprintf("user_id = '%s' && (is_chunk = false || is_chunk = '')", userID)
	log.Printf("🔍 [USAGE SUMMARY] Querying summary for user: %s with filter: %s", userID, filter)
	if month != "" {
		// Add month filter if specified
		filter += fmt.Sprintf(" && created >= '%s-01 00:00:00' && created < '%s-01 00:00:00'", month, getNextMonth(month))
	}

	records, err := app.FindRecordsByFilter("processed_files", filter, "", 0, 0)
	if err != nil {
		log.Printf("❌ [USAGE SUMMARY REQUEST] FAILED: Database query error | User: %s | Error: %v", userEmail, err)
		return e.JSON(500, map[string]string{"error": "Failed to retrieve usage data"})
	}
	
	log.Printf("📊 [USAGE SUMMARY] Found %d records for summary | User: %s", len(records), userEmail)

	// Aggregate statistics
	summary := calculateUsageSummary(records)
	summary["user_id"] = userID
	summary["period"] = month
	if month == "" {
		summary["period"] = "all_time"
	}

	log.Printf("✅ [USAGE SUMMARY REQUEST] SUCCESS | User: %s | Records: %d | Period: %s | IP: %s", 
		userEmail, len(records), summary["period"], clientIP)

	return e.JSON(200, summary)
}

// UsageFilesHandler provides detailed list of processed files for authenticated users via API key
func UsageFilesHandler(e *core.RequestEvent, app core.App) error {
	_ = getClientIP(e) // Get client IP for potential logging
	
	// Validate API key
	apiKey := extractBearerToken(e.Request.Header.Get("Authorization"))
	if apiKey == "" {
		return e.JSON(401, map[string]string{"error": "Missing or invalid API key"})
	}

	user, err := validateAPIKey(app, apiKey)
	if err != nil {
		return e.JSON(401, map[string]string{"error": "Invalid API key"})
	}

	userID := user.Id

	// Parse pagination parameters
	page := 1
	perPage := 50
	if p := e.Request.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if pp := e.Request.URL.Query().Get("per_page"); pp != "" {
		fmt.Sscanf(pp, "%d", &perPage)
		if perPage > 100 {
			perPage = 100 // Max 100 per page
		}
	}

	// Query processed files (exclude chunk records) - get records where is_chunk is false or empty
	filter := fmt.Sprintf("user_id = '%s' && (is_chunk = false || is_chunk = '')", userID)
	
	// Add debug logging for troubleshooting
	log.Printf("🔍 [USAGE FILES] Querying files for user: %s with filter: %s", userID, filter)
	sort := "" // No sorting for now to avoid created field issues
	
	records, err := app.FindRecordsByFilter("processed_files", filter, sort, perPage, (page-1)*perPage)
	if err != nil {
		log.Printf("❌ [USAGE FILES] Database query failed: %v", err)
		return e.JSON(500, map[string]string{"error": "Failed to retrieve files data"})
	}
	
	log.Printf("📊 [USAGE FILES] Found %d records for user %s", len(records), userID)

	// Convert to response format
	files := make([]map[string]interface{}, len(records))
	for i, record := range records {
		files[i] = map[string]interface{}{
			"id":                 record.Id,
			"filename":           record.GetString("filename"),
			"file_size_bytes":    record.GetInt("file_size_bytes"),
			"duration_seconds":   record.GetFloat("duration_seconds"),
			"processing_time_ms": record.GetInt("processing_time_ms"),
			"processing_count":   record.GetInt("processing_count"),
			"status":            record.GetString("status"),
			"transcript_length": record.GetInt("transcript_length"),
			"words_count":       record.GetInt("words_count"),
			"model_used":        record.GetString("model_used"),
			"created":           record.GetDateTime("created"),
			"updated":           record.GetDateTime("updated"),
		}
	}

	// Get total count for pagination
	totalRecords := int64(0)
	if allRecords, err := app.FindRecordsByFilter("processed_files", filter, "", 0, 0); err == nil {
		totalRecords = int64(len(allRecords))
	} else {
		totalRecords = int64(len(records)) // Fallback
	}

	response := map[string]interface{}{
		"files":        files,
		"page":         page,
		"per_page":     perPage,
		"total":        totalRecords,
		"total_pages":  (totalRecords + int64(perPage) - 1) / int64(perPage),
	}
	
	log.Printf("✅ [USAGE FILES] Returning %d files to user %s", len(files), userID)

	return e.JSON(200, response)
}

// UsageStatsHandler provides current usage statistics for authenticated users via API key
func UsageStatsHandler(e *core.RequestEvent, app core.App) error {
	_ = getClientIP(e) // Get client IP for potential logging
	
	// Validate API key
	apiKey := extractBearerToken(e.Request.Header.Get("Authorization"))
	if apiKey == "" {
		return e.JSON(401, map[string]string{"error": "Missing or invalid API key"})
	}

	user, err := validateAPIKey(app, apiKey)
	if err != nil {
		return e.JSON(401, map[string]string{"error": "Invalid API key"})
	}

	userID := user.Id

	// Get current month and last month
	now := time.Now()
	currentMonth := now.Format("2006-01")
	lastMonth := now.AddDate(0, -1, 0).Format("2006-01")

	// Query current month (exclude chunk records)
	currentFilter := fmt.Sprintf("user_id = '%s' && (is_chunk = false || is_chunk = '') && created >= '%s-01 00:00:00' && created < '%s-01 00:00:00'", 
		userID, currentMonth, getNextMonth(currentMonth))
	currentRecords, _ := app.FindRecordsByFilter("processed_files", currentFilter, "", 0, 0)
	
	// Query last month (exclude chunk records)
	lastFilter := fmt.Sprintf("user_id = '%s' && (is_chunk = false || is_chunk = '') && created >= '%s-01 00:00:00' && created < '%s-01 00:00:00'", 
		userID, lastMonth, currentMonth)
	lastRecords, _ := app.FindRecordsByFilter("processed_files", lastFilter, "", 0, 0)

	// Calculate stats
	currentStats := calculateUsageSummary(currentRecords)
	lastStats := calculateUsageSummary(lastRecords)

	response := map[string]interface{}{
		"current_month": map[string]interface{}{
			"period": currentMonth,
			"stats":  currentStats,
		},
		"last_month": map[string]interface{}{
			"period": lastMonth,
			"stats":  lastStats,
		},
		"comparison": map[string]interface{}{
			"files_change":    currentStats["total_files"].(int) - lastStats["total_files"].(int),
			"duration_change": currentStats["total_duration"].(float64) - lastStats["total_duration"].(float64),
		},
	}

	return e.JSON(200, response)
}

// Helper functions for usage calculations

func calculateUsageSummary(records []*core.Record) map[string]interface{} {
	totalFiles := len(records)
	totalDuration := 0.0
	totalFileSize := int64(0)
	totalProcessingTime := int64(0)
	statusCounts := map[string]int{
		"completed":  0,
		"processing": 0,
		"failed":     0,
	}

	for _, record := range records {
		totalDuration += record.GetFloat("duration_seconds")
		totalFileSize += int64(record.GetInt("file_size_bytes"))
		totalProcessingTime += int64(record.GetInt("processing_time_ms"))
		
		status := record.GetString("status")
		if count, exists := statusCounts[status]; exists {
			statusCounts[status] = count + 1
		}
	}

	// Convert duration to minutes and hours
	totalMinutes := totalDuration / 60
	totalHours := totalMinutes / 60

	return map[string]interface{}{
		"total_files":              totalFiles,
		"total_duration_seconds":   totalDuration,
		"total_duration_minutes":   totalMinutes,
		"total_duration_hours":     totalHours,
		"total_file_size_bytes":    totalFileSize,
		"total_file_size_mb":       float64(totalFileSize) / (1024 * 1024),
		"total_processing_time_ms": totalProcessingTime,
		"avg_processing_time_ms":   func() float64 {
			if totalFiles > 0 {
				return float64(totalProcessingTime) / float64(totalFiles)
			}
			return 0
		}(),
		"status_breakdown": statusCounts,
		"success_rate": func() float64 {
			if totalFiles > 0 {
				return float64(statusCounts["completed"]) / float64(totalFiles) * 100
			}
			return 0
		}(),
	}
}

func getNextMonth(month string) string {
	// Parse YYYY-MM format and return next month
	if len(month) != 7 {
		return month
	}
	
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	
	nextMonth := t.AddDate(0, 1, 0)
	return nextMonth.Format("2006-01")
}



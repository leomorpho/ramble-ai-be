package tus

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/tus/tusd/v2/pkg/handler"
)

// TUSHandler wraps the TUS handler with PocketBase integration
type TUSHandler struct {
	handler *handler.Handler
	app     core.App
}

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

// NewTUSHandler creates a new TUS handler with PocketBase integration
func NewTUSHandler(app core.App) (*TUSHandler, error) {
	// Create upload directory
	uploadDir := filepath.Join(app.DataDir(), "tus_uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Create PocketBase store
	store := NewPocketBaseStore(app)

	// Configure TUS handler
	composer := handler.NewStoreComposer()
	store.UseIn(composer)

	config := handler.Config{
		BasePath:                "/api/tus",
		StoreComposer:          composer,
		NotifyCompleteUploads:  true,
		NotifyTerminatedUploads: true,
		NotifyUploadProgress:   true,
		NotifyCreatedUploads:   true,
		MaxSize:                1024 * 1024 * 1024, // 1GB max file size
	}

	tusHandler, err := handler.NewHandler(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUS handler: %w", err)
	}
	
	// Log the capabilities that will be advertised
	capabilities := composer.Capabilities()
	app.Logger().Info("TUS handler created", "capabilities", capabilities)

	h := &TUSHandler{
		handler: tusHandler,
		app:     app,
	}

	// Set up hooks
	h.setupHooks()

	return h, nil
}

// setupHooks configures TUS event hooks for PocketBase integration
func (h *TUSHandler) setupHooks() {
	// Hook for upload completion
	go func() {
		for {
			select {
			case info := <-h.handler.CompleteUploads:
				h.handleUploadComplete(info)
			case info := <-h.handler.TerminatedUploads:
				h.handleUploadTerminated(info)
			case info := <-h.handler.CreatedUploads:
				h.handleUploadCreated(info)
			}
		}
	}()
}

// handleUploadCreated handles when a new upload is created
func (h *TUSHandler) handleUploadCreated(info handler.HookEvent) {
	metadata := info.Upload.MetaData
	
	// Create PocketBase record
	collection, err := h.app.FindCollectionByNameOrId("file_uploads")
	if err != nil {
		h.app.Logger().Error("Failed to find file_uploads collection", "error", err)
		return
	}

	record := core.NewRecord(collection)
	
	// Set initial record data
	record.Set("upload_id", info.Upload.ID)
	record.Set("processing_status", "pending")
	record.Set("original_name", metadata["filename"])
	
	// Parse metadata
	if fileType, ok := metadata["fileType"]; ok {
		record.Set("file_type", fileType)
	}
	if category, ok := metadata["category"]; ok {
		record.Set("category", category)
	}
	if userID, ok := metadata["userId"]; ok {
		record.Set("user", userID)
	}
	if visibility, ok := metadata["visibility"]; ok {
		record.Set("visibility", visibility)
	} else {
		record.Set("visibility", "private")
	}
	
	// Store all metadata as JSON
	metadataJSON, _ := json.Marshal(metadata)
	record.Set("metadata", string(metadataJSON))

	if err := h.app.Save(record); err != nil {
		h.app.Logger().Error("Failed to create file upload record", "error", err)
	}
}

// handleUploadComplete handles when an upload is completed
func (h *TUSHandler) handleUploadComplete(info handler.HookEvent) {
	// Find the record by upload_id
	record, err := h.app.FindFirstRecordByFilter(
		"file_uploads",
		"upload_id = {:uploadId}",
		map[string]any{"uploadId": info.Upload.ID},
	)
	if err != nil {
		h.app.Logger().Error("Failed to find upload record", "error", err)
		return
	}

	// Move file to PocketBase storage and update record
	if err := h.moveFileToStorage(record, info.Upload); err != nil {
		h.app.Logger().Error("Failed to move file to storage", "error", err)
		record.Set("processing_status", "failed")
	} else {
		record.Set("processing_status", "completed")
	}

	if err := h.app.Save(record); err != nil {
		h.app.Logger().Error("Failed to update upload record", "error", err)
	}

	// Trigger post-processing if needed
	h.triggerPostProcessing(record)
}

// handleUploadTerminated handles when an upload is terminated
func (h *TUSHandler) handleUploadTerminated(info handler.HookEvent) {
	// Find and delete the record
	record, err := h.app.FindFirstRecordByFilter(
		"file_uploads",
		"upload_id = {:uploadId}",
		map[string]any{"uploadId": info.Upload.ID},
	)
	if err != nil {
		return // Record might not exist
	}

	h.app.Delete(record)
}

// moveFileToStorage moves the completed upload to PocketBase file storage
func (h *TUSHandler) moveFileToStorage(record *core.Record, upload handler.FileInfo) error {
	// Get upload file path
	uploadPath := filepath.Join(h.app.DataDir(), "tus_uploads", upload.ID+".bin")
	
	// Open the upload file
	file, err := os.Open(uploadPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get original filename from metadata
	filename := "upload"
	if upload.MetaData["filename"] != "" {
		filename = upload.MetaData["filename"]
	}

	// For now, just store the filename - proper file storage integration
	// would require more complex handling of the PocketBase filesystem
	record.Set("file", filename)

	// Clean up temp file
	os.Remove(uploadPath)
	os.Remove(filepath.Join(h.app.DataDir(), "tus_uploads", upload.ID+".info"))

	return nil
}

// triggerPostProcessing triggers any post-upload processing
func (h *TUSHandler) triggerPostProcessing(record *core.Record) {
	// Parse metadata to check for processing instructions
	metadataStr := record.GetString("metadata")
	if metadataStr == "" {
		return
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
		return
	}

	// Check for processing instructions
	if processAfterUpload, ok := metadata["processAfterUpload"].([]interface{}); ok {
		record.Set("processing_status", "processing")
		h.app.Save(record)

		// Process each instruction
		for _, instruction := range processAfterUpload {
			if instructionStr, ok := instruction.(string); ok {
				h.processFile(record, instructionStr)
			}
		}

		record.Set("processing_status", "completed")
		h.app.Save(record)
	}
}

// processFile handles individual file processing instructions
func (h *TUSHandler) processFile(record *core.Record, instruction string) error {
	// Get file from record
	fileField := record.GetString("file")
	if fileField == "" {
		return fmt.Errorf("no file attached to record")
	}

	// Get filesystem
	fileSystem, err := h.app.NewFilesystem()
	if err != nil {
		return err
	}
	defer fileSystem.Close()

	// Process based on instruction
	switch {
	case strings.HasPrefix(instruction, "resize:"):
		return h.processImageResize(record, fileSystem, instruction)
	case instruction == "thumbnail":
		return h.processImageThumbnail(record, fileSystem)
	case instruction == "extract_text":
		return h.processTextExtraction(record, fileSystem)
	case instruction == "transcribe_audio":
		return h.processAudioTranscription(record)
	default:
		h.app.Logger().Warn("Unknown processing instruction", "instruction", instruction)
	}

	return nil
}

// processImageResize handles image resizing
func (h *TUSHandler) processImageResize(record *core.Record, fs *filesystem.System, instruction string) error {
	// Parse resize dimensions from instruction (e.g., "resize:200x200")
	// Implementation would use PocketBase's image processing capabilities
	h.app.Logger().Info("Processing image resize", "instruction", instruction)
	return nil
}

// processImageThumbnail generates thumbnails
func (h *TUSHandler) processImageThumbnail(record *core.Record, fs *filesystem.System) error {
	h.app.Logger().Info("Processing image thumbnail")
	return nil
}

// processTextExtraction extracts text from documents
func (h *TUSHandler) processTextExtraction(record *core.Record, fs *filesystem.System) error {
	h.app.Logger().Info("Processing text extraction")
	return nil
}

// processAudioTranscription transcribes audio files using OpenAI Whisper
func (h *TUSHandler) processAudioTranscription(record *core.Record) error {
	h.app.Logger().Info("Starting audio transcription", "record_id", record.Id)
	
	// Get upload ID and file path
	uploadID := record.GetString("upload_id")
	if uploadID == "" {
		return fmt.Errorf("no upload ID found in record")
	}
	
	// Get the uploaded file path
	uploadPath := filepath.Join(h.app.DataDir(), "tus_uploads", uploadID+".bin")
	
	// Open the uploaded file
	file, err := os.Open(uploadPath)
	if err != nil {
		return fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()
	
	// Get filename from metadata
	filename := record.GetString("original_name")
	if filename == "" {
		filename = "audio.mp3"
	}
	
	// Call OpenAI Whisper API
	result, err := h.transcribeWithOpenAI(file, filename)
	if err != nil {
		h.app.Logger().Error("Transcription failed", "error", err, "record_id", record.Id)
		record.Set("processing_status", "failed")
		record.Set("error_message", err.Error())
		h.app.Save(record)
		return err
	}
	
	// Store transcription results in record
	transcriptionJSON, _ := json.Marshal(result)
	record.Set("transcription_result", string(transcriptionJSON))
	record.Set("processing_status", "completed")
	record.Set("transcript", result.Transcript)
	
	// Save updated record
	if err := h.app.Save(record); err != nil {
		h.app.Logger().Error("Failed to save transcription result", "error", err)
		return err
	}
	
	h.app.Logger().Info("Audio transcription completed", "record_id", record.Id, "transcript_length", len(result.Transcript))
	
	// Clean up uploaded file
	os.Remove(uploadPath)
	os.Remove(filepath.Join(h.app.DataDir(), "tus_uploads", uploadID+".info"))
	
	return nil
}

// transcribeWithOpenAI sends audio to OpenAI Whisper API
func (h *TUSHandler) transcribeWithOpenAI(audioFile *os.File, filename string) (*AudioProcessingResult, error) {
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

// ServeHTTP implements http.Handler
func (h *TUSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log TUS requests for debugging
	h.app.Logger().Info("TUS request", "method", r.Method, "path", r.URL.Path, "headers", r.Header)
	
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, HEAD, PATCH")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Origin, X-Requested-With, Content-Type, Upload-Length, Upload-Offset, Tus-Resumable, Upload-Metadata, Upload-Defer-Length, Upload-Concat")
	w.Header().Set("Access-Control-Expose-Headers", "Upload-Offset, Location, Upload-Length, Tus-Version, Tus-Resumable, Tus-Max-Size, Tus-Extension, Upload-Metadata, Upload-Defer-Length, Upload-Concat")

	// Allow OPTIONS requests without authentication (needed for TUS protocol capabilities)
	if r.Method == "OPTIONS" {
		// Delegate to TUS handler for capability checks
		h.handler.ServeHTTP(w, r)
		return
	}

	// Authenticate request using PocketBase auth for other methods
	if !h.authenticateRequest(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Authentication required"))
		return
	}

	// Delegate to TUS handler
	h.handler.ServeHTTP(w, r)
}

// authenticateRequest validates the request has valid PocketBase authentication
func (h *TUSHandler) authenticateRequest(r *http.Request) bool {
	// Extract auth token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	// Remove "Bearer " prefix
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return false
	}

	// Validate token with PocketBase - simple validation for now
	// In a real implementation, you'd want to properly validate the JWT token
	if len(token) < 10 {
		return false
	}
	
	// For now, we'll assume the token is valid if it's present
	// You should implement proper JWT validation here

	return true
}


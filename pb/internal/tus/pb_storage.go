package tus

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"
	"github.com/tus/tusd/v2/pkg/handler"
)

// PocketBaseStore implements tusd.DataStore using PocketBase's file storage
type PocketBaseStore struct {
	app core.App
}

// NewPocketBaseStore creates a new PocketBase storage backend for TUS
func NewPocketBaseStore(app core.App) *PocketBaseStore {
	return &PocketBaseStore{
		app: app,
	}
}

// NewUpload creates a new upload and returns its upload id
func (store *PocketBaseStore) NewUpload(ctx context.Context, info handler.FileInfo) (handler.Upload, error) {
	id := info.ID
	
	// Log the creation for debugging
	store.app.Logger().Info("Creating new TUS upload", "id", id, "size", info.Size, "metadata", info.MetaData)
	
	// Create the upload directory in PocketBase's storage
	uploadPath := store.getUploadPath(id)
	if err := os.MkdirAll(filepath.Dir(uploadPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}
	
	// Create the upload file
	file, err := os.OpenFile(uploadPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload file: %w", err)
	}
	file.Close()
	
	// Create info file to store upload metadata
	infoPath := store.getInfoPath(id)
	if err := store.writeInfo(infoPath, info); err != nil {
		return nil, fmt.Errorf("failed to write upload info: %w", err)
	}
	
	upload := &PocketBaseUpload{
		store: store,
		id:    id,
		info:  info,
	}
	
	store.app.Logger().Info("TUS upload created successfully", "id", id, "path", uploadPath)
	
	return upload, nil
}

// GetUpload retrieves an existing upload
func (store *PocketBaseStore) GetUpload(ctx context.Context, id string) (handler.Upload, error) {
	infoPath := store.getInfoPath(id)
	
	info, err := store.readInfo(infoPath)
	if err != nil {
		return nil, err
	}
	
	upload := &PocketBaseUpload{
		store: store,
		id:    id,
		info:  info,
	}
	
	return upload, nil
}

// AsTerminatableUpload returns the upload as a terminatable upload
func (store *PocketBaseStore) AsTerminatableUpload(upload handler.Upload) handler.TerminatableUpload {
	return upload.(*PocketBaseUpload)
}

// AsLengthDeclarableUpload returns the upload as a length declarable upload
func (store *PocketBaseStore) AsLengthDeclarableUpload(upload handler.Upload) handler.LengthDeclarableUpload {
	return upload.(*PocketBaseUpload)
}

// AsConcatableUpload returns the upload as a concatenatable upload
func (store *PocketBaseStore) AsConcatableUpload(upload handler.Upload) handler.ConcatableUpload {
	return upload.(*PocketBaseUpload)
}

// getUploadPath returns the file path for storing the upload data
func (store *PocketBaseStore) getUploadPath(id string) string {
	// Store uploads in pb_data/tus_uploads/
	return filepath.Join(store.app.DataDir(), "tus_uploads", id+".bin")
}

// getInfoPath returns the file path for storing upload metadata
func (store *PocketBaseStore) getInfoPath(id string) string {
	return filepath.Join(store.app.DataDir(), "tus_uploads", id+".info")
}

// writeInfo writes upload info to file
func (store *PocketBaseStore) writeInfo(path string, info handler.FileInfo) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Simple JSON-like format for storing file info
	content := fmt.Sprintf(`{
		"ID": "%s",
		"Size": %d,
		"Offset": %d,
		"MetaData": %q,
		"IsPartial": %t,
		"IsFinal": %t,
		"PartialUploads": %q
	}`, info.ID, info.Size, info.Offset, formatMetadata(info.MetaData), 
		info.IsPartial, info.IsFinal, formatPartialUploads(info.PartialUploads))
	
	_, err = file.WriteString(content)
	return err
}

// readInfo reads upload info from file
func (store *PocketBaseStore) readInfo(path string) (handler.FileInfo, error) {
	var info handler.FileInfo
	
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return info, handler.ErrNotFound
	}
	
	file, err := os.Open(path)
	if err != nil {
		return info, err
	}
	defer file.Close()
	
	// Read and parse the info (simplified parsing)
	content, err := io.ReadAll(file)
	if err != nil {
		return info, err
	}
	
	// For simplicity, we'll parse basic info
	// In production, you might want to use proper JSON parsing
	info.ID = extractValue(string(content), "ID")
	
	return info, nil
}

// Helper functions for formatting metadata
func formatMetadata(meta map[string]string) string {
	result := "{"
	for k, v := range meta {
		result += fmt.Sprintf(`"%s":"%s",`, k, v)
	}
	if len(meta) > 0 {
		result = result[:len(result)-1] // Remove trailing comma
	}
	result += "}"
	return result
}

func formatPartialUploads(uploads []string) string {
	result := "["
	for i, upload := range uploads {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`"%s"`, upload)
	}
	result += "]"
	return result
}

func extractValue(content, key string) string {
	// Simplified extraction - in production use proper JSON parsing
	return ""
}

// UseIn implements the store interface for TUS composer
func (store *PocketBaseStore) UseIn(composer *handler.StoreComposer) {
	// Core functionality (required for basic TUS operations including creation)
	composer.UseCore(store)
	
	// Enable termination extension (allows deleting uploads)
	composer.UseTerminater(store)
	
	// Enable length deferrer extension (allows uploads with unknown size initially)
	composer.UseLengthDeferrer(store)
	
	// Enable concatenation extension (allows combining partial uploads)
	composer.UseConcater(store)
}
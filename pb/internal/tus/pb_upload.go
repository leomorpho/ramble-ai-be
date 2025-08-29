package tus

import (
	"context"
	"io"
	"os"

	"github.com/tus/tusd/v2/pkg/handler"
)

// PocketBaseUpload implements the handler.Upload interface
type PocketBaseUpload struct {
	store *PocketBaseStore
	id    string
	info  handler.FileInfo
}

// GetInfo returns information about the upload
func (upload *PocketBaseUpload) GetInfo(ctx context.Context) (handler.FileInfo, error) {
	// Refresh info from storage
	infoPath := upload.store.getInfoPath(upload.id)
	info, err := upload.store.readInfo(infoPath)
	if err != nil {
		return upload.info, err
	}
	
	// Update current offset by checking file size
	uploadPath := upload.store.getUploadPath(upload.id)
	if stat, err := os.Stat(uploadPath); err == nil {
		info.Offset = stat.Size()
	}
	
	upload.info = info
	return upload.info, nil
}

// WriteChunk writes a chunk of data to the upload
func (upload *PocketBaseUpload) WriteChunk(ctx context.Context, offset int64, src io.Reader) (int64, error) {
	uploadPath := upload.store.getUploadPath(upload.id)
	
	file, err := os.OpenFile(uploadPath, os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	
	// Seek to the offset
	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, err
	}
	
	// Write the chunk
	written, err := io.Copy(file, src)
	if err != nil {
		return written, err
	}
	
	// Update offset in info
	upload.info.Offset = offset + written
	
	// Update info file
	infoPath := upload.store.getInfoPath(upload.id)
	if err := upload.store.writeInfo(infoPath, upload.info); err != nil {
		return written, err
	}
	
	return written, nil
}

// GetReader returns a reader for the uploaded data
func (upload *PocketBaseUpload) GetReader(ctx context.Context) (io.ReadCloser, error) {
	uploadPath := upload.store.getUploadPath(upload.id)
	return os.Open(uploadPath)
}

// FinishUpload is called when the upload is complete
func (upload *PocketBaseUpload) FinishUpload(ctx context.Context) error {
	// Mark upload as completed
	upload.info.Offset = upload.info.Size
	
	infoPath := upload.store.getInfoPath(upload.id)
	return upload.store.writeInfo(infoPath, upload.info)
}

// Terminate implements handler.TerminatableUpload
func (upload *PocketBaseUpload) Terminate(ctx context.Context) error {
	uploadPath := upload.store.getUploadPath(upload.id)
	infoPath := upload.store.getInfoPath(upload.id)
	
	// Remove both upload file and info file
	os.Remove(uploadPath)
	os.Remove(infoPath)
	
	return nil
}

// DeclareLength implements handler.LengthDeclarableUpload
func (upload *PocketBaseUpload) DeclareLength(ctx context.Context, length int64) error {
	upload.info.Size = length
	
	infoPath := upload.store.getInfoPath(upload.id)
	return upload.store.writeInfo(infoPath, upload.info)
}

// ConcatUploads implements handler.ConcatableUpload
func (upload *PocketBaseUpload) ConcatUploads(ctx context.Context, partialUploads []handler.Upload) error {
	uploadPath := upload.store.getUploadPath(upload.id)
	
	file, err := os.OpenFile(uploadPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Concatenate all partial uploads
	for _, partialUpload := range partialUploads {
		reader, err := partialUpload.GetReader(ctx)
		if err != nil {
			return err
		}
		
		_, err = io.Copy(file, reader)
		reader.Close()
		if err != nil {
			return err
		}
	}
	
	// Update info
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	
	upload.info.Size = stat.Size()
	upload.info.Offset = stat.Size()
	
	infoPath := upload.store.getInfoPath(upload.id)
	return upload.store.writeInfo(infoPath, upload.info)
}
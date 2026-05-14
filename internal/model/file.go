package model

import (
	"time"

	"github.com/google/uuid"
)

type FileStatus string

const (
	FileStatusUploading  FileStatus = "UPLOADING"
	FileStatusProcessing FileStatus = "PROCESSING"
	FileStatusReady      FileStatus = "READY"
	FileStatusFailed     FileStatus = "FAILED"
	FileStatusDeleted    FileStatus = "DELETED"
)

type File struct {
	ID          uuid.UUID  `json:"id"`
	OwnerID     uuid.UUID  `json:"ownerId"`
	Filename    string     `json:"filename"`
	ObjectKey   string     `json:"objectKey"`
	Size        int64      `json:"size"`
	ContentType string     `json:"contentType"`
	Status      FileStatus `json:"status"`
	Checksum    string     `json:"checksum,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type FileChunk struct {
	ID         uuid.UUID `json:"id"`
	FileID     uuid.UUID `json:"fileId"`
	ChunkIndex int       `json:"chunkIndex"`
	ETag       string    `json:"etag,omitempty"`
	Uploaded   bool      `json:"uploaded"`
	CreatedAt  time.Time `json:"createdAt"`
}

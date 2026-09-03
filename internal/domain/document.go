package domain

import "time"

type DocumentStatus string

const (
	DocumentPending    DocumentStatus = "pending"
	DocumentProcessing DocumentStatus = "processing"
	DocumentReady      DocumentStatus = "ready"
	DocumentFailed     DocumentStatus = "failed"
)

func (s DocumentStatus) Valid() bool {
	switch s {
	case DocumentPending, DocumentProcessing, DocumentReady, DocumentFailed:
		return true
	default:
		return false
	}
}

type Document struct {
	ID              int64          `json:"id"`
	OwnerID         int64          `json:"owner_id"`
	OriginalName    string         `json:"original_name"`
	MediaType       string         `json:"media_type"`
	SizeBytes       int64          `json:"size_bytes"`
	StorageKey      string         `json:"-"`
	ChecksumSHA256  *string        `json:"checksum_sha256,omitempty"`
	Status          DocumentStatus `json:"status"`
	ProcessingError *string        `json:"processing_error,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       *time.Time     `json:"deleted_at,omitempty"`
}

type NewDocument struct {
	OwnerID      int64
	OriginalName string
	MediaType    string
	SizeBytes    int64
	StorageKey   string
}

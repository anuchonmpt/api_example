package domain

import "testing"

func TestDocumentStatusValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		status DocumentStatus
		valid  bool
	}{
		{DocumentPending, true}, {DocumentProcessing, true}, {DocumentReady, true}, {DocumentFailed, true}, {DocumentStatus("unknown"), false},
	}
	for _, tt := range tests {
		if got := tt.status.Valid(); got != tt.valid {
			t.Fatalf("DocumentStatus(%q).Valid() = %v", tt.status, got)
		}
	}
}

func TestNormalizePagination(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		page, pageSize         int
		wantPage, wantPageSize int
		wantOffset             int32
	}{
		{name: "defaults", wantPage: 1, wantPageSize: 20, wantOffset: 0},
		{name: "caps size", page: 2, pageSize: 500, wantPage: 2, wantPageSize: 100, wantOffset: 100},
		{name: "normal", page: 3, pageSize: 10, wantPage: 3, wantPageSize: 10, wantOffset: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizePagination(tt.page, tt.pageSize)
			if got.Page != tt.wantPage || got.PageSize != tt.wantPageSize || got.Offset() != tt.wantOffset {
				t.Fatalf("NormalizePagination() = %+v offset %d", got, got.Offset())
			}
		})
	}
}

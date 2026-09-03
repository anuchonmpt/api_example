package errors

import (
	"regexp"
	"testing"
)

func TestDomainErrorCodesAreUniqueAndWellFormed(t *testing.T) {
	t.Parallel()
	codes := []string{
		CodeCommonInvalidRequest, CodeCommonInvalidBody, CodeCommonInternal,
		CodeAuthInvalidRequest, CodeAuthInvalidBody, CodeAuthUnauthorized, CodeAuthInvalidCredentials, CodeAuthInvalidRefresh, CodeAuthInvalidTokenFormat, CodeAuthInvalidToken, CodeAuthUserNotFound, CodeAuthEmailConflict, CodeAuthInternal,
		CodeDocumentInvalidRequest, CodeDocumentUnauthorized, CodeDocumentForbidden, CodeDocumentNotFound, CodeDocumentNotReady, CodeDocumentConflict, CodeDocumentInternal,
		CodeStorageInvalidKey, CodeStorageObjectNotFound, CodeStorageUnavailable, CodeStorageInternal,
		CodeQueueInvalidJob, CodeQueueUnavailable, CodeQueueInternal,
	}
	pattern := regexp.MustCompile(`^E[0-9]{8}$`)
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		if !pattern.MatchString(code) {
			t.Fatalf("invalid code format %q", code)
		}
		if _, exists := seen[code]; exists {
			t.Fatalf("duplicate code %q", code)
		}
		seen[code] = struct{}{}
	}
}

func TestErrorCodeDomainAndHTTPTypeMapping(t *testing.T) {
	t.Parallel()
	tests := []struct{ code, domain, httpType string }{
		{CodeAuthInvalidCredentials, "01", "41"},
		{CodeDocumentNotFound, "02", "44"},
		{CodeStorageUnavailable, "03", "53"},
		{CodeQueueUnavailable, "04", "53"},
	}
	for _, tt := range tests {
		if tt.code[1:3] != tt.domain || tt.code[3:5] != tt.httpType {
			t.Fatalf("code %s does not map to domain=%s type=%s", tt.code, tt.domain, tt.httpType)
		}
	}
}

package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code      string
	Message   string
	Status    int
	Retryable bool
	cause     error
}

func (e *AppError) Error() string {
	if e.cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.cause)
}

func (e *AppError) Unwrap() error { return e.cause }

func (e *AppError) Is(target error) bool {
	other, ok := target.(*AppError)
	return ok && e.Code == other.Code
}

var (
	ErrCommonInvalidRequest = &AppError{Code: CodeCommonInvalidRequest, Message: "invalid request", Status: http.StatusBadRequest}
	ErrCommonInvalidBody    = &AppError{Code: CodeCommonInvalidBody, Message: "invalid request body", Status: http.StatusBadRequest}
	ErrCommonInternal       = &AppError{Code: CodeCommonInternal, Message: "internal server error", Status: http.StatusInternalServerError}

	ErrAuthInvalidRequest     = &AppError{Code: CodeAuthInvalidRequest, Message: "invalid authentication request", Status: http.StatusBadRequest}
	ErrAuthInvalidBody        = &AppError{Code: CodeAuthInvalidBody, Message: "invalid authentication request body", Status: http.StatusBadRequest}
	ErrAuthUnauthorized       = &AppError{Code: CodeAuthUnauthorized, Message: "authentication required", Status: http.StatusUnauthorized}
	ErrAuthInvalidCredentials = &AppError{Code: CodeAuthInvalidCredentials, Message: "invalid email or password", Status: http.StatusUnauthorized}
	ErrAuthInvalidRefresh     = &AppError{Code: CodeAuthInvalidRefresh, Message: "invalid refresh token", Status: http.StatusUnauthorized}
	ErrAuthInvalidTokenFormat = &AppError{Code: CodeAuthInvalidTokenFormat, Message: "invalid authorization header format", Status: http.StatusUnauthorized}
	ErrAuthInvalidToken       = &AppError{Code: CodeAuthInvalidToken, Message: "invalid or expired token", Status: http.StatusUnauthorized}
	ErrAuthUserNotFound       = &AppError{Code: CodeAuthUserNotFound, Message: "user not found", Status: http.StatusNotFound}
	ErrAuthEmailConflict      = &AppError{Code: CodeAuthEmailConflict, Message: "email already registered", Status: http.StatusConflict}
	ErrAuthInternal           = &AppError{Code: CodeAuthInternal, Message: "authentication service error", Status: http.StatusInternalServerError}

	ErrDocumentInvalidRequest = &AppError{Code: CodeDocumentInvalidRequest, Message: "invalid document request", Status: http.StatusBadRequest}
	ErrDocumentUnauthorized   = &AppError{Code: CodeDocumentUnauthorized, Message: "authentication required", Status: http.StatusUnauthorized}
	ErrDocumentForbidden      = &AppError{Code: CodeDocumentForbidden, Message: "document access forbidden", Status: http.StatusForbidden}
	ErrDocumentNotFound       = &AppError{Code: CodeDocumentNotFound, Message: "document not found", Status: http.StatusNotFound}
	ErrDocumentNotReady       = &AppError{Code: CodeDocumentNotReady, Message: "document is not ready", Status: http.StatusConflict}
	ErrDocumentConflict       = &AppError{Code: CodeDocumentConflict, Message: "document conflict", Status: http.StatusConflict}
	ErrDocumentInternal       = &AppError{Code: CodeDocumentInternal, Message: "document service error", Status: http.StatusInternalServerError}

	ErrStorageInvalidKey     = &AppError{Code: CodeStorageInvalidKey, Message: "invalid storage key", Status: http.StatusBadRequest}
	ErrStorageObjectNotFound = &AppError{Code: CodeStorageObjectNotFound, Message: "stored object not found", Status: http.StatusNotFound}
	ErrStorageUnavailable    = &AppError{Code: CodeStorageUnavailable, Message: "object storage is unavailable", Status: http.StatusServiceUnavailable, Retryable: true}
	ErrStorageInternal       = &AppError{Code: CodeStorageInternal, Message: "object storage error", Status: http.StatusInternalServerError}

	ErrQueueInvalidJob  = &AppError{Code: CodeQueueInvalidJob, Message: "invalid queue job", Status: http.StatusBadRequest}
	ErrQueueUnavailable = &AppError{Code: CodeQueueUnavailable, Message: "processing queue is unavailable", Status: http.StatusServiceUnavailable, Retryable: true}
	ErrQueueInternal    = &AppError{Code: CodeQueueInternal, Message: "processing queue error", Status: http.StatusInternalServerError}
)

func Wrap(base *AppError, cause error) error {
	if cause == nil {
		return base
	}
	copy := *base
	copy.cause = cause
	return &copy
}

func WithMessage(base *AppError, message string) error {
	copy := *base
	copy.Message = message
	return &copy
}

func As(err error) *AppError {
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return appErr
	}
	return ErrCommonInternal
}

package errors

// Error code format: E + Domain(2) + HTTP Type(2) + Sequence(4).
//
// Domains: 00=Common, 01=Auth, 02=Document, 03=Storage, 04=Queue.
// Types: 40=BadRequest, 41=Unauthorized, 42=Forbidden,
// 44=NotFound, 09=Conflict, 50=InternalServerError, 53=ServiceUnavailable.
const (
	CodeCommonInvalidRequest = "E00400001"
	CodeCommonInvalidBody    = "E00400002"
	CodeCommonInternal       = "E00500001"
)

const (
	CodeAuthInvalidRequest     = "E01400001"
	CodeAuthInvalidBody        = "E01400002"
	CodeAuthUnauthorized       = "E01410001"
	CodeAuthInvalidCredentials = "E01410002"
	CodeAuthInvalidRefresh     = "E01410003"
	CodeAuthInvalidTokenFormat = "E01410004"
	CodeAuthInvalidToken       = "E01410005"
	CodeAuthUserNotFound       = "E01440001"
	CodeAuthEmailConflict      = "E01090001"
	CodeAuthInternal           = "E01500001"
)

const (
	CodeDocumentInvalidRequest = "E02400001"
	CodeDocumentUnauthorized   = "E02410001"
	CodeDocumentForbidden      = "E02420001"
	CodeDocumentNotFound       = "E02440001"
	CodeDocumentNotReady       = "E02090001"
	CodeDocumentConflict       = "E02090002"
	CodeDocumentInternal       = "E02500001"
)

const (
	CodeStorageInvalidKey     = "E03400001"
	CodeStorageObjectNotFound = "E03440001"
	CodeStorageUnavailable    = "E03530001"
	CodeStorageInternal       = "E03500001"
)

const (
	CodeQueueInvalidJob  = "E04400001"
	CodeQueueUnavailable = "E04530001"
	CodeQueueInternal    = "E04500001"
)

// Package apperror defines language-independent errors safe to expose to clients.
package apperror

import "errors"

type Error struct {
	code    string
	message string
}

func New(code, message string) *Error { return &Error{code: code, message: message} }

func (e *Error) Error() string { return e.message }

// Public unwraps expected errors; unexpected failures never expose internal details.
func Public(err error) *Error {
	var public *Error
	if errors.As(err, &public) && public != nil {
		return public
	}
	return Internal
}

func Code(err error) string { return Public(err).code }

var (
	InvalidBody        = New("HTTP_INVALID_BODY", "invalid request body")
	InvalidQuery       = New("HTTP_INVALID_QUERY", "invalid query parameters")
	Page               = New("HTTP_INVALID_PAGE", "p must be one int32 value")
	PageSize           = New("HTTP_INVALID_PAGE_SIZE", "s must be one int32 value")
	IdempotencyKey     = New("HTTP_INVALID_IDEMPOTENCY_KEY", "invalid idempotency key")
	DuplicateRequest   = New("HTTP_DUPLICATE_REQUEST", "idempotency key has already been used")
	Internal           = New("HTTP_INTERNAL", "internal server error")
	BadRequest         = New("HTTP_BAD_REQUEST", "Bad Request")
	Unauthorized       = New("HTTP_UNAUTHORIZED", "Unauthorized")
	Forbidden          = New("HTTP_FORBIDDEN", "Forbidden")
	NotFound           = New("HTTP_NOT_FOUND", "Not Found")
	MethodNotAllowed   = New("HTTP_METHOD_NOT_ALLOWED", "Method Not Allowed")
	GatewayTimeout     = New("HTTP_GATEWAY_TIMEOUT", "Gateway Timeout")
	ServiceUnavailable = New("HTTP_SERVICE_UNAVAILABLE", "Service Unavailable")
)

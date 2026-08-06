package auth

import "hotspot/backend-go/internal/platform/httpx"

func Unauthorized(message string) *httpx.APIError {
	return &httpx.APIError{Code: httpx.CodeUnauthorized, Message: message}
}

func Forbidden(message string) *httpx.APIError {
	return &httpx.APIError{Code: httpx.CodeForbidden, Message: message}
}

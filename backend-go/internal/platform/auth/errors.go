package auth

import "hotspot/backend-go/internal/platform/httpx"

func Unauthorized(message string) *httpx.APIError {
	return &httpx.APIError{Code: 10100, Message: message}
}

func Forbidden(message string) *httpx.APIError {
	return &httpx.APIError{Code: 10200, Message: message}
}

package httpx

import "fmt"

type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error %d: %s", e.Code, e.Message)
}

func InvalidRequest(message string) *APIError {
	return &APIError{Code: 10000, Message: message}
}

func InternalUnavailable(message string) *APIError {
	return &APIError{Code: 10600, Message: message}
}

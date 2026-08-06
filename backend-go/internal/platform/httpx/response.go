package httpx

import (
	"encoding/json"
	"net/http"
)

type ApiResponse[T any] struct {
	Code    Code   `json:"code"`
	Message string `json:"message,omitempty"`
	Data    *T     `json:"data,omitempty"`
}

func Success[T any](data T) ApiResponse[T] {
	return ApiResponse[T]{Code: CodeOK, Data: &data}
}

func SuccessEmpty() ApiResponse[struct{}] {
	empty := struct{}{}
	return ApiResponse[struct{}]{Code: CodeOK, Data: &empty}
}

func Error(code Code, message string) ApiResponse[struct{}] {
	return ApiResponse[struct{}]{Code: code, Message: message}
}

func WriteJSON[T any](w http.ResponseWriter, status int, payload ApiResponse[T]) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

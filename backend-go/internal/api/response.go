package api

import (
	"encoding/json"
	"net/http"
)

// Response 为所有接口的统一信封结构，对应 spec 中的 ApiResponse<T>。
type Response[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *T     `json:"data,omitempty"`
}

func WriteSuccess[T any](w http.ResponseWriter, r *http.Request, data T) {
	resp := Response[T]{
		Code:    0,
		Message: "ok",
		Data:    &data,
	}
	writeJSON(w, resp)
}

// WriteError 由上层根据统一的错误类型转换为 code/message 再写回。
func WriteError(w http.ResponseWriter, r *http.Request, code int, message string) {
	resp := Response[struct{}]{
		Code:    code,
		Message: message,
	}
	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

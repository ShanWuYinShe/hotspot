package httpx

import (
	"errors"
	"net/http"

	"hotspot/backend-go/internal/platform/observability"
)

// WriteInvalidRequest writes a CodeInvalidRequest response with the matching
// business-error log line. It is the single exit for the "参数错误" handler
// boilerplate (log + JSON pair) across all modules.
func WriteInvalidRequest(w http.ResponseWriter, r *http.Request, op, message string) {
	WriteBusinessError(w, r, op, CodeInvalidRequest, message)
}

// WriteBusinessError writes an arbitrary business-error response (business
// log + JSON), the unified exit for handler business-error branches.
func WriteBusinessError(w http.ResponseWriter, r *http.Request, op string, code Code, message string) {
	observability.LogBusinessError(r, op, int(code), message)
	WriteJSON(w, http.StatusOK, Error(code, message))
}

// WriteAPIError is the unified error-response exit for handlers: an
// *APIError is written as a business error (business log + its code/message);
// any other error is written as a server error (error log + 500 semantics).
func WriteAPIError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		observability.LogBusinessError(r, op, int(apiErr.Code), apiErr.Message)
		if apiErr.Data != nil {
			WriteJSON(w, http.StatusOK, ErrorWithData(apiErr.Code, apiErr.Message, apiErr.Data))
			return
		}
		WriteJSON(w, http.StatusOK, Error(apiErr.Code, apiErr.Message))
		return
	}
	observability.LogServerError(r, op, err)
	WriteJSON(w, http.StatusOK, Error(CodeInternalUnavailable, "服务暂时不可用"))
}

// Business 统一「业务错误日志 + ApiResponse」出口：handler 在 errors.Is 分支
// 里调用它替代手写的 LogBusinessError + ApiResponse{Code, Message} 成对样板，
// 保证日志与响应的 code/message 永远一致。T 显式传入（Go 不做返回类型推断）。
func Business[T any](r *http.Request, method string, code Code, message string) ApiResponse[T] {
	observability.LogBusinessError(r, method, int(code), message)
	return ApiResponse[T]{Code: code, Message: message}
}

// BusinessForbidden 10200 + data.forbidden_reason 的回调版出口（契约
// api_spec_errors.md 要求 10200 必带 forbidden_reason）。与 Business 一样
// 统一「日志 + 响应」一致出口，供 idempotency.ReplayOrStore 等回调使用。
func BusinessForbidden(r *http.Request, method string, reason ForbiddenReason, message string) ApiResponse[any] {
	observability.LogBusinessError(r, method, int(CodeForbidden), message)
	data := any(ForbiddenErrorData{ForbiddenReason: reason})
	return ApiResponse[any]{Code: CodeForbidden, Message: message, Data: &data}
}

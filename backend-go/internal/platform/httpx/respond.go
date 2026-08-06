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
		WriteJSON(w, http.StatusOK, Error(apiErr.Code, apiErr.Message))
		return
	}
	observability.LogServerError(r, op, err)
	WriteJSON(w, http.StatusOK, Error(CodeInternalUnavailable, "服务暂时不可用"))
}

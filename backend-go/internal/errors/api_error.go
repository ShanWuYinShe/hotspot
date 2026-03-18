package errors

// APIError 表示对外的统一业务错误。
// code 必须来自 spec/api_spec_errors.md，message 为用户可见文案。
type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}

func New(code int, message string) *APIError {
	return &APIError{Code: code, Message: message}
}

// 常用错误构造函数（错误码需与 spec 对齐）。

// ErrInvalidRequest 表示通用的请求参数非法，对应 10000。
func ErrInvalidRequest(message string) *APIError {
	if message == "" {
		message = "invalid_request"
	}
	return &APIError{Code: 10000, Message: message}
}

// ErrInternal 表示内部错误，对应 10600。
func ErrInternal(message string) *APIError {
	if message == "" {
		message = "internal_error"
	}
	return &APIError{Code: 10600, Message: message}
}

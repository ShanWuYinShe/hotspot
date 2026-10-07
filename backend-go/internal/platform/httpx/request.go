package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// bodyTooLargeMessage 与 spec/api_spec_errors.md 的 10000 示例文案对齐：
// 超出体积上限属于「过大」，不能笼统归到「格式错误」，否则客户端提示误导。
const bodyTooLargeMessage = "请求体过大"

// isBodyTooLarge 判断解码失败是否源自 http.MaxBytesReader（LimitRequestBody 中间件）。
func isBodyTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}

func DecodeStrictJSON(r *http.Request, dest any) *APIError {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		if isBodyTooLarge(err) {
			return InvalidRequest(bodyTooLargeMessage)
		}
		return InvalidRequest("请求体格式错误")
	}
	var extra struct{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if isBodyTooLarge(err) {
			return InvalidRequest(bodyTooLargeMessage)
		}
		return InvalidRequest("请求体格式错误")
	}
	return nil
}

// RequireEmptyObject validates that the request body is a JSON object (per
// api_spec_base.md 空请求体约定). It deliberately does NOT reject unknown
// fields: the spec mandates empty-object endpoints silently ignore unknown
// keys so SDK telemetry/gateway-injected fields don't hard-fail. It only
// rejects non-object bodies (arrays, strings, etc.) and malformed JSON.
func RequireEmptyObject(r *http.Request) *APIError {
	defer r.Body.Close()
	var payload map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		if isBodyTooLarge(err) {
			return InvalidRequest(bodyTooLargeMessage)
		}
		return InvalidRequest("请求体必须是 JSON 对象")
	}
	return nil
}

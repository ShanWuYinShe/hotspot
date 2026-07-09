package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func DecodeStrictJSON(r *http.Request, dest any) *APIError {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return InvalidRequest("请求体格式错误")
	}
	var extra struct{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
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
		return InvalidRequest("请求体必须是 JSON 对象")
	}
	return nil
}

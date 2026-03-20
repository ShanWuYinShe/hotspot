package httpx

import (
	"encoding/json"
	"net/http"
)

func RequireEmptyObject(r *http.Request) *APIError {
	defer r.Body.Close()
	var payload map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return InvalidRequest("请求体必须是 JSON 空对象 {}")
	}
	if len(payload) != 0 {
		return InvalidRequest("请求体必须是 JSON 空对象 {}")
	}
	return nil
}

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

func RequireEmptyObject(r *http.Request) *APIError {
	var payload map[string]json.RawMessage
	if err := DecodeStrictJSON(r, &payload); err != nil {
		return InvalidRequest("请求体必须是 JSON 空对象 {}")
	}
	if len(payload) != 0 {
		return InvalidRequest("请求体必须是 JSON 空对象 {}")
	}
	return nil
}

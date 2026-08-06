package auth

import (
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

func TestAPIErrorMappingCoversAllBranches(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
	}{
		{"nil", nil, 0},
		{"conflict", ErrLoginIDConflict, 10400},
		{"invalid credentials", ErrInvalidCredentials, 10100},
		{"account locked", ErrAccountLocked, 10500},
		{"invalid old password", ErrInvalidOldPassword, 10100},
		{"reset not found", ErrPasswordResetNotFound, 10300},
		{"reset throttled", ErrPasswordResetThrottled, 10500},
		{"email unavailable", ErrEmailServiceUnavailable, 10500},
		{"unknown", errors.New("boom"), 10600},
	}
	for _, tt := range tests {
		apiErr := apiErrorFrom(tt.err)
		if tt.code == 0 {
			if apiErr != nil {
				t.Fatalf("%s: expected nil, got %#v", tt.name, apiErr)
			}
			continue
		}
		if apiErr == nil || apiErr.Code != tt.code {
			t.Fatalf("%s: expected code %d, got %#v", tt.name, tt.code, apiErr)
		}
	}
	// APIError 直接透传
	apiErr := apiErrorFrom(&httpx.APIError{Code: 10300, Message: "x"})
	if apiErr == nil || apiErr.Code != 10300 {
		t.Fatalf("expected passthrough, got %#v", apiErr)
	}
	// 包装的 APIError 也能解包
	wrapped := errors.New("inner")
	wrapped = &httpx.APIError{Code: 10400, Message: "inner"}
	apiErr = apiErrorFrom(wrapped)
	if apiErr == nil || apiErr.Code != 10400 {
		t.Fatalf("expected wrapped passthrough, got %#v", apiErr)
	}
}

package auth

import (
	"errors"
	"fmt"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

func TestAPIErrorMappingCoversAllBranches(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code httpx.Code
	}{
		{"nil", nil, 0},
		{"conflict", ErrLoginIDConflict, httpx.CodeConflict},
		{"invalid credentials", ErrInvalidCredentials, httpx.CodeUnauthorized},
		{"account locked", ErrAccountLocked, httpx.CodeRateLimited},
		{"invalid old password", ErrInvalidOldPassword, httpx.CodeInvalidOldCredential},
		{"reset not found", ErrPasswordResetNotFound, httpx.CodeNotFound},
		{"reset throttled", ErrPasswordResetThrottled, httpx.CodeRateLimited},
		{"verification unavailable", ErrVerificationServiceUnavailable, httpx.CodeInternalUnavailable},
		{"unknown", errors.New("boom"), httpx.CodeInternalUnavailable},
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
	apiErr := apiErrorFrom(&httpx.APIError{Code: httpx.CodeNotFound, Message: "x"})
	if apiErr == nil || apiErr.Code != httpx.CodeNotFound {
		t.Fatalf("expected passthrough, got %#v", apiErr)
	}
	// 包装的 APIError 也能解包（fmt.Errorf %w 后 errors.As 仍命中同一 APIError）
	wrapped := fmt.Errorf("wrap: %w", &httpx.APIError{Code: httpx.CodeConflict, Message: "inner"})
	apiErr = apiErrorFrom(wrapped)
	if apiErr == nil || apiErr.Code != httpx.CodeConflict {
		t.Fatalf("expected wrapped passthrough, got %#v", apiErr)
	}
}

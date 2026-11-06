package auth

import "time"

// 仓储层内部行类型与节流常量：PG 与内存实现共用（内存实现位于测试构建）。

type accountRecord struct {
	Account
	PasswordHash      string
	RegisterIP        string
	RegisterLocation  string
	LastLoginIP       string
	LastLoginLocation string
}

type passwordResetRecord struct {
	RequestID        string
	LoginID          string
	VerificationCode string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	UsedAt           *time.Time
	FailedAttempts   int
}

type registerCodeRecord struct {
	RequestID        string
	LoginID          string
	VerificationCode string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	ConsumedAt       *time.Time
	FailedAttempts   int
}

const passwordResetThrottleWindow = 10 * time.Minute

const registerCodeThrottleWindow = 60 * time.Second

const maxVerificationCodeFailures = 5

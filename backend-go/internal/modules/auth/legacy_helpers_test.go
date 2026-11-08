package auth

import (
	"time"
)

// 以下辅助从生产代码下沉到测试：生产路径已不使用，仅测试替身实现需要。

type registerCodeRecord struct {
	RequestID        string
	LoginID          string
	VerificationCode string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	ConsumedAt       *time.Time
	FailedAttempts   int
}

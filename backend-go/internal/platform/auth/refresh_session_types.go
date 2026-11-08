package auth

import "time"

type RefreshSession struct {
	UserID       string
	SessionID    string
	RefreshToken string
	RotatedTo    string
	GraceUntil   time.Time
	ExpiresAt    time.Time
	// GraceTokenEnc 宽限期托管：轮换后新链 token 的 AES-GCM 密文（密钥
	// 不在 DB），供并发刷新败者在宽限期内换回明文。仅被轮换的旧行携带。
	GraceTokenEnc []byte
}

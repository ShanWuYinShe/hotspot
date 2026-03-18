package domain

import "time"

// User 代表持久化层的用户实体（不直接暴露给 API）。
type User struct {
	UserID    string
	LoginID   *string
	Nickname  string
	Password  *string // 哈希后的密码（nil 表示访客或无密码账号）
	UserType  string  // 对应 spec 中的 UserType（temp/normal）
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Session 表示一个登录会话。
type Session struct {
	SessionID    string
	UserID       string
	DeviceType   *string
	DeviceName   *string
	IPAddress    *string
	IPLocation   *string
	CreatedAt    time.Time
	LastActiveAt time.Time
	RevokedAt    *time.Time
}

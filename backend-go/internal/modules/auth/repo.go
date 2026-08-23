package auth

import (
	"context"
	"time"
)

type ProfileDefaults struct {
	AvatarURL    string
	LocationName string
	Latitude     float64
	Longitude    float64
	CreatedAt    time.Time
}

type Account struct {
	UserID   string
	LoginID  string
	Nickname string
	UserType string
}

type RegisterParams struct {
	LoginID          string
	Nickname         string
	Password         string
	Defaults         ProfileDefaults
	RegisterIP       string
	RegisterLocation string
}

type GuestParams struct {
	Nickname         string
	Defaults         ProfileDefaults
	RegisterIP       string
	RegisterLocation string
}

type Repository interface {
	Register(context.Context, RegisterParams) (Account, error)
	CreateGuest(context.Context, GuestParams) (Account, error)
	Authenticate(context.Context, string, string) (Account, error)
	UpdateLoginMeta(context.Context, string, string, string) error
	CreatePasswordReset(context.Context, string, time.Time) (string, error)
	ResetPassword(context.Context, string, string, string) (Account, error)
	ChangePassword(context.Context, string, string, string) error
	// CreateRegisterCode 生成并落库一条注册验证码（spec api_spec_public.md →
	// auth/send_register_code）。无论 login_id 是否已被注册均落库（防枚举）。
	CreateRegisterCode(context.Context, string, time.Time) (string, error)
	// VerifyRegisterCode 校验注册验证码有效（存在、未消费、未过期），不消费。
	VerifyRegisterCode(context.Context, string, string) error
	// ConsumeRegisterCode 将验证码标记为已消费（一次性，成功注册后调用）。
	ConsumeRegisterCode(context.Context, string, string) error
}

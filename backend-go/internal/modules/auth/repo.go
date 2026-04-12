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
	CreatePasswordReset(context.Context, string, time.Time) error
	ResetPassword(context.Context, string, string, string) (Account, error)
	ChangePassword(context.Context, string, string, string) error
}

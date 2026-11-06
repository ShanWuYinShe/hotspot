package users

import (
	"context"
	"errors"
	"time"
)

// ProvisionUserInput 是注册/游客建用户所需的资料字段（users 表归属本模块；
// auth 模块注册流程经 ProfileWriter 写入，不再直写 users/user_settings，
// 架构审计 P0-4）。
type ProvisionUserInput struct {
	UserID           string
	Nickname         string
	AvatarURL        string
	UserType         string
	Email            string
	Phone            string
	LocationName     string
	Latitude         float64
	Longitude        float64
	CreatedAt        time.Time
	RegisterIP       string
	RegisterLocation string
}

// ProfileWriter 是 auth 模块消费的最小写能力（注册建用户/游客建用户/登录元数据）。
type ProfileWriter interface {
	ProvisionUser(ctx context.Context, input ProvisionUserInput) error
	RecordLogin(ctx context.Context, userID, ip, location string) error
}

// profileWriterStore 是 PG 仓储的可选实现能力（内存实现不需要）。
type profileWriterStore interface {
	ProvisionUser(ctx context.Context, input ProvisionUserInput) error
	RecordLogin(ctx context.Context, userID, ip, location string) error
}

var errProfileWriterUnsupported = errors.New("users: profile writer unsupported")

// ProvisionUser 创建注册用户资料与默认设置（在调用方事务上下文中执行）。
func (s *Service) ProvisionUser(ctx context.Context, input ProvisionUserInput) error {
	store, ok := s.repo.(profileWriterStore)
	if !ok {
		return errProfileWriterUnsupported
	}
	return store.ProvisionUser(ctx, input)
}

// RecordLogin 更新最近登录 IP/位置（users 表归属本模块）。
func (s *Service) RecordLogin(ctx context.Context, userID, ip, location string) error {
	store, ok := s.repo.(profileWriterStore)
	if !ok {
		return errProfileWriterUnsupported
	}
	return store.RecordLogin(ctx, userID, ip, location)
}

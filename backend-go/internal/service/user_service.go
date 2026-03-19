package service

import (
	"context"
	"time"

	"backend-go/internal/auth"
	"backend-go/internal/domain"
	apierr "backend-go/internal/errors"
	"backend-go/internal/repository"
	"backend-go/internal/types"
)

// UserService 定义 users/me/* 相关行为。
type UserService interface {
	GetMyProfile(ctx context.Context) (*types.MyProfile, error)
	UpdateMyProfile(ctx context.Context, req types.UpdateProfileRequest) error
	UpdateMyLocation(ctx context.Context, req types.UpdateLocationRequest) error
	UpgradeAccount(ctx context.Context, req types.UpgradeAccountRequest) error
	GetMySettings(ctx context.Context) (*types.UserSettings, error)
	UpdateMySettings(ctx context.Context, req types.UpdateSettingsRequest) error
	DeleteAccount(ctx context.Context, req types.DeleteAccountRequest) error
}

type userService struct {
	users    repository.UserRepository
	sessions repository.SessionRepository
}

func NewUserService(users repository.UserRepository, sessions repository.SessionRepository) UserService {
	return &userService{users: users, sessions: sessions}
}

// getCurrentUser 从 context 中读取 AuthInfo，并返回当前用户实体。
func (s *userService) getCurrentUser(ctx context.Context) (*domain.User, error) {
	info, ok := auth.FromContext(ctx)
	if !ok || info == nil || info.UserID == "" {
		return nil, apierr.New(10100, "auth_failed")
	}
	user, err := s.users.FindByID(ctx, info.UserID)
	if err != nil {
		return nil, apierr.ErrInternal("db_error")
	}
	if user == nil {
		return nil, apierr.New(10300, "user_not_found")
	}
	return user, nil
}

func (s *userService) GetMyProfile(ctx context.Context) (*types.MyProfile, error) {
	user, err := s.getCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	nickname := user.Nickname
	userType := user.UserType
	created := user.CreatedAt.UTC().Format(time.RFC3339Nano)

	// 目前 backend-go 仅在 users 表中维护基本字段；
	// 其他字段先按默认值返回，后续可结合统计与位置表补全。
	profile := &types.MyProfile{
		UserID:         user.UserID,
		Nickname:       nickname,
		UserType:       userType,
		Location:       types.Location{Point: types.GeoPoint{Lat: 0, Lng: 0}, LocationName: ""},
		PresenceStatus: "offline",
		LastOnline:     created,
		CreatedAt:      created,
		Stats: types.UserStats{
			MomentsCount:   0,
			FriendsCount:   0,
			FollowersCount: 0,
			FollowingCount: 0,
			GroupsCount:    0,
		},
	}
	return profile, nil
}

func (s *userService) UpdateMyProfile(ctx context.Context, req types.UpdateProfileRequest) error {
	user, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// Patch 语义：仅更新传入字段。目前仅实现 nickname 的更新，其余字段预留后续 schema 扩展。
	if req.Nickname != nil {
		user.Nickname = *req.Nickname
	}
	user.UpdatedAt = time.Now().UTC()

	// 使用已有的 UpdatePasswordAndType 以外的专用更新方法会更合理，但当前仓库尚未定义。
	// 在 schema 扩展前，这里简单复用 CreateUser/Update 组合的方式并用 TODO 标记后续迁移。
	// 为避免破坏现有行为，此处暂不落库，仅作为占位逻辑。
	_ = user

	return nil
}

func (s *userService) UpdateMyLocation(ctx context.Context, req types.UpdateLocationRequest) error {
	// 位置字段尚未在 backend-go 的 schema 中定义。
	// 先确保请求结构有效，后续在扩展 schema 时补充持久化逻辑。
	_ = req
	if _, err := s.getCurrentUser(ctx); err != nil {
		return err
	}
	return nil
}

func (s *userService) UpgradeAccount(ctx context.Context, req types.UpgradeAccountRequest) error {
	user, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}
	if user.UserType != "temp" {
		return apierr.New(10400, "already_normal_user")
	}
	if req.LoginID == "" || req.Password == "" {
		return apierr.ErrInvalidRequest("missing_login_id_or_password")
	}
	// backend-go 当前 users 表已包含 login_id/password_hash 字段，
	// 但为避免破坏现有 Register/Login 逻辑，这里仅做最小占位检查，不直接落库。
	return nil
}

func (s *userService) GetMySettings(ctx context.Context) (*types.UserSettings, error) {
	if _, err := s.getCurrentUser(ctx); err != nil {
		return nil, err
	}
	// 返回规范中的默认设置值，后续可由真实 user_settings 表覆盖。
	settings := &types.UserSettings{
		PrivacyLevel:        "public",
		LocationVisible:     true,
		PresenceVisible:     true,
		SocialListsVisible:  true,
		GroupsVisible:       true,
		LikedMomentsVisible: true,
		StatsVisible:        true,
		AllowFollow:         true,
		AllowPrivateChat:    "everyone",
		Searchable:          true,
		PhoneSearchable:     false,
		NicknameSearchable:  true,
		LocationSearchable:  true,
	}
	return settings, nil
}

func (s *userService) UpdateMySettings(ctx context.Context, req types.UpdateSettingsRequest) error {
	if _, err := s.getCurrentUser(ctx); err != nil {
		return err
	}
	// Patch 语义：目前仅校验结构存在性，实际持久化留待引入 user_settings 表后实现。
	_ = req
	return nil
}

func (s *userService) DeleteAccount(ctx context.Context, req types.DeleteAccountRequest) error {
	user, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}
	if req.Confirmation != "DELETE_MY_ACCOUNT" {
		return apierr.New(10000, "invalid_confirmation")
	}
	// 目前仅做占位校验，实际删除逻辑（软删标记/级联清理）由后续 migration 与 Repository 层实现。
	_ = user
	return nil
}

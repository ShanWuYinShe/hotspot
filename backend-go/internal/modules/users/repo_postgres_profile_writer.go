package users

import (
	"context"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
)

// ProvisionUser 创建注册用户资料与默认设置（与 auth 凭证写入共用调用方
// 事务上下文；users/user_settings 归属本模块，架构审计 P0-4）。
func (r *PostgresRepository) ProvisionUser(ctx context.Context, input ProvisionUserInput) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	s2CellID := geo.LeafCellID(geo.Point{Latitude: input.Latitude, Longitude: input.Longitude})
	now := input.CreatedAt.UTC()
	if err := queryable.Exec(ctx, queryProvisionUserProfile,
		input.UserID, input.Nickname, input.AvatarURL, input.UserType,
		nullIfEmpty(input.Email), nullIfEmpty(input.Phone),
		input.LocationName, input.Latitude, input.Longitude, s2CellID,
		now, now, input.RegisterIP, input.RegisterLocation, input.RegisterIP, input.RegisterLocation); err != nil {
		return err
	}
	return queryable.Exec(ctx, queryProvisionUserSettings, input.UserID)
}

// RecordLogin 更新最近登录 IP/位置（auth 登录成功后调用）。
func (r *PostgresRepository) RecordLogin(ctx context.Context, userID, ip, location string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryRecordLoginMeta, userID, ip, location)
}

// nullIfEmpty 把空字符串转 NULL（email/phone 未提供时保持 NULL，与历史注册口径一致）。
func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

package admin

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	memory := NewInMemoryRepository()
	if factory == nil || factory.Config.PostgresDSN == "" {
		return &PersistenceRepository{impl: memory}
	}
	return &PersistenceRepository{impl: NewPostgresRepository(factory, memory)}
}

func (r *PersistenceRepository) GetActor(ctx context.Context, userID string) (AdminActor, bool, error) {
	return r.impl.GetActor(ctx, userID)
}

func (r *PersistenceRepository) ListPermissions(ctx context.Context) ([]AdminPermissionItem, error) {
	return r.impl.ListPermissions(ctx)
}

func (r *PersistenceRepository) ListRoles(ctx context.Context) ([]AdminRoleItem, error) {
	return r.impl.ListRoles(ctx)
}

func (r *PersistenceRepository) CreateRole(ctx context.Context, req AdminRbacRoleCreateRequest) (AdminRoleItem, error) {
	return r.impl.CreateRole(ctx, req)
}

func (r *PersistenceRepository) UpdateRole(ctx context.Context, req AdminRbacRoleUpdateRequest) error {
	return r.impl.UpdateRole(ctx, req)
}

func (r *PersistenceRepository) DeleteRole(ctx context.Context, roleID string) error {
	return r.impl.DeleteRole(ctx, roleID)
}

func (r *PersistenceRepository) SetRolePermissions(ctx context.Context, roleID string, permissionKeys []string) error {
	return r.impl.SetRolePermissions(ctx, roleID, permissionKeys)
}

func (r *PersistenceRepository) ListAdmins(ctx context.Context, pagination PaginationParams) (adminListPage, error) {
	return r.impl.ListAdmins(ctx, pagination)
}

func (r *PersistenceRepository) GrantAdmin(ctx context.Context, grant AdminGrantUpsert) error {
	return r.impl.GrantAdmin(ctx, grant)
}

func (r *PersistenceRepository) RevokeAdmin(ctx context.Context, userID string) error {
	return r.impl.RevokeAdmin(ctx, userID)
}

func (r *PersistenceRepository) ListAuditLogs(ctx context.Context, filter AuditLogFilter, includeIPAddress bool) (auditLogPage, error) {
	return r.impl.ListAuditLogs(ctx, filter, includeIPAddress)
}

func (r *PersistenceRepository) GetAuditLog(ctx context.Context, logID string, includeIPAddress bool) (AuditLogItem, bool, error) {
	return r.impl.GetAuditLog(ctx, logID, includeIPAddress)
}

func (r *PersistenceRepository) SearchUsers(ctx context.Context, req AdminUserSearchRequest) (adminUserPage, error) {
	return r.impl.SearchUsers(ctx, req)
}

func (r *PersistenceRepository) GetUser(ctx context.Context, userID string) (AdminUserItem, bool, error) {
	return r.impl.GetUser(ctx, userID)
}

func (r *PersistenceRepository) GetUserBanState(ctx context.Context, userID string) (UserBanState, bool, error) {
	return r.impl.GetUserBanState(ctx, userID)
}

func (r *PersistenceRepository) SetUserBanState(ctx context.Context, userID string, state UserBanState) error {
	return r.impl.SetUserBanState(ctx, userID, state)
}

func (r *PersistenceRepository) ListUserSessions(ctx context.Context, userID string) ([]string, error) {
	return r.impl.ListUserSessions(ctx, userID)
}

func (r *PersistenceRepository) RevokeSession(ctx context.Context, sessionID string) error {
	return r.impl.RevokeSession(ctx, sessionID)
}

func (r *PersistenceRepository) SearchGroups(ctx context.Context, req AdminGroupSearchRequest) (adminGroupPage, error) {
	return r.impl.SearchGroups(ctx, req)
}

func (r *PersistenceRepository) GetGroup(ctx context.Context, groupID string) (AdminGroupItem, bool, error) {
	return r.impl.GetGroup(ctx, groupID)
}

func (r *PersistenceRepository) DismissGroup(ctx context.Context, groupID string) (AdminGroupItem, bool, error) {
	return r.impl.DismissGroup(ctx, groupID)
}

func (r *PersistenceRepository) ListGroupMemberIDs(ctx context.Context, groupID string) ([]string, error) {
	return r.impl.ListGroupMemberIDs(ctx, groupID)
}

func (r *PersistenceRepository) SearchReports(ctx context.Context, req AdminReportSearchRequest) (adminReportPage, error) {
	return r.impl.SearchReports(ctx, req)
}

func (r *PersistenceRepository) GetReportHandleRecord(ctx context.Context, reportID string) (reportHandleRecord, bool, error) {
	return r.impl.GetReportHandleRecord(ctx, reportID)
}

func (r *PersistenceRepository) HandleReport(ctx context.Context, reportID string, action ReportHandleAction, note string, handledBy *UserInfo) (AdminReportItem, bool, error) {
	return r.impl.HandleReport(ctx, reportID, action, note, handledBy)
}

func (r *PersistenceRepository) RecallGroupMessage(ctx context.Context, messageID string) error {
	return r.impl.RecallGroupMessage(ctx, messageID)
}

func (r *PersistenceRepository) RecallPrivateMessage(ctx context.Context, messageID string) error {
	return r.impl.RecallPrivateMessage(ctx, messageID)
}

func (r *PersistenceRepository) DeleteMoment(ctx context.Context, momentID string) error {
	return r.impl.DeleteMoment(ctx, momentID)
}

func (r *PersistenceRepository) DeleteComment(ctx context.Context, commentID string) error {
	return r.impl.DeleteComment(ctx, commentID)
}

func (r *PersistenceRepository) AppendAuditLog(ctx context.Context, entry AuditLogEntry) error {
	return r.impl.AppendAuditLog(ctx, entry)
}

func (r *PersistenceRepository) IsBanned(ctx context.Context, userID string) (bool, error) {
	type banChecker interface {
		IsBanned(context.Context, string) (bool, error)
	}
	checker, ok := r.impl.(banChecker)
	if !ok {
		return false, nil
	}
	return checker.IsBanned(ctx, userID)
}

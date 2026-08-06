package admin

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type adminRowValues struct {
	values []any
	err    error
}

func (r adminRowValues) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *UserAccountStatus:
			*d = UserAccountStatus(r.values[i].(string))
		case *bool:
			*d = r.values[i].(bool)
		case *int:
			*d = r.values[i].(int)
		case *float64:
			*d = r.values[i].(float64)
		case *time.Time:
			*d = r.values[i].(time.Time)
		case *[]string:
			*d = append([]string(nil), r.values[i].([]string)...)
		case **string:
			if r.values[i] == nil {
				*d = nil
			} else if v, ok := r.values[i].(string); ok {
				*d = &v
			}
		case **time.Time:
			if r.values[i] == nil {
				*d = nil
			} else if v, ok := r.values[i].(time.Time); ok {
				*d = &v
			}
		}
	}
	return nil
}

type adminRowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *adminRowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *adminRowsValues) Scan(dest ...any) error {
	row := adminRowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *adminRowsValues) Close() error { return nil }
func (r *adminRowsValues) Err() error   { return r.err }

type adminFallbackRepo struct{ used bool }

func (r *adminFallbackRepo) GetActor(context.Context, string) (AdminActor, bool, error) {
	r.used = true
	return AdminActor{}, false, nil
}
func (r *adminFallbackRepo) ListPermissions(context.Context) ([]AdminPermissionItem, error) {
	r.used = true
	return nil, nil
}
func (r *adminFallbackRepo) ListRoles(context.Context) ([]AdminRoleItem, error) {
	r.used = true
	return nil, nil
}
func (r *adminFallbackRepo) CreateRole(context.Context, AdminRbacRoleCreateRequest) (AdminRoleItem, error) {
	r.used = true
	return AdminRoleItem{}, nil
}
func (r *adminFallbackRepo) UpdateRole(context.Context, AdminRbacRoleUpdateRequest) error {
	r.used = true
	return nil
}
func (r *adminFallbackRepo) DeleteRole(context.Context, string) error {
	r.used = true
	return nil
}
func (r *adminFallbackRepo) SetRolePermissions(context.Context, string, []string) error {
	r.used = true
	return nil
}
func (r *adminFallbackRepo) ListAdmins(context.Context, PaginationParams) (adminListPage, error) {
	r.used = true
	return adminListPage{}, nil
}
func (r *adminFallbackRepo) GrantAdmin(context.Context, AdminGrantUpsert) error {
	r.used = true
	return nil
}
func (r *adminFallbackRepo) RevokeAdmin(context.Context, string) error {
	r.used = true
	return nil
}
func (r *adminFallbackRepo) ListAuditLogs(context.Context, AuditLogFilter, bool) (auditLogPage, error) {
	r.used = true
	return auditLogPage{}, nil
}
func (r *adminFallbackRepo) GetAuditLog(context.Context, string, bool) (AuditLogItem, bool, error) {
	r.used = true
	return AuditLogItem{}, false, nil
}
func (r *adminFallbackRepo) AppendAuditLog(context.Context, AuditLogEntry) error {
	r.used = true
	return nil
}
func (r *adminFallbackRepo) SearchUsers(context.Context, AdminUserSearchRequest) (adminUserPage, error) {
	r.used = true
	return adminUserPage{}, nil
}
func (r *adminFallbackRepo) GetUser(context.Context, string) (AdminUserItem, bool, error) {
	r.used = true
	return AdminUserItem{}, false, nil
}
func (r *adminFallbackRepo) GetUserBanState(context.Context, string) (UserBanState, bool, error) {
	r.used = true
	return UserBanState{}, false, nil
}
func (r *adminFallbackRepo) SetUserBanState(context.Context, string, UserBanState) error {
	r.used = true
	return nil
}
func (r *adminFallbackRepo) ListUserSessions(context.Context, string) ([]string, error) {
	r.used = true
	return nil, nil
}
func (r *adminFallbackRepo) RevokeSession(context.Context, string) error {
	r.used = true
	return nil
}
func (r *adminFallbackRepo) SearchGroups(context.Context, AdminGroupSearchRequest) (adminGroupPage, error) {
	r.used = true
	return adminGroupPage{}, nil
}
func (r *adminFallbackRepo) GetGroup(context.Context, string) (AdminGroupItem, bool, error) {
	r.used = true
	return AdminGroupItem{}, false, nil
}
func (r *adminFallbackRepo) DismissGroup(context.Context, string) (AdminGroupItem, bool, error) {
	r.used = true
	return AdminGroupItem{}, false, nil
}
func (r *adminFallbackRepo) ListGroupMemberIDs(context.Context, string) ([]string, error) {
	r.used = true
	return nil, nil
}
func (r *adminFallbackRepo) SearchReports(context.Context, AdminReportListRequest) (adminReportPage, error) {
	r.used = true
	return adminReportPage{}, nil
}
func (r *adminFallbackRepo) GetReportHandleRecord(context.Context, string) (reportHandleRecord, bool, error) {
	r.used = true
	return reportHandleRecord{}, false, nil
}
func (r *adminFallbackRepo) HandleReport(context.Context, string, ReportAction, string, *UserInfo) (AdminReportItem, bool, error) {
	r.used = true
	return AdminReportItem{}, false, nil
}
func (r *adminFallbackRepo) RecallGroupMessage(context.Context, string) error {
	r.used = true
	return nil
}
func (r *adminFallbackRepo) RecallPrivateMessage(context.Context, string) error {
	r.used = true
	return nil
}
func (r *adminFallbackRepo) DeleteMoment(context.Context, string) error {
	r.used = true
	return nil
}
func (r *adminFallbackRepo) DeleteComment(context.Context, string) error {
	r.used = true
	return nil
}

type adminRoleStub struct {
	RoleID      string
	Name        string
	Description string
	Permissions []string
	Deleted     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type adminGrantStub struct {
	UserID              string
	GrantAllPermissions bool
	GrantedAt           time.Time
	RoleIDs             []string
	Revoked             bool
}

type adminBackendStub struct {
	roles        map[string]adminRoleStub
	admins       map[string]adminGrantStub
	users        map[string]UserInfo
	blockedTags  map[string]struct{}
	momentTags   map[string]struct{}
	execErrs     map[string]error
	queryErrs    map[string]error
	queryRowErrs map[string]error
}

func newAdminBackendStub() *adminBackendStub {
	return &adminBackendStub{
		roles:        make(map[string]adminRoleStub),
		admins:       make(map[string]adminGrantStub),
		users:        make(map[string]UserInfo),
		blockedTags:  make(map[string]struct{}),
		momentTags:   make(map[string]struct{}),
		execErrs:     make(map[string]error),
		queryErrs:    make(map[string]error),
		queryRowErrs: make(map[string]error),
	}
}

func (b *adminBackendStub) Exec(_ context.Context, query string, args ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	switch query {
	case queryRoleInsert:
		b.roles[args[0].(string)] = adminRoleStub{RoleID: args[0].(string), Name: args[1].(string), Description: stringValue(args[2]), CreatedAt: args[3].(time.Time), UpdatedAt: args[4].(time.Time)}
	case queryRoleUpdate:
		role := b.roles[args[0].(string)]
		role.Name = args[1].(string)
		role.Description = stringValue(args[2])
		role.UpdatedAt = args[3].(time.Time)
		b.roles[role.RoleID] = role
	case queryRoleDelete:
		role := b.roles[args[0].(string)]
		role.Deleted = true
		role.UpdatedAt = args[1].(time.Time)
		b.roles[role.RoleID] = role
	case queryRoleReplacePermissions:
		role := b.roles[args[0].(string)]
		role.Permissions = nil
		b.roles[role.RoleID] = role
	case queryRoleInsertPermission:
		role := b.roles[args[0].(string)]
		role.Permissions = append(role.Permissions, args[1].(string))
		sort.Strings(role.Permissions)
		b.roles[role.RoleID] = role
	case queryRoleInsertPermissionsBatch:
		// args: roleID, now, []string keys — mirror the per-permission insert.
		role := b.roles[args[0].(string)]
		for _, key := range args[2].([]string) {
			role.Permissions = append(role.Permissions, key)
		}
		sort.Strings(role.Permissions)
		b.roles[role.RoleID] = role
	case queryAdminUpsert:
		grant := b.admins[args[0].(string)]
		grant.UserID = args[0].(string)
		grant.GrantAllPermissions = args[1].(bool)
		grant.GrantedAt = args[2].(time.Time)
		grant.Revoked = false
		b.admins[grant.UserID] = grant
	case queryAdminReplaceRoles:
		grant := b.admins[args[0].(string)]
		grant.RoleIDs = nil
		b.admins[grant.UserID] = grant
	case queryAdminInsertRole:
		grant := b.admins[args[0].(string)]
		grant.RoleIDs = append(grant.RoleIDs, args[1].(string))
		b.admins[grant.UserID] = grant
	case queryAdminRevoke:
		grant := b.admins[args[0].(string)]
		grant.Revoked = true
		b.admins[grant.UserID] = grant
	case queryBlockTag:
		b.blockedTags[args[0].(string)] = struct{}{}
	case queryDeleteMomentTagAssignments:
		delete(b.momentTags, args[0].(string))
	}
	return nil
}

func (b *adminBackendStub) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	if err := b.queryRowErrs[query]; err != nil {
		return adminRowValues{err: err}
	}
	switch query {
	case queryRoleByID:
		role, ok := b.roles[args[0].(string)]
		if !ok {
			return adminRowValues{err: pgx.ErrNoRows}
		}
		return adminRowValues{values: []any{role.RoleID, role.Name, role.Description, role.CreatedAt, role.UpdatedAt, role.Deleted}}
	case queryRoleByName:
		for _, role := range b.roles {
			if strings.EqualFold(role.Name, args[0].(string)) {
				return adminRowValues{values: []any{role.RoleID, role.Name, role.Description, role.CreatedAt, role.UpdatedAt, role.Deleted}}
			}
		}
		return adminRowValues{err: pgx.ErrNoRows}
	case queryUserExists:
		user, ok := b.users[args[0].(string)]
		if !ok {
			return adminRowValues{err: pgx.ErrNoRows}
		}
		return adminRowValues{values: []any{user.UserID, user.Nickname, user.AvatarURL}}
	case queryTagDeleteState:
		tagName := args[0].(string)
		_, isBlocked := b.blockedTags[tagName]
		_, hasMomentRefs := b.momentTags[tagName]
		return adminRowValues{values: []any{isBlocked, hasMomentRefs}}
	default:
		return adminRowValues{err: persistence.ErrNotImplemented}
	}
}

func (b *adminBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	switch query {
	case queryRoleList:
		rows := make([][]any, 0)
		for _, role := range b.roles {
			if role.Deleted {
				continue
			}
			rows = append(rows, []any{role.RoleID, role.Name, role.Description, append([]string(nil), role.Permissions...)})
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i][1].(string) == rows[j][1].(string) {
				return rows[i][0].(string) < rows[j][0].(string)
			}
			return rows[i][1].(string) < rows[j][1].(string)
		})
		return &adminRowsValues{rows: rows}, nil
	case queryRolePermissions:
		role := b.roles[args[0].(string)]
		rows := make([][]any, 0, len(role.Permissions))
		for _, permission := range role.Permissions {
			rows = append(rows, []any{permission})
		}
		return &adminRowsValues{rows: rows}, nil
	case queryAdminList, queryAdminActorDetail:
		grants := make([]adminGrantStub, 0)
		for _, grant := range b.admins {
			if grant.Revoked {
				continue
			}
			if query == queryAdminActorDetail && grant.UserID != args[0].(string) {
				continue
			}
			grants = append(grants, grant)
		}
		sort.Slice(grants, func(i, j int) bool {
			if grants[i].GrantedAt.Equal(grants[j].GrantedAt) {
				return grants[i].UserID < grants[j].UserID
			}
			return grants[i].GrantedAt.After(grants[j].GrantedAt)
		})
		rows := make([][]any, 0)
		for _, grant := range grants {
			user := b.users[grant.UserID]
			activeRoles := make([]adminRoleStub, 0, len(grant.RoleIDs))
			for _, roleID := range grant.RoleIDs {
				role := b.roles[roleID]
				if role.Deleted {
					continue
				}
				activeRoles = append(activeRoles, role)
			}
			sort.Slice(activeRoles, func(i, j int) bool {
				if activeRoles[i].Name == activeRoles[j].Name {
					return activeRoles[i].RoleID < activeRoles[j].RoleID
				}
				return activeRoles[i].Name < activeRoles[j].Name
			})
			if len(activeRoles) == 0 {
				rows = append(rows, []any{grant.UserID, user.Nickname, user.AvatarURL, grant.GrantAllPermissions, grant.GrantedAt, "", "", []string{}})
				continue
			}
			for _, role := range activeRoles {
				rows = append(rows, []any{grant.UserID, user.Nickname, user.AvatarURL, grant.GrantAllPermissions, grant.GrantedAt, role.RoleID, role.Name, append([]string(nil), role.Permissions...)})
			}
		}
		return &adminRowsValues{rows: rows}, nil
	default:
		return &persistence.InMemoryRows{}, nil
	}
}

func stringValue(v any) string {
	if v == nil {
		return ""
	}
	return v.(string)
}

func TestPostgresRepositoryRoleAndAdminFlows(t *testing.T) {
	backend := newAdminBackendStub()
	backend.users["user_bob"] = UserInfo{UserID: "user_bob", Nickname: "Bob", AvatarURL: "avatar"}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	created, err := repo.CreateRole(context.Background(), AdminRbacRoleCreateRequest{Name: "Reviewer"})
	if err != nil || created.RoleID == "" {
		t.Fatalf("unexpected created role: %#v err=%v", created, err)
	}
	if err := repo.SetRolePermissions(context.Background(), created.RoleID, []string{"admin.reports.read"}); err != nil {
		t.Fatalf("set permissions: %v", err)
	}
	roles, err := repo.ListRoles(context.Background())
	if err != nil || len(roles) != 1 || roles[0].Permissions[0] != "admin.reports.read" {
		t.Fatalf("unexpected roles: %#v err=%v", roles, err)
	}
	if err := repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "user_bob", RoleIDs: []string{created.RoleID}}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	actor, ok, err := repo.GetActor(context.Background(), "user_bob")
	if err != nil || !ok || !containsString(actor.Permissions, "admin.reports.read") {
		t.Fatalf("unexpected actor: %#v ok=%v err=%v", actor, ok, err)
	}
	list, err := repo.ListAdmins(context.Background(), PaginationParams{Limit: 10})
	if err != nil || len(list.Items) != 1 || list.Items[0].User.UserID != "user_bob" {
		t.Fatalf("unexpected list: %#v err=%v", list, err)
	}
	if err := repo.RevokeAdmin(context.Background(), "user_bob"); err != nil {
		t.Fatalf("revoke admin: %v", err)
	}
	_, ok, err = repo.GetActor(context.Background(), "user_bob")
	if err != nil || ok {
		t.Fatalf("expected revoked actor to be missing, ok=%v err=%v", ok, err)
	}
}

func TestPostgresRepositoryReturnsErrorOnQueryFailure(t *testing.T) {
	backend := newAdminBackendStub()
	backend.queryErrs[queryRoleList] = errors.New("query failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.ListRoles(context.Background())
	if err == nil {
		t.Fatal("expected list roles error")
	}
}

func TestPersistenceModerationStoreDeleteTagSoftBlockKeepsAssignments(t *testing.T) {
	backend := newAdminBackendStub()
	backend.momentTags["hotspot"] = struct{}{}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	store := NewPersistenceModerationStore(factory, nil)

	if err := store.DeleteTag(context.Background(), "hotspot", false); err != nil {
		t.Fatalf("delete tag: %v", err)
	}
	if _, ok := backend.blockedTags["hotspot"]; !ok {
		t.Fatalf("expected blocked tag marker, got %#v", backend.blockedTags)
	}
	if _, ok := backend.momentTags["hotspot"]; !ok {
		t.Fatalf("expected moment tag assignments to survive soft block, got %#v", backend.momentTags)
	}
}

func TestPersistenceModerationStoreDeleteTagHardDeleteRemovesMomentAssignments(t *testing.T) {
	backend := newAdminBackendStub()
	backend.momentTags["hotspot"] = struct{}{}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	store := NewPersistenceModerationStore(factory, nil)

	if err := store.DeleteTag(context.Background(), "hotspot", true); err != nil {
		t.Fatalf("hard delete tag: %v", err)
	}
	if _, ok := backend.momentTags["hotspot"]; ok {
		t.Fatalf("expected moment tag assignments removed after hard delete, got %#v", backend.momentTags)
	}
}

func TestPersistenceModerationStoreDeleteTagReturnsNotFound(t *testing.T) {
	backend := newAdminBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	store := NewPersistenceModerationStore(factory, nil)

	if err := store.DeleteTag(context.Background(), "missing", false); !errors.Is(err, errTagNotFound) {
		t.Fatalf("expected tag not found, got %v", err)
	}
}

package groups

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/idgen"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

// Create wraps the group insert and its owner-member insert in a single
// transaction so a failure of the second statement cannot leave an orphan
// group with no members (which would be unreachable and impossible to
// dissolve through normal flows). Mirrors ApproveApplication/AcceptInvite.
func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateGroupRequest) (GroupItem, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return GroupItem{}, persistence.ErrNotImplemented
	}
	if _, ok := r.factory.DB.(persistence.QueryableDB); !ok {
		return GroupItem{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	groupID, err := idgen.WithPrefix("group_")
	if err != nil {
		return GroupItem{}, err
	}
	isSearchable := true
	if req.IsSearchable != nil {
		isSearchable = *req.IsSearchable
	}
	membersSearchable := true
	if req.MembersSearchable != nil {
		membersSearchable = *req.MembersSearchable
	}
	maxMembers := req.MaxMembers
	if maxMembers == 0 {
		maxMembers = 200
	}
	joinMethod := req.JoinMethod
	if joinMethod == "" {
		joinMethod = "direct"
	}
	requiresApproval := false
	if req.RequiresApproval != nil {
		requiresApproval = *req.RequiresApproval
	}
	nowRFC3339 := now.Format(time.RFC3339)
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	item := GroupItem{Group: GroupInfo{GroupID: groupID, Name: req.Name, AvatarURL: types.StrPtr(req.AvatarFileID), MembersCount: 1}, Description: types.StrPtr(req.Description), Tags: tags, HasPassword: req.Password != "", Location: newRelativeLocationPtr(req.Location), ViewerState: &GroupViewerState{Role: "owner", UnreadCount: 0, IsMuted: false, IsPinned: false, JoinedAt: nowRFC3339}, JoinMethod: joinMethod, RequiresApproval: requiresApproval, IsSearchable: isSearchable, MembersSearchable: membersSearchable, MaxMembers: maxMembers, CreatedAt: &nowRFC3339, OwnerID: &userID}
	var passwordHash *string
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return GroupItem{}, fmt.Errorf("groups.CreateGroup: hash password: %w", err)
		}
		hashStr := string(hash)
		passwordHash = &hashStr
	}
	s2CellToken := geo.CellTokenAtLevel(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude}, 14)
	s2CellID := geo.LeafCellID(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude})
	ownerRole := string(GroupRoleOwner)
	if err := persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := queryable.Exec(txCtx, queryInsertGroup, groupID, req.Name, req.AvatarFileID, req.Description, req.Password != "", passwordHash, req.Location.LocationName, req.Location.Point.Latitude, req.Location.Point.Longitude, joinMethod, requiresApproval, nil, isSearchable, membersSearchable, maxMembers, userID, now, now, tags, s2CellToken, s2CellID); err != nil {
			return fmt.Errorf("groups.CreateGroup: insert group: %w", err)
		}
		if err := queryable.Exec(txCtx, queryInsertGroupMember, groupID, userID, ownerRole, now); err != nil {
			return fmt.Errorf("groups.CreateGroup: insert owner member: %w", err)
		}
		return nil
	}); err != nil {
		return GroupItem{}, err
	}
	return item, nil
}

func (r *PostgresRepository) Update(ctx context.Context, userID string, req UpdateGroupRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	var hasPassword *bool
	var passwordHash *string
	var joinMethodOverride *string
	if req.ClearPassword != nil && *req.ClearPassword {
		cleared := false
		hasPassword = &cleared
		// 清除密码即解除 password 门槛：join_method 联动退化为 direct，不留「password 群无密码」的矛盾态
		direct := string(GroupJoinMethodDirect)
		joinMethodOverride = &direct
		emptyHash := ""
		passwordHash = &emptyHash
	} else if req.Password != nil && *req.Password != "" {
		hasPwd := true
		hasPassword = &hasPwd
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("groups.Update: hash password: %w", err)
		}
		hashStr := string(hash)
		passwordHash = &hashStr
		pw := string(GroupJoinMethodPassword)
		joinMethodOverride = &pw
	}
	var locationName *string
	var latitude *float64
	var longitude *float64
	var s2CellToken *string
	var s2CellID *int64
	if req.Location != nil {
		locationName = &req.Location.LocationName
		latitude = &req.Location.Point.Latitude
		longitude = &req.Location.Point.Longitude
		token := geo.CellTokenAtLevel(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude}, 14)
		s2CellToken = &token
		id := geo.LeafCellID(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude})
		s2CellID = &id
	}
	_ = userID
	joinMethod := joinMethodOverride
	if joinMethod == nil {
		joinMethod = nilOrStringJoinMethod(req.JoinMethod)
	}
	err := queryable.Exec(ctx, queryUpdateGroup, req.GroupID, nilOrString(req.Name), nilOrString(req.AvatarFileID), nilOrString(req.Description), hasPassword, passwordHash, joinMethod, req.RequiresApproval, req.NextActivityAt, nilOrBool(req.IsSearchable), nilOrBool(req.MembersSearchable), nilOrInt(req.MaxMembers), locationName, latitude, longitude, nilOrStrings(req.Tags), s2CellToken, s2CellID)
	if err != nil {
		return fmt.Errorf("groups.Update: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Dismiss(ctx context.Context, groupID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	err := queryable.Exec(ctx, queryDeleteGroup, groupID)
	if err != nil {
		return fmt.Errorf("groups.Dismiss: %w", err)
	}
	return nil
}

func (r *PostgresRepository) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return false, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	var role string
	if err := queryable.QueryRow(ctx, queryGetGroupMembership, groupID, userID).Scan(&role, new(time.Time), new(bool), new(*time.Time)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetApplication loads a single application by ID. Used to replace the O(groups
// × applications) scan in findApplicationForApprover with a single lookup
// followed by an admin-rights check.
func (r *PostgresRepository) ListMyGroups(ctx context.Context, userID string, limit int, pageToken string) ([]GroupItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > types.MaxPageSize {
		limit = types.MaxPageSize
	}
	// Decode the opaque cursor (keyset over joined_at DESC, group_id DESC).
	var pageCursor myGroupsCursor
	if pageToken != "" {
		if err := cursor.Decode(myGroupsCursorNamespace, pageToken, &pageCursor); err != nil {
			return nil, "", ErrInvalidCursor
		}
	}
	// Build the query dynamically: base + optional keyset predicate + LIMIT.
	// Fetch limit+1 rows to detect whether another page exists.
	args := []any{userID}
	query := queryListMyGroupsBase
	if pageCursor.JoinedAt != "" && pageCursor.GroupID != "" {
		args = append(args, pageCursor.JoinedAt, pageCursor.GroupID)
		query += fmt.Sprintf(" AND (gm.joined_at < $%d OR (gm.joined_at = $%d AND g.group_id < $%d))", len(args)-1, len(args)-1, len(args))
	}
	args = append(args, limit+1)
	query += fmt.Sprintf(" ORDER BY gm.joined_at DESC, g.group_id DESC LIMIT $%d", len(args))

	rows, err := queryable.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	type scannedItem struct {
		item             GroupItem
		locationName     string
		latitude         float64
		longitude        float64
		joinMethod       string
		requiresApproval bool
		nextActivity     *string
		roleText         string
		createdAt        *time.Time
		lastActivityAt   *time.Time
		tags             []string
		membersCount     int
		onlineCount      int
		joinedAt         time.Time
		isMuted          bool
		mutedUntil       *time.Time
		isPinned         bool
		unreadCount      int
		// last_message 预览(LEFT JOIN LATERAL,可能为空)
		lastMessageID    *string
		lastMessageType  *string
		lastContent      *string
		lastSenderID     *string
		lastSenderName   *string
		lastSenderAvatar *string
		lastSentAt       *time.Time
	}
	scanned := make([]scannedItem, 0)
	for rows.Next() {
		var s scannedItem
		if err := rows.Scan(&s.item.Group.GroupID, &s.item.Group.Name, &s.item.Group.AvatarURL, &s.item.Description, &s.item.HasPassword, &s.locationName, &s.latitude, &s.longitude, &s.joinMethod, &s.requiresApproval, &s.nextActivity, &s.item.IsSearchable, &s.item.MembersSearchable, &s.item.MaxMembers, &s.item.OwnerID, &s.createdAt, &s.lastActivityAt, &s.tags, &s.membersCount, &s.onlineCount, &s.roleText, &s.joinedAt, &s.isMuted, &s.mutedUntil, &s.isPinned, &s.unreadCount, &s.lastMessageID, &s.lastMessageType, &s.lastContent, &s.lastSenderID, &s.lastSenderName, &s.lastSenderAvatar, &s.lastSentAt); err != nil {
			rows.Close()
			return nil, "", err
		}
		scanned = append(scanned, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	// Detect next page: if we fetched more than `limit` rows, trim and encode
	// a cursor from the last returned row.
	nextCursor := ""
	if len(scanned) > limit {
		last := scanned[limit-1]
		nextCursor = encodeMyGroupsCursor(myGroupsCursor{JoinedAt: last.joinedAt.UTC().Format(time.RFC3339Nano), GroupID: last.item.Group.GroupID})
		scanned = scanned[:limit]
	}
	items := make([]GroupItem, 0, len(scanned))
	for _, s := range scanned {
		location := RelativeLocation{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: s.latitude, Longitude: s.longitude}, LocationName: s.locationName}
		s.item.Location = &location
		s.item.Tags = s.tags
		s.item.JoinMethod = GroupJoinMethod(s.joinMethod)
		s.item.RequiresApproval = s.requiresApproval
		s.item.NextActivityAt = s.nextActivity
		s.item.OnlineMembersCount = &s.onlineCount
		s.item.CreatedAt = formatOptionalTime(s.createdAt)
		s.item.LastActivityAt = formatOptionalTime(s.lastActivityAt)
		s.item.Group.MembersCount = s.membersCount
		s.item.ViewerState = &GroupViewerState{Role: GroupRole(s.roleText), UnreadCount: s.unreadCount, IsMuted: s.isMuted, MutedUntil: formatMutedUntil(s.mutedUntil), IsPinned: s.isPinned, JoinedAt: s.joinedAt.UTC().Format(time.RFC3339)}
		if s.lastMessageID != nil && s.lastMessageType != nil && s.lastSenderID != nil && s.lastSentAt != nil {
			s.item.LastMessage = &GroupLastMessage{
				MessageID: *s.lastMessageID,
				Type:      *s.lastMessageType,
				Content:   types.StrPtr(stringValue(s.lastContent)),
				Sender: UserInfo{
					UserID:    *s.lastSenderID,
					Nickname:  stringValue(s.lastSenderName),
					AvatarURL: types.StrPtr(stringValue(s.lastSenderAvatar)),
				},
				SentAt: s.lastSentAt.UTC().Format(time.RFC3339),
			}
		}
		items = append(items, s.item)
	}
	return items, nextCursor, nil
}

func (r *PostgresRepository) JoinGroup(ctx context.Context, userID string, groupID string, password *string, message *string) (EnterGroupResult, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return EnterGroupResult{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return EnterGroupResult{}, persistence.ErrNotImplemented
	}
	group, err := getGroupBase(ctx, queryable, groupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return EnterGroupResult{}, ErrGroupNotFound
		}
		return EnterGroupResult{}, err
	}
	membership, err := loadGroupMembership(ctx, queryable, groupID, userID)
	if err != nil {
		return EnterGroupResult{}, err
	}
	if membership != nil {
		return EnterGroupResult{Status: EnterGroupStatusJoined}, nil
	}
	// invite_only：仅受邀用户可加入（存在 pending 邀请视为受邀）
	if group.JoinMethod == GroupJoinMethodInviteOnly {
		var inviteID string
		var expiresAt time.Time
		if err := queryable.QueryRow(ctx, queryFindPendingGroupInvite, groupID, userID).Scan(&inviteID, &expiresAt); err != nil {
			return EnterGroupResult{}, ErrPermissionDenied
		}
	}
	// password：bcrypt 比对（密码群的真实门槛）
	if group.JoinMethod == GroupJoinMethodPassword {
		if password == nil || *password == "" {
			return EnterGroupResult{}, ErrInvalidPassword
		}
		var storedHash *string
		if err := queryable.QueryRow(ctx, queryGetGroupPasswordHash, groupID).Scan(&storedHash); err != nil {
			return EnterGroupResult{}, err
		}
		if storedHash == nil || *storedHash == "" {
			// 迁移语义：password 群没有已设置的密码（历史数据已统一迁移为 direct，
			// 这里只剩管理员未重设密码的异常态）——拒绝加入而非放行
			return EnterGroupResult{}, ErrInvalidPassword
		}
		if bcrypt.CompareHashAndPassword([]byte(*storedHash), []byte(*password)) != nil {
			return EnterGroupResult{}, ErrInvalidPassword
		}
	}
	// requires_approval：创建加入申请，审核通过后才算真正加入
	if group.RequiresApproval {
		var existingID string
		if err := queryable.QueryRow(ctx, queryFindPendingGroupApplication, groupID, userID).Scan(&existingID); err == nil && existingID != "" {
			// 自然去重（spec api_spec_base.md）：命中已有 pending 申请，返回已有 application_id，
			// 不重复创建；IsNewApplication=false 使 service 跳过重复通知。
			return EnterGroupResult{Status: EnterGroupStatusPendingApproval, ApplicationID: &existingID}, nil
		}
		now := time.Now().UTC()
		applicationID, err := idgen.WithPrefix("app_")
		if err != nil {
			return EnterGroupResult{}, err
		}
		if err := queryable.Exec(ctx, queryInsertGroupApplication, applicationID, groupID, userID, nullableStringPtr(message), string(ApplicationStatusPending), now); err != nil {
			return EnterGroupResult{}, err
		}
		return EnterGroupResult{Status: EnterGroupStatusPendingApproval, IsNewApplication: true, ApplicationID: &applicationID}, nil
	}
	// direct：直接加入。容量检查与插入放同一事务并对群行 FOR UPDATE，
	// 并发入群在相同 max_members 快照下串行判定，杜绝竞态超员。
	if err := persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		tx, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := enforceGroupCapacity(txCtx, tx, groupID); err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Exec(txCtx, queryInsertGroupMember, groupID, userID, string(GroupRoleMember), now); err != nil {
			return fmt.Errorf("groups.JoinGroup: insert member: %w", err)
		}
		return nil
	}); err != nil {
		return EnterGroupResult{}, err
	}
	return EnterGroupResult{Status: EnterGroupStatusJoined}, nil
}

// enforceGroupCapacity 在事务内（调用方已对群行 FOR UPDATE）校验成员数未达 max_members。
func enforceGroupCapacity(ctx context.Context, tx persistence.Queryable, groupID string) error {
	var maxMembers, memberCount int
	if err := tx.QueryRow(ctx, queryLockGroupCapacity, groupID).Scan(&maxMembers, &memberCount); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGroupNotFound
		}
		return err
	}
	if maxMembers > 0 && memberCount >= maxMembers {
		return ErrGroupFull
	}
	return nil
}

func (r *PostgresRepository) UpdatePreferences(ctx context.Context, userID string, req UpdateGroupPreferencesRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupMemberSettings, req.GroupID, userID, req.IsMuted, req.IsPinned)
}

func (r *PostgresRepository) LeaveGroup(ctx context.Context, userID string, groupID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteGroupMember, groupID, userID)
}

func nilOrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// queryableForTx returns the queryable bound to ctx (an in-flight transaction)
// when one is present, otherwise falls back to the shared pool. Used by methods
// that opt into RunInTransaction so the same code path serves both the pooled
// and the transactional case.

func nilOrStringJoinMethod(g *GroupJoinMethod) *string {
	if g == nil {
		return nil
	}
	s := string(*g)
	return &s
}

func nilOrBool(b *bool) *bool {
	return b
}

func nilOrInt(i *int) *int {
	return i
}

func nullableStringPtr(value *string) any {
	if value == nil || *value == "" {
		return nil
	}
	return *value
}

func derefTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func nilOrStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return values
}

func derefStringPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

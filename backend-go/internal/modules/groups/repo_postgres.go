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

	"github.com/jackc/pgx/v5"
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
	item := GroupItem{Group: GroupInfo{GroupID: groupID, Name: req.Name, AvatarURL: req.AvatarFileID, MembersCount: 1}, Description: req.Description, Tags: tags, HasPassword: req.Password != "", Location: newRelativeLocation(req.Location), MyState: &MyGroupState{Role: "owner", IsJoined: true, UnreadCount: 0, IsMuted: false, IsPinned: false, JoinedAt: nowRFC3339}, JoinMethod: joinMethod, RequiresApproval: requiresApproval, IsSearchable: isSearchable, MaxMembers: maxMembers, CreatedAt: &nowRFC3339, OwnerID: userID}
	s2CellToken := geo.CellTokenAtLevel(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude}, 14)
	s2CellID := geo.LeafCellID(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude})
	ownerRole := string(GroupRoleOwner)
	if err := persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := queryable.Exec(txCtx, queryInsertGroup, groupID, req.Name, req.AvatarFileID, req.Description, req.Password != "", req.Location.LocationName, req.Location.Point.Latitude, req.Location.Point.Longitude, joinMethod, requiresApproval, nil, isSearchable, maxMembers, userID, now, now, tags, s2CellToken, s2CellID); err != nil {
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
	if req.ClearPassword != nil && *req.ClearPassword {
		cleared := false
		hasPassword = &cleared
	} else if req.Password != nil {
		hasPwd := *req.Password != ""
		hasPassword = &hasPwd
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
	err := queryable.Exec(ctx, queryUpdateGroup, req.GroupID, nilOrString(req.Name), nilOrString(req.AvatarFileID), nilOrString(req.Description), hasPassword, nilOrStringJoinMethod(req.JoinMethod), req.RequiresApproval, req.NextActivity, nilOrBool(req.IsSearchable), nilOrInt(req.MaxMembers), locationName, latitude, longitude, nilOrStrings(req.Tags), s2CellToken, s2CellID)
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

func (r *PostgresRepository) ListMembers(ctx context.Context, groupID string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	roleValue := ""
	if role != nil {
		roleValue = string(*role)
	}
	rows, err := queryable.Query(ctx, queryListGroupMembers, groupID, roleValue)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupMemberItem, 0)
	for rows.Next() {
		var item GroupMemberItem
		var roleText string
		var joinedAt time.Time
		if err := rows.Scan(&item.User.UserID, &item.User.Nickname, &roleText, &joinedAt); err != nil {
			return nil, "", err
		}
		item.Role = GroupRole(roleText)
		item.JoinedAt = joinedAt.UTC().Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

// IsMember is a direct membership probe (single indexed lookup) used on hot
// paths such as realtime authorization. It replaces the previous pattern of
// paging through the whole member list to answer a yes/no question.
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
func (r *PostgresRepository) GetApplication(ctx context.Context, applicationID string) (GroupApplicationItem, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return GroupApplicationItem{}, false, persistence.ErrNotImplemented
	}
	item, err := r.loadApplication(ctx, applicationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GroupApplicationItem{}, false, nil
		}
		return GroupApplicationItem{}, false, err
	}
	return item, true, nil
}

func (r *PostgresRepository) SearchMembers(ctx context.Context, groupID string, keyword string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	roleValue := ""
	if role != nil {
		roleValue = string(*role)
	}
	rows, err := queryable.Query(ctx, querySearchGroupMembers, groupID, roleValue, keyword)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupMemberItem, 0)
	for rows.Next() {
		var item GroupMemberItem
		var roleText string
		var joinedAt time.Time
		if err := rows.Scan(&item.User.UserID, &item.User.Nickname, &roleText, &joinedAt); err != nil {
			return nil, "", err
		}
		item.Role = GroupRole(roleText)
		item.JoinedAt = joinedAt.UTC().Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

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
	if limit > 100 {
		limit = 100
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
		if err := rows.Scan(&s.item.Group.GroupID, &s.item.Group.Name, &s.item.Group.AvatarURL, &s.item.Description, &s.item.HasPassword, &s.locationName, &s.latitude, &s.longitude, &s.joinMethod, &s.requiresApproval, &s.nextActivity, &s.item.IsSearchable, &s.item.MaxMembers, &s.item.OwnerID, &s.createdAt, &s.lastActivityAt, &s.tags, &s.membersCount, &s.onlineCount, &s.roleText, &s.joinedAt, &s.isMuted, &s.mutedUntil, &s.lastMessageID, &s.lastMessageType, &s.lastContent, &s.lastSenderID, &s.lastSenderName, &s.lastSenderAvatar, &s.lastSentAt); err != nil {
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
		s.item.Location = RelativeLocation{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: s.latitude, Longitude: s.longitude}, LocationName: s.locationName}
		s.item.Tags = s.tags
		s.item.JoinMethod = GroupJoinMethod(s.joinMethod)
		s.item.RequiresApproval = s.requiresApproval
		s.item.NextActivity = s.nextActivity
		s.item.OnlineMembersCount = &s.onlineCount
		s.item.CreatedAt = formatOptionalTime(s.createdAt)
		s.item.LastActivityAt = formatOptionalTime(s.lastActivityAt)
		s.item.Group.MembersCount = s.membersCount
		s.item.MyState = &MyGroupState{Role: GroupRole(s.roleText), IsJoined: true, UnreadCount: 0, IsMuted: s.isMuted, MutedUntil: formatMutedUntil(s.mutedUntil), IsPinned: false, JoinedAt: s.joinedAt.UTC().Format(time.RFC3339)}
		if s.lastMessageID != nil && s.lastMessageType != nil && s.lastSenderID != nil && s.lastSentAt != nil {
			s.item.LastMessage = &GroupLastMessage{
				MessageID: *s.lastMessageID,
				Type:      *s.lastMessageType,
				Content:   stringValue(s.lastContent),
				Sender: UserInfo{
					UserID:    *s.lastSenderID,
					Nickname:  stringValue(s.lastSenderName),
					AvatarURL: stringValue(s.lastSenderAvatar),
				},
				SentAt: s.lastSentAt.UTC().Format(time.RFC3339),
			}
		}
		items = append(items, s.item)
	}
	return items, nextCursor, nil
}

func (r *PostgresRepository) JoinGroup(ctx context.Context, userID string, groupID string, password *string, message *string) (JoinGroupResult, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return JoinGroupResult{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return JoinGroupResult{}, persistence.ErrNotImplemented
	}
	group, err := getGroupBase(ctx, queryable, groupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return JoinGroupResult{}, ErrGroupNotFound
		}
		return JoinGroupResult{}, err
	}
	membership, err := loadGroupMembership(ctx, queryable, groupID, userID)
	if err != nil {
		return JoinGroupResult{}, err
	}
	if membership != nil {
		return JoinGroupResult{Status: JoinGroupStatusJoined}, nil
	}
	// invite_only：仅受邀用户可加入（存在 pending 邀请视为受邀）
	if group.JoinMethod == GroupJoinMethodInviteOnly {
		var inviteID string
		var expiresAt time.Time
		if err := queryable.QueryRow(ctx, queryFindPendingGroupInvite, groupID, userID).Scan(&inviteID, &expiresAt); err != nil {
			return JoinGroupResult{}, ErrPermissionDenied
		}
	}
	// password：需提供密码（has_password 为布尔标记，密码内容校验不在本实现范围）
	if group.JoinMethod == GroupJoinMethodPassword {
		if password == nil || *password == "" {
			return JoinGroupResult{}, ErrInvalidPassword
		}
	}
	// requires_approval：创建加入申请，审核通过后才算真正加入
	if group.RequiresApproval {
		var existingID string
		if err := queryable.QueryRow(ctx, queryFindPendingGroupApplication, groupID, userID).Scan(&existingID); err == nil && existingID != "" {
			// 自然去重（spec api_spec_base.md）：命中已有 pending 申请，返回已有 application_id，
			// 不重复创建；IsNewApplication=false 使 service 跳过重复通知。
			return JoinGroupResult{Status: JoinGroupStatusPendingApproval, Application: &struct {
				ApplicationID string `json:"application_id"`
			}{ApplicationID: existingID}}, nil
		}
		now := time.Now().UTC()
		applicationID, err := idgen.WithPrefix("app_")
		if err != nil {
			return JoinGroupResult{}, err
		}
		if err := queryable.Exec(ctx, queryInsertGroupApplication, applicationID, groupID, userID, nullableStringPtr(message), string(ApplicationStatusPending), now); err != nil {
			return JoinGroupResult{}, err
		}
		return JoinGroupResult{Status: JoinGroupStatusPendingApproval, IsNewApplication: true, Application: &struct {
			ApplicationID string `json:"application_id"`
		}{ApplicationID: applicationID}}, nil
	}
	// direct：直接加入
	now := time.Now().UTC()
	if err := queryable.Exec(ctx, queryInsertGroupMember, groupID, userID, string(GroupRoleMember), now); err != nil {
		return JoinGroupResult{}, fmt.Errorf("groups.JoinGroup: insert member: %w", err)
	}
	return JoinGroupResult{Status: JoinGroupStatusJoined}, nil
}

func (r *PostgresRepository) CreateAnnouncement(ctx context.Context, creatorID string, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return GroupAnnouncementItem{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return GroupAnnouncementItem{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	announcementID, err := idgen.WithPrefix("announce_")
	if err != nil {
		return GroupAnnouncementItem{}, err
	}
	isPinned := false
	if req.IsPinned != nil {
		isPinned = *req.IsPinned
	}
	if err := queryable.Exec(ctx, queryInsertGroupAnnouncement, announcementID, req.GroupID, req.Title, req.Content, creatorID, isPinned, now); err != nil {
		return GroupAnnouncementItem{}, err
	}
	return GroupAnnouncementItem{
		AnnouncementID: announcementID,
		GroupID:        req.GroupID,
		Title:          req.Title,
		Content:        req.Content,
		CreatedBy:      UserInfo{UserID: creatorID, Nickname: creatorID},
		CreatedAt:      now.Format(time.RFC3339),
		IsPinned:       isPinned,
	}, nil
}

func (r *PostgresRepository) ListAnnouncements(ctx context.Context, groupID string) ([]GroupAnnouncementItem, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListGroupAnnouncements, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]GroupAnnouncementItem, 0)
	for rows.Next() {
		var item GroupAnnouncementItem
		var createdByUserID string
		var createdAt time.Time
		var isPinned bool
		if err := rows.Scan(&item.AnnouncementID, &item.GroupID, &item.Title, &item.Content, &createdByUserID, &isPinned, &createdAt); err != nil {
			return nil, err
		}
		item.CreatedBy = UserInfo{UserID: createdByUserID, Nickname: createdByUserID}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		item.IsPinned = isPinned
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) UpdateAnnouncement(ctx context.Context, groupID string, req UpdateGroupAnnouncementRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	return queryable.Exec(ctx, queryUpdateGroupAnnouncement, req.AnnouncementID, nilOrString(derefStringPtr(req.Title)), nilOrString(derefStringPtr(req.Content)), req.IsPinned, now)
}

func (r *PostgresRepository) DeleteAnnouncement(ctx context.Context, groupID string, announcementID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteGroupAnnouncement, announcementID, time.Now().UTC())
}

func (r *PostgresRepository) ListApplications(ctx context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	rows, err := queryable.Query(ctx, queryListGroupApplications, groupID, statusValue)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupApplicationItem, 0)
	for rows.Next() {
		var item GroupApplicationItem
		var message, statusText, applicantUserID string
		var createdAt time.Time
		var handledAt *time.Time
		var handledByUserID *string
		if err := rows.Scan(&item.ApplicationID, &applicantUserID, &message, &statusText, &createdAt, &handledAt, &handledByUserID); err != nil {
			return nil, "", err
		}
		item.Applicant = UserInfo{UserID: applicantUserID, Nickname: applicantUserID}
		if message != "" {
			item.Message = &message
		}
		item.Status = ApplicationStatus(statusText)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if handledAt != nil {
			value := handledAt.UTC().Format(time.RFC3339)
			item.HandledAt = &value
		}
		if handledByUserID != nil && *handledByUserID != "" {
			item.HandledBy = &UserInfo{UserID: *handledByUserID, Nickname: *handledByUserID}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) ApproveApplication(ctx context.Context, approverID string, applicationID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	record, err := r.loadApplication(ctx, applicationID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	// Wrap the application-status update and the member insert in a single tx.
	// Previously these were two separate Execs: if the second failed the
	// application was already marked accepted with no matching member row.
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := queryable.Exec(txCtx, queryUpdateGroupApplication, applicationID, string(ApplicationStatusAccepted), now, approverID); err != nil {
			return err
		}
		return queryable.Exec(txCtx, queryInsertGroupMember, record.GroupID, record.Applicant.UserID, string(GroupRoleMember), now)
	})
}

func (r *PostgresRepository) RejectApplication(ctx context.Context, applicationID string, reason *string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupApplication, applicationID, string(ApplicationStatusRejected), time.Now().UTC(), nil)
}

func (r *PostgresRepository) UpdateMySettings(ctx context.Context, userID string, req UpdateGroupMySettingsRequest) error {
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

func (r *PostgresRepository) ListMyApplications(ctx context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	rows, err := queryable.Query(ctx, queryListMyGroupApplications, userID, statusValue)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupApplicationItem, 0)
	for rows.Next() {
		var item GroupApplicationItem
		var message, statusText, groupID string
		var createdAt time.Time
		var handledAt *time.Time
		var handledByUserID *string
		if err := rows.Scan(&item.ApplicationID, &groupID, &message, &statusText, &createdAt, &handledAt, &handledByUserID); err != nil {
			return nil, "", err
		}
		item.Applicant = UserInfo{UserID: userID, Nickname: userID}
		if message != "" {
			item.Message = &message
		}
		item.Status = ApplicationStatus(statusText)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if handledAt != nil {
			value := handledAt.UTC().Format(time.RFC3339)
			item.HandledAt = &value
		}
		if handledByUserID != nil && *handledByUserID != "" {
			item.HandledBy = &UserInfo{UserID: *handledByUserID, Nickname: *handledByUserID}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) CancelApplication(ctx context.Context, applicationID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupApplication, applicationID, string(ApplicationStatusRevoked), time.Now().UTC(), nil)
}

func (r *PostgresRepository) ListMyInvites(ctx context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupInviteItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	rows, err := queryable.Query(ctx, queryListMyInvites, userID, statusValue)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupInviteItem, 0)
	for rows.Next() {
		var item GroupInviteItem
		var inviterUserID, groupID, message, statusText, groupName, groupAvatar string
		var membersCount int
		var createdAt time.Time
		var expiresAt, respondedAt *time.Time
		if err := rows.Scan(&item.InviteID, &groupID, &groupName, &groupAvatar, &membersCount, &inviterUserID, &message, &statusText, &createdAt, &expiresAt, &respondedAt); err != nil {
			return nil, "", err
		}
		item.Group = GroupInfo{GroupID: groupID, Name: groupName, AvatarURL: groupAvatar, MembersCount: membersCount}
		item.Inviter = UserInfo{UserID: inviterUserID, Nickname: inviterUserID}
		if message != "" {
			item.Message = &message
		}
		item.Status = ApplicationStatus(statusText)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if expiresAt != nil {
			value := expiresAt.UTC().Format(time.RFC3339)
			item.ExpiresAt = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) AcceptInvite(ctx context.Context, userID string, inviteID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	invite, err := r.loadInvite(ctx, inviteID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	// Wrap the invite-status update and the member insert in a single tx so a
	// failure of the second statement rolls back the invite acceptance.
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := queryable.Exec(txCtx, queryUpdateGroupInvite, inviteID, string(ApplicationStatusAccepted), now); err != nil {
			return err
		}
		return queryable.Exec(txCtx, queryInsertGroupMember, invite.Group.GroupID, userID, string(GroupRoleMember), now)
	})
}

func (r *PostgresRepository) RejectInvite(ctx context.Context, inviteID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupInvite, inviteID, string(ApplicationStatusRejected), time.Now().UTC())
}

func (r *PostgresRepository) InviteMember(ctx context.Context, userID string, req InviteMemberRequest) (Invitation, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return Invitation{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return Invitation{}, persistence.ErrNotImplemented
	}
	var existingID string
	var existingExpiresAt *time.Time
	if err := queryable.QueryRow(ctx, queryFindPendingGroupInvite, req.GroupID, req.UserID).Scan(&existingID, &existingExpiresAt); err == nil && existingID != "" {
		return Invitation{InviteID: existingID, ExpiresAt: derefTime(existingExpiresAt)}, nil
	}
	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour)
	inviteID, err := idgen.WithPrefix("invite_")
	if err != nil {
		return Invitation{}, err
	}
	if err := queryable.Exec(ctx, queryInsertGroupInvite, inviteID, req.GroupID, userID, req.UserID, nullableStringPtr(req.Message), string(ApplicationStatusPending), now, expiresAt); err != nil {
		return Invitation{}, err
	}
	return Invitation{InviteID: inviteID, ExpiresAt: expiresAt.Format(time.RFC3339)}, nil
}

func (r *PostgresRepository) ListSentInvites(ctx context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupSentInviteItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	rows, err := queryable.Query(ctx, queryListSentInvites, groupID, statusValue)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupSentInviteItem, 0)
	for rows.Next() {
		var item GroupSentInviteItem
		var inviteeUserID, message, statusText string
		var createdAt time.Time
		var expiresAt, respondedAt *time.Time
		if err := rows.Scan(&item.InviteID, &inviteeUserID, &message, &statusText, &createdAt, &expiresAt, &respondedAt); err != nil {
			return nil, "", err
		}
		item.Invitee = UserInfo{UserID: inviteeUserID, Nickname: inviteeUserID}
		if message != "" {
			item.Message = &message
		}
		item.Status = ApplicationStatus(statusText)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if expiresAt != nil {
			value := expiresAt.UTC().Format(time.RFC3339)
			item.ExpiresAt = &value
		}
		if respondedAt != nil {
			value := respondedAt.UTC().Format(time.RFC3339)
			item.RespondedAt = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) RevokeInvite(ctx context.Context, groupID string, inviteID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupInvite, inviteID, string(ApplicationStatusRevoked), time.Now().UTC())
}

func (r *PostgresRepository) SetMemberRole(ctx context.Context, userID string, req SetMemberRoleRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupMemberRole, req.GroupID, req.UserID, string(req.Role))
}

func (r *PostgresRepository) MuteMember(ctx context.Context, userID string, req MuteMemberRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	mutedUntil := time.Now().UTC().Add(time.Duration(req.DurationMinutes) * time.Minute)
	return queryable.Exec(ctx, queryMuteGroupMember, req.GroupID, req.UserID, mutedUntil)
}

func (r *PostgresRepository) UnmuteMember(ctx context.Context, userID string, req UnmuteMemberRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUnmuteGroupMember, req.GroupID, req.UserID)
}

func (r *PostgresRepository) BanMember(ctx context.Context, userID string, req BanMemberRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	if err := queryable.Exec(ctx, queryInsertGroupBan, req.GroupID, req.UserID, userID, nullableStringPtr(req.Reason), now); err != nil {
		return err
	}
	_ = queryable.Exec(ctx, queryDeleteGroupMember, req.GroupID, req.UserID)
	return nil
}

func (r *PostgresRepository) UnbanMember(ctx context.Context, userID string, req UnbanMemberRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteGroupBan, req.GroupID, req.UserID)
}

func (r *PostgresRepository) ListBannedMembers(ctx context.Context, groupID string, limit int, cursor string) ([]BannedMemberItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListGroupBans, groupID)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]BannedMemberItem, 0)
	for rows.Next() {
		var item BannedMemberItem
		var bannedAt time.Time
		var reason *string
		var bannedByUserID *string
		if err := rows.Scan(&item.User.UserID, &bannedAt, &reason, &bannedByUserID); err != nil {
			return nil, "", err
		}
		item.User.Nickname = item.User.UserID
		item.BannedAt = bannedAt.UTC().Format(time.RFC3339)
		item.Reason = reason
		if bannedByUserID != nil && *bannedByUserID != "" {
			item.BannedBy = &UserInfo{UserID: *bannedByUserID, Nickname: *bannedByUserID}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, userID string, req RemoveMemberRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteGroupMember, req.GroupID, req.UserID)
}

func (r *PostgresRepository) TransferOwnership(ctx context.Context, userID string, req TransferOwnershipRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	if err := queryable.Exec(ctx, queryUpdateGroupMemberRole, req.GroupID, userID, string(GroupRoleAdmin)); err != nil {
		return err
	}
	if err := queryable.Exec(ctx, queryUpdateGroupMemberRole, req.GroupID, req.NewOwnerID, string(GroupRoleOwner)); err != nil {
		return err
	}
	return queryable.Exec(ctx, queryUpdateGroupOwner, req.GroupID, req.NewOwnerID)
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

func (r *PostgresRepository) loadApplication(ctx context.Context, applicationID string) (GroupApplicationItem, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return GroupApplicationItem{}, persistence.ErrNotImplemented
	}
	var item GroupApplicationItem
	var groupID, applicantUserID, message, statusText string
	var createdAt time.Time
	var handledAt *time.Time
	var handledByUserID *string
	if err := queryable.QueryRow(ctx, queryGetGroupApplication, applicationID).Scan(&item.ApplicationID, &groupID, &applicantUserID, &message, &statusText, &createdAt, &handledAt, &handledByUserID); err != nil {
		return GroupApplicationItem{}, err
	}
	item.GroupID = groupID
	item.Applicant = UserInfo{UserID: applicantUserID, Nickname: applicantUserID}
	if message != "" {
		item.Message = &message
	}
	item.Status = ApplicationStatus(statusText)
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	if handledAt != nil {
		value := handledAt.UTC().Format(time.RFC3339)
		item.HandledAt = &value
	}
	if handledByUserID != nil && *handledByUserID != "" {
		item.HandledBy = &UserInfo{UserID: *handledByUserID, Nickname: *handledByUserID}
	}
	group, err := getGroupBase(ctx, queryable, groupID)
	if err == nil {
		_ = group
	}
	return item, nil
}

func (r *PostgresRepository) loadInvite(ctx context.Context, inviteID string) (GroupInviteItem, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return GroupInviteItem{}, persistence.ErrNotImplemented
	}
	var item GroupInviteItem
	var groupID, inviterUserID, inviteeUserID, message, statusText string
	var createdAt time.Time
	var expiresAt, respondedAt *time.Time
	if err := queryable.QueryRow(ctx, queryGetGroupInvite, inviteID).Scan(&item.InviteID, &groupID, &inviterUserID, &inviteeUserID, &message, &statusText, &createdAt, &expiresAt, &respondedAt); err != nil {
		return GroupInviteItem{}, err
	}
	group, err := getGroupBase(ctx, queryable, groupID)
	if err != nil {
		return GroupInviteItem{}, err
	}
	count, err := countGroupMembers(ctx, queryable, groupID)
	if err != nil {
		return GroupInviteItem{}, err
	}
	group.Group.MembersCount = count
	item.Group = group.Group
	item.Inviter = UserInfo{UserID: inviterUserID, Nickname: inviterUserID}
	if message != "" {
		item.Message = &message
	}
	item.Status = ApplicationStatus(statusText)
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	if expiresAt != nil {
		value := expiresAt.UTC().Format(time.RFC3339)
		item.ExpiresAt = &value
	}
	_ = respondedAt
	_ = inviteeUserID
	return item, nil
}

package groups

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type PostgresRepository struct {
	factory  *persistence.Factory
	fallback Repository
}

func NewPostgresRepository(factory *persistence.Factory, fallback Repository) *PostgresRepository {
	if fallback == nil {
		fallback = NewInMemoryRepository()
	}
	return &PostgresRepository{factory: factory, fallback: fallback}
}

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateGroupRequest) (GroupItem, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Create(ctx, userID, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return GroupItem{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	groupID := fmt.Sprintf("group_%d", now.UnixNano())
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
		joinMethod = "open"
	}
	item := GroupItem{Group: GroupInfo{GroupID: groupID, Name: req.Name, AvatarURL: req.AvatarFileID, MembersCount: 1}, Description: req.Description, Tags: req.Tags, HasPassword: req.Password != "", Location: req.Location, MyState: &MyGroupState{Role: "owner", IsJoined: true, UnreadCount: 0, IsMuted: false, IsPinned: false, JoinedAt: now.Format(time.RFC3339)}, JoinMethod: joinMethod, IsSearchable: isSearchable, MaxMembers: maxMembers}
	if err := queryable.Exec(ctx, queryInsertGroup, groupID, req.Name, req.AvatarFileID, req.Description, req.Password != "", req.Location.LocationName, req.Location.Point.Latitude, req.Location.Point.Longitude, joinMethod, isSearchable, maxMembers); err != nil {
		return GroupItem{}, err
	}
	if err := queryable.Exec(ctx, queryInsertGroupMember, groupID, userID, string(GroupRoleOwner), now); err != nil {
		return GroupItem{}, err
	}
	return item, nil
}

func (r *PostgresRepository) Update(ctx context.Context, userID string, req UpdateGroupRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Update(ctx, userID, req)
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
	if req.Location != nil {
		locationName = &req.Location.LocationName
		latitude = &req.Location.Point.Latitude
		longitude = &req.Location.Point.Longitude
	}
	_ = userID
	err := queryable.Exec(ctx, queryUpdateGroup, req.GroupID, nilOrString(req.Name), nilOrString(req.AvatarFileID), nilOrString(req.Description), hasPassword, nilOrStringJoinMethod(req.JoinMethod), nilOrBool(req.IsSearchable), nilOrInt(req.MaxMembers), locationName, latitude, longitude)
	return err
}

func (r *PostgresRepository) Dismiss(ctx context.Context, groupID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Dismiss(ctx, groupID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	err := queryable.Exec(ctx, queryDeleteGroup, groupID)
	return err
}

func (r *PostgresRepository) ListMembers(ctx context.Context, groupID string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListMembers(ctx, groupID, role, limit, cursor)
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
		if err := rows.Scan(&item.User.UserID, &roleText, &joinedAt); err != nil {
			return nil, "", err
		}
		item.User.Nickname = item.User.UserID
		item.Role = GroupRole(roleText)
		item.JoinedAt = joinedAt.UTC().Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) SearchMembers(ctx context.Context, groupID string, keyword string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	items, _, err := r.ListMembers(ctx, groupID, role, limit, cursor)
	if err != nil {
		return nil, "", err
	}
	filtered := make([]GroupMemberItem, 0, len(items))
	for _, item := range items {
		if keyword == "" || item.User.UserID == keyword || item.User.Nickname == keyword {
			filtered = append(filtered, item)
		}
	}
	return filtered, "", nil
}

func (r *PostgresRepository) ListMyGroups(ctx context.Context, userID string, limit int, cursor string) ([]GroupItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListMyGroups(ctx, userID, limit, cursor)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListMyGroups, userID)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]GroupItem, 0)
	for rows.Next() {
		var item GroupItem
		var locationName, joinMethod, roleText string
		var latitude, longitude float64
		var joinedAt time.Time
		if err := rows.Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &item.Description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.IsSearchable, &item.MaxMembers, &roleText, &joinedAt); err != nil {
			return nil, "", err
		}
		item.Location = Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
		item.JoinMethod = GroupJoinMethod(joinMethod)
		item.MyState = &MyGroupState{Role: GroupRole(roleText), IsJoined: true, UnreadCount: 0, IsMuted: false, IsPinned: false, JoinedAt: joinedAt.UTC().Format(time.RFC3339)}
		item.Group.MembersCount, err = countGroupMembers(ctx, queryable, item.Group.GroupID)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) JoinGroup(ctx context.Context, userID string, groupID string, password *string, message *string) (JoinGroupResult, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.JoinGroup(ctx, userID, groupID, password, message)
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
	if group.JoinMethod == GroupJoinMethodInviteOnly {
		return JoinGroupResult{}, ErrPermissionDenied
	}
	if group.JoinMethod == GroupJoinMethodApproval {
		var existingID string
		if err := queryable.QueryRow(ctx, queryFindPendingGroupApplication, groupID, userID).Scan(&existingID); err == nil && existingID != "" {
			return JoinGroupResult{Status: JoinGroupStatusPendingApproval, Application: &struct {
				ApplicationID string `json:"application_id"`
			}{ApplicationID: existingID}}, nil
		}
		now := time.Now().UTC()
		applicationID := fmt.Sprintf("app_%d", now.UnixNano())
		if err := queryable.Exec(ctx, queryInsertGroupApplication, applicationID, groupID, userID, nullableStringPtr(message), string(ApplicationStatusPending), now); err != nil {
			return JoinGroupResult{}, err
		}
		return JoinGroupResult{Status: JoinGroupStatusPendingApproval, Application: &struct {
			ApplicationID string `json:"application_id"`
		}{ApplicationID: applicationID}}, nil
	}
	if group.HasPassword && (password == nil || *password == "") {
		return JoinGroupResult{}, ErrInvalidPassword
	}
	now := time.Now().UTC()
	if err := queryable.Exec(ctx, queryInsertGroupMember, groupID, userID, string(GroupRoleMember), now); err != nil {
		return JoinGroupResult{}, err
	}
	return JoinGroupResult{Status: JoinGroupStatusJoined}, nil
}

func (r *PostgresRepository) CreateAnnouncement(ctx context.Context, creatorID string, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	return r.fallback.CreateAnnouncement(ctx, creatorID, req)
}

func (r *PostgresRepository) ListAnnouncements(ctx context.Context, groupID string) ([]GroupAnnouncementItem, error) {
	return r.fallback.ListAnnouncements(ctx, groupID)
}

func (r *PostgresRepository) UpdateAnnouncement(ctx context.Context, groupID string, req UpdateGroupAnnouncementRequest) error {
	return r.fallback.UpdateAnnouncement(ctx, groupID, req)
}

func (r *PostgresRepository) DeleteAnnouncement(ctx context.Context, groupID string, announcementID string) error {
	return r.fallback.DeleteAnnouncement(ctx, groupID, announcementID)
}

func (r *PostgresRepository) ListApplications(ctx context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListApplications(ctx, groupID, status, limit, cursor)
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
		return r.fallback.ApproveApplication(ctx, approverID, applicationID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	record, err := r.loadApplication(ctx, applicationID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := queryable.Exec(ctx, queryUpdateGroupApplication, applicationID, string(ApplicationStatusAccepted), now, approverID); err != nil {
		return err
	}
	return queryable.Exec(ctx, queryInsertGroupMember, record.GroupID, record.Applicant.UserID, string(GroupRoleMember), now)
}

func (r *PostgresRepository) RejectApplication(ctx context.Context, applicationID string, reason *string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.RejectApplication(ctx, applicationID, reason)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupApplication, applicationID, string(ApplicationStatusRejected), time.Now().UTC(), nil)
}

func (r *PostgresRepository) UpdateMySettings(ctx context.Context, userID string, req UpdateGroupMySettingsRequest) error {
	return r.fallback.UpdateMySettings(ctx, userID, req)
}

func (r *PostgresRepository) LeaveGroup(ctx context.Context, userID string, groupID string) error {
	return r.fallback.LeaveGroup(ctx, userID, groupID)
}

func (r *PostgresRepository) ListMyApplications(ctx context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListMyApplications(ctx, userID, status, limit, cursor)
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
		return r.fallback.CancelApplication(ctx, applicationID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupApplication, applicationID, string(ApplicationStatusRevoked), time.Now().UTC(), nil)
}

func (r *PostgresRepository) ListMyInvites(ctx context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupInviteItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListMyInvites(ctx, userID, status, limit, cursor)
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
		return r.fallback.AcceptInvite(ctx, userID, inviteID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	invite, err := r.loadInvite(ctx, inviteID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := queryable.Exec(ctx, queryUpdateGroupInvite, inviteID, string(ApplicationStatusAccepted), now); err != nil {
		return err
	}
	return queryable.Exec(ctx, queryInsertGroupMember, invite.Group.GroupID, userID, string(GroupRoleMember), now)
}

func (r *PostgresRepository) RejectInvite(ctx context.Context, inviteID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.RejectInvite(ctx, inviteID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupInvite, inviteID, string(ApplicationStatusRejected), time.Now().UTC())
}

func (r *PostgresRepository) InviteMember(ctx context.Context, userID string, req InviteMemberRequest) (Invitation, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.InviteMember(ctx, userID, req)
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
	inviteID := fmt.Sprintf("invite_%d", now.UnixNano())
	if err := queryable.Exec(ctx, queryInsertGroupInvite, inviteID, req.GroupID, userID, req.UserID, nullableStringPtr(req.Message), string(ApplicationStatusPending), now, expiresAt); err != nil {
		return Invitation{}, err
	}
	return Invitation{InviteID: inviteID, ExpiresAt: expiresAt.Format(time.RFC3339)}, nil
}

func (r *PostgresRepository) ListSentInvites(ctx context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupSentInviteItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListSentInvites(ctx, groupID, status, limit, cursor)
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
		return r.fallback.RevokeInvite(ctx, groupID, inviteID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupInvite, inviteID, string(ApplicationStatusRevoked), time.Now().UTC())
}

func (r *PostgresRepository) SetMemberRole(ctx context.Context, userID string, req SetMemberRoleRequest) error {
	return r.fallback.SetMemberRole(ctx, userID, req)
}

func (r *PostgresRepository) MuteMember(ctx context.Context, userID string, req MuteMemberRequest) error {
	return r.fallback.MuteMember(ctx, userID, req)
}

func (r *PostgresRepository) UnmuteMember(ctx context.Context, userID string, req UnmuteMemberRequest) error {
	return r.fallback.UnmuteMember(ctx, userID, req)
}

func (r *PostgresRepository) BanMember(ctx context.Context, userID string, req BanMemberRequest) error {
	return r.fallback.BanMember(ctx, userID, req)
}

func (r *PostgresRepository) UnbanMember(ctx context.Context, userID string, req UnbanMemberRequest) error {
	return r.fallback.UnbanMember(ctx, userID, req)
}

func (r *PostgresRepository) ListBannedMembers(ctx context.Context, groupID string, limit int, cursor string) ([]BannedMemberItem, string, error) {
	return r.fallback.ListBannedMembers(ctx, groupID, limit, cursor)
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, userID string, req RemoveMemberRequest) error {
	return r.fallback.RemoveMember(ctx, userID, req)
}

func (r *PostgresRepository) TransferOwnership(ctx context.Context, userID string, req TransferOwnershipRequest) error {
	return r.fallback.TransferOwnership(ctx, userID, req)
}

func nilOrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

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

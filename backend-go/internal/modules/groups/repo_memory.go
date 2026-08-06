package groups

import (
	"context"
	"fmt"
	"hotspot/backend-go/internal/platform/types"
	"sort"
	"strings"
	"sync"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/geo"
)

type inMemoryInviteRecord struct {
	InviteID    string
	GroupID     string
	InviterID   string
	InviteeID   string
	Status      ApplicationStatus
	Message     *string
	CreatedAt   string
	ExpiresAt   *string
	RespondedAt *string
}

type inMemoryApplicationRecord struct {
	ApplicationID string
	GroupID       string
	ApplicantID   string
	Status        ApplicationStatus
	Message       *string
	CreatedAt     string
	HandledAt     *string
	HandledBy     *UserInfo
}

type inMemoryMembership struct {
	Role       GroupRole
	JoinedAt   string
	IsMuted    bool
	MutedUntil string
}

type inMemoryBanRecord struct {
	BannedAt string
	Reason   *string
	BannedBy string
}

type InMemoryRepository struct {
	mu            sync.RWMutex
	groups        map[string]GroupItem
	members       map[string]map[string]inMemoryMembership
	banned        map[string]map[string]inMemoryBanRecord
	invites       map[string]inMemoryInviteRecord
	applications  map[string]inMemoryApplicationRecord
	announcements map[string]GroupAnnouncementItem
	// seq disambiguates group ids generated within the same nanosecond so the
	// in-memory map never overwrites an entry on rapid successive Create calls.
	seq uint64
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		groups:        make(map[string]GroupItem),
		members:       make(map[string]map[string]inMemoryMembership),
		announcements: make(map[string]GroupAnnouncementItem),
		banned:        make(map[string]map[string]inMemoryBanRecord),
		invites:       make(map[string]inMemoryInviteRecord),
		applications:  make(map[string]inMemoryApplicationRecord),
	}
}

func (r *InMemoryRepository) Create(_ context.Context, userID string, req CreateGroupRequest) (GroupItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	nowRFC3339 := now.Format(time.RFC3339)
	r.seq++
	groupID := fmt.Sprintf("group_%d_%d", now.UnixNano(), r.seq)
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
	item := GroupItem{Group: GroupInfo{GroupID: groupID, Name: req.Name, AvatarURL: req.AvatarFileID, MembersCount: 1}, Description: req.Description, Tags: req.Tags, HasPassword: req.Password != "", Location: newRelativeLocation(req.Location), MyState: &MyGroupState{Role: "owner", IsJoined: true, UnreadCount: 0, IsMuted: false, IsPinned: false, JoinedAt: nowRFC3339}, JoinMethod: joinMethod, RequiresApproval: requiresApproval, IsSearchable: isSearchable, MaxMembers: maxMembers, CreatedAt: &nowRFC3339, OwnerID: userID}
	r.groups[groupID] = item
	r.members[groupID] = map[string]inMemoryMembership{userID: {Role: GroupRoleOwner, JoinedAt: nowRFC3339}}
	return item, nil
}
func (r *InMemoryRepository) Get(ctx context.Context, groupID string) (GroupItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.groups[groupID]
	if !ok {
		return GroupItem{}, false, nil
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return r.cloneGroupForUser(item, principal.UserID), true, nil
}
func (r *InMemoryRepository) Search(ctx context.Context, req SearchGroupsRequest) (PaginatedGroups, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	req = normalizedSearchRequest(req)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	items := make([]GroupItem, 0)
	metas := make(map[string]groupSearchMeta)
	var (
		plan    geo.SearchPlan
		hasPlan bool
	)
	if req.Radius != nil {
		plan = geo.NewSearchPlan(searchGeoPoint(req.ReferencePoint), *req.Radius, req.SortBy)
		hasPlan = true
	}
	for _, item := range r.groups {
		if !item.IsSearchable {
			continue
		}
		if req.Keyword != "" {
			keyword := strings.ToLower(req.Keyword)
			if !strings.Contains(strings.ToLower(item.Group.Name), keyword) && !strings.Contains(strings.ToLower(item.Description), keyword) {
				continue
			}
		}
		if req.Tag != "" {
			matchedTag := false
			for _, tag := range item.Tags {
				if strings.EqualFold(tag, req.Tag) {
					matchedTag = true
					break
				}
			}
			if !matchedTag {
				continue
			}
		}
		meta := groupSearchMeta{CreatedAt: groupCreatedAt(item), NameMatch: searchNameMatchScore(item, req.Keyword)}
		if hasPlan {
			distance, bearing, ok := plan.Match(groupLocationPoint(item.Location))
			if !ok {
				continue
			}
			meta.Distance = distance
			meta.Bearing = bearing
		}
		cloned := r.cloneGroupForUser(item, principal.UserID)
		items = append(items, cloned)
		metas[cloned.Group.GroupID] = meta
		if req.SortBy == searchSortRelevance && req.Keyword == "" {
			metas[cloned.Group.GroupID] = meta
		}
	}
	sortGroupsForSearch(items, metas, req.SortBy)
	cursor, err := decodeSearchCursor(types.PaginationCursor(req.Pagination))
	if err != nil {
		return PaginatedGroups{}, err
	}
	if cursor.GroupID != "" {
		filtered := make([]GroupItem, 0, len(items))
		for _, item := range items {
			if matchesSearchCursor(metas[item.Group.GroupID], item, cursor) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	limit := types.PaginationLimitWith(req.Pagination, defaultSearchLimit, maxSearchLimit)
	response := PaginatedGroups{}
	if len(items) <= limit {
		response.Items = items
		return response, nil
	}
	response.Items = items[:limit]
	last := response.Items[len(response.Items)-1]
	lastMeta := metas[last.Group.GroupID]
	next, err := encodeSearchCursor(groupSearchCursor{SortBy: req.SortBy, GroupID: last.Group.GroupID, Distance: lastMeta.Distance, MembersCount: last.Group.MembersCount, NameMatch: lastMeta.NameMatch, CreatedAt: lastMeta.CreatedAt.Format(time.RFC3339)})
	if err != nil {
		return PaginatedGroups{}, err
	}
	response.NextCursor = next
	return response, nil
}

func (r *InMemoryRepository) Update(_ context.Context, userID string, req UpdateGroupRequest) error {
	_ = userID
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.groups[req.GroupID]
	if !ok {
		return ErrGroupNotFound
	}
	if req.Name != "" {
		item.Group.Name = req.Name
	}
	if req.Description != "" {
		item.Description = req.Description
	}
	if req.AvatarFileID != "" {
		item.Group.AvatarURL = req.AvatarFileID
	}
	if req.Tags != nil {
		item.Tags = req.Tags
	}
	if req.ClearPassword != nil && *req.ClearPassword {
		item.HasPassword = false
	} else if req.Password != nil {
		item.HasPassword = *req.Password != ""
	}
	if req.JoinMethod != nil {
		item.JoinMethod = *req.JoinMethod
	}
	if req.RequiresApproval != nil {
		item.RequiresApproval = *req.RequiresApproval
	}
	if req.NextActivity != nil {
		item.NextActivity = req.NextActivity
	}
	if req.IsSearchable != nil {
		item.IsSearchable = *req.IsSearchable
	}
	if req.MaxMembers != nil {
		item.MaxMembers = *req.MaxMembers
	}
	if req.Location != nil {
		item.Location = newRelativeLocation(*req.Location)
	}
	r.groups[req.GroupID] = item
	return nil
}

func (r *InMemoryRepository) Dismiss(_ context.Context, groupID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[groupID]; !ok {
		return ErrGroupNotFound
	}
	delete(r.groups, groupID)
	delete(r.members, groupID)
	for inviteID, invite := range r.invites {
		if invite.GroupID == groupID {
			delete(r.invites, inviteID)
		}
	}
	for applicationID, application := range r.applications {
		if application.GroupID == groupID {
			delete(r.applications, applicationID)
		}
	}
	return nil
}

func (r *InMemoryRepository) ListMembers(_ context.Context, groupID string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	group, ok := r.groups[groupID]
	if !ok {
		return nil, "", ErrGroupNotFound
	}
	items := make([]GroupMemberItem, 0)
	for userID, membership := range r.members[groupID] {
		if role != nil && membership.Role != *role {
			continue
		}
		items = append(items, GroupMemberItem{User: UserInfo{UserID: userID, Nickname: userID}, Role: membership.Role, JoinedAt: membership.JoinedAt})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].JoinedAt == items[j].JoinedAt {
			return items[i].User.UserID < items[j].User.UserID
		}
		return items[i].JoinedAt < items[j].JoinedAt
	})
	for i := range items {
		if items[i].User.UserID == group.Group.GroupID+"_owner" {
			items[i].User.Nickname = group.Group.Name + " Owner"
		}
	}
	if len(items) > limit {
		return items[:limit], items[limit-1].JoinedAt + ":" + items[limit-1].User.UserID, nil
	}
	return items, "", nil
}

// IsMember is a direct membership probe used on hot paths (realtime authz),
// replacing the prior pattern of paging the whole member list.
func (r *InMemoryRepository) IsMember(_ context.Context, groupID, userID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if members, ok := r.members[groupID]; ok {
		_, isMember := members[userID]
		return isMember, nil
	}
	return false, nil
}

// GetApplication loads a single application by ID (O(1) map lookup) for paths
// that previously scanned all of an approver's groups and each group's
// applications to find one record.
func (r *InMemoryRepository) GetApplication(_ context.Context, applicationID string) (GroupApplicationItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.applications[applicationID]
	if !ok {
		return GroupApplicationItem{}, false, nil
	}
	return GroupApplicationItem{
		ApplicationID: record.ApplicationID,
		GroupID:       record.GroupID,
		Applicant:     UserInfo{UserID: record.ApplicantID, Nickname: record.ApplicantID},
		Message:       record.Message,
		Status:        record.Status,
		CreatedAt:     record.CreatedAt,
		HandledAt:     record.HandledAt,
		HandledBy:     record.HandledBy,
	}, true, nil
}

func (r *InMemoryRepository) SearchMembers(ctx context.Context, groupID string, keyword string, role *GroupRole, limit int, cursor string) ([]GroupMemberItem, string, error) {
	items, _, err := r.ListMembers(ctx, groupID, role, limit, cursor)
	if err != nil {
		return nil, "", err
	}
	filtered := make([]GroupMemberItem, 0, len(items))
	for _, item := range items {
		if keyword == "" || item.User.Nickname == keyword || item.User.UserID == keyword {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) > limit {
		return filtered[:limit], filtered[limit-1].JoinedAt + ":" + filtered[limit-1].User.UserID, nil
	}
	return filtered, "", nil
}

func (r *InMemoryRepository) ListMyGroups(_ context.Context, userID string, limit int, pageToken string) ([]GroupItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	// Decode the opaque keyset cursor (joined_at DESC, group_id DESC) so the
	// in-memory path mirrors the postgres pagination contract.
	var pageCursor myGroupsCursor
	if pageToken != "" {
		if err := cursor.Decode(myGroupsCursorNamespace, pageToken, &pageCursor); err != nil {
			return nil, "", ErrInvalidCursor
		}
	}
	items := []GroupItem{}
	for _, group := range r.groups {
		member, joined := r.members[group.Group.GroupID][userID]
		if !joined {
			continue
		}
		clone := r.cloneGroupForUser(group, userID)
		if clone.MyState != nil {
			clone.MyState.JoinedAt = member.JoinedAt
		}
		items = append(items, clone)
	}
	// Sort by joined_at DESC, then group_id DESC (matches the SQL ORDER BY).
	sort.Slice(items, func(i, j int) bool {
		ji, jj := items[i].MyState.JoinedAt, items[j].MyState.JoinedAt
		if ji != jj {
			return ji > jj
		}
		return items[i].Group.GroupID > items[j].Group.GroupID
	})
	// Apply the keyset predicate: skip rows until we're past the cursor.
	if pageCursor.GroupID != "" && pageCursor.JoinedAt != "" {
		filtered := items[:0]
		for _, it := range items {
			if it.MyState == nil {
				continue
			}
			if it.MyState.JoinedAt < pageCursor.JoinedAt ||
				(it.MyState.JoinedAt == pageCursor.JoinedAt && it.Group.GroupID < pageCursor.GroupID) {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}
	// Fetch limit+1 to detect a following page.
	nextCursor := ""
	if len(items) > limit {
		last := items[limit-1]
		if last.MyState != nil {
			nextCursor = encodeMyGroupsCursor(myGroupsCursor{JoinedAt: last.MyState.JoinedAt, GroupID: last.Group.GroupID})
		}
		items = items[:limit]
	}
	return items, nextCursor, nil
}

func (r *InMemoryRepository) JoinGroup(_ context.Context, userID, groupID string, password *string, message *string) (JoinGroupResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	group, ok := r.groups[groupID]
	if !ok {
		return JoinGroupResult{}, ErrGroupNotFound
	}
	if _, joined := r.members[groupID][userID]; joined {
		return JoinGroupResult{Status: JoinGroupStatusJoined}, nil
	}
	// invite_only：仅受邀用户可加入（内存实现：存在 pending 邀请视为受邀）
	if group.JoinMethod == GroupJoinMethodInviteOnly {
		invited := false
		for _, inv := range r.invites {
			if inv.GroupID == groupID && inv.InviteeID == userID && inv.Status == ApplicationStatusPending {
				invited = true
				break
			}
		}
		if !invited {
			return JoinGroupResult{}, ErrAccessRestricted
		}
	}
	// password：需提供密码
	if group.JoinMethod == GroupJoinMethodPassword {
		if password == nil || *password == "" {
			return JoinGroupResult{}, ErrInvalidPassword
		}
	}
	// requires_approval：创建加入申请，审核通过后才算真正加入
	if group.RequiresApproval {
		// 自然去重：命中已有 pending 申请返回已有 application_id（与 postgres 实现一致）
		for _, record := range r.applications {
			if record.GroupID == groupID && record.ApplicantID == userID && record.Status == ApplicationStatusPending {
				return JoinGroupResult{Status: JoinGroupStatusPendingApproval, Application: &struct {
					ApplicationID string `json:"application_id"`
				}{ApplicationID: record.ApplicationID}}, nil
			}
		}
		now := time.Now().UTC()
		applicationID := "app_" + now.Format("20060102150405.000000000")
		record := inMemoryApplicationRecord{ApplicationID: applicationID, GroupID: groupID, ApplicantID: userID, Status: ApplicationStatusPending, Message: message, CreatedAt: now.Format(time.RFC3339)}
		r.applications[applicationID] = record
		return JoinGroupResult{Status: JoinGroupStatusPendingApproval, IsNewApplication: true, Application: &struct {
			ApplicationID string `json:"application_id"`
		}{ApplicationID: applicationID}}, nil
	}
	if group.MaxMembers > 0 && len(r.members[groupID]) >= group.MaxMembers {
		return JoinGroupResult{}, ErrGroupFull
	}
	if r.members[groupID] == nil {
		r.members[groupID] = make(map[string]inMemoryMembership)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	r.members[groupID][userID] = inMemoryMembership{Role: GroupRoleMember, JoinedAt: now}
	group.Group.MembersCount = len(r.members[groupID])
	r.groups[groupID] = group
	return JoinGroupResult{Status: JoinGroupStatusJoined}, nil
}

func (r *InMemoryRepository) CreateAnnouncement(_ context.Context, creatorID string, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	r.seq++
	announcementID := fmt.Sprintf("announce_%d_%d", now.UnixNano(), r.seq)
	isPinned := false
	if req.IsPinned != nil {
		isPinned = *req.IsPinned
	}
	item := GroupAnnouncementItem{
		AnnouncementID: announcementID,
		GroupID:        req.GroupID,
		Title:          req.Title,
		Content:        req.Content,
		CreatedBy:      UserInfo{UserID: creatorID, Nickname: creatorID},
		IsPinned:       isPinned,
		CreatedAt:      now.Format(time.RFC3339),
	}
	r.announcements[announcementID] = item
	return item, nil
}

func (r *InMemoryRepository) ListAnnouncements(_ context.Context, groupID string) ([]GroupAnnouncementItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return nil, ErrGroupNotFound
	}
	items := make([]GroupAnnouncementItem, 0)
	for _, item := range r.announcements {
		if item.GroupID == groupID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (r *InMemoryRepository) UpdateAnnouncement(_ context.Context, groupID string, req UpdateGroupAnnouncementRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[groupID]; !ok {
		return ErrGroupNotFound
	}
	item, ok := r.announcements[req.AnnouncementID]
	if !ok || item.GroupID != groupID {
		return ErrAnnouncementNotFound
	}
	if req.Title != nil {
		item.Title = *req.Title
	}
	if req.Content != nil {
		item.Content = *req.Content
	}
	if req.IsPinned != nil {
		item.IsPinned = *req.IsPinned
	}
	r.announcements[req.AnnouncementID] = item
	return nil
}

func (r *InMemoryRepository) DeleteAnnouncement(_ context.Context, groupID string, announcementID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[groupID]; !ok {
		return ErrGroupNotFound
	}
	item, ok := r.announcements[announcementID]
	if !ok || item.GroupID != groupID {
		return ErrAnnouncementNotFound
	}
	delete(r.announcements, announcementID)
	return nil
}

func (r *InMemoryRepository) ListApplications(_ context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return nil, "", ErrGroupNotFound
	}
	items := make([]GroupApplicationItem, 0)
	for _, record := range r.applications {
		if record.GroupID != groupID {
			continue
		}
		if status != nil && *status != record.Status {
			continue
		}
		items = append(items, GroupApplicationItem{
			ApplicationID: record.ApplicationID,
			Applicant:     UserInfo{UserID: record.ApplicantID, Nickname: record.ApplicantID},
			Message:       record.Message,
			Status:        record.Status,
			CreatedAt:     record.CreatedAt,
			HandledAt:     record.HandledAt,
			HandledBy:     record.HandledBy,
		})
	}
	return items, "", nil
}

func (r *InMemoryRepository) ApproveApplication(_ context.Context, approverID, applicationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.applications[applicationID]
	if !ok {
		return ErrGroupNotFound
	}
	if record.Status != ApplicationStatusPending {
		return ErrAlreadyProcessed
	}
	record.Status = ApplicationStatusAccepted
	now := time.Now().UTC().Format(time.RFC3339)
	record.HandledAt = &now
	record.HandledBy = &UserInfo{UserID: approverID, Nickname: approverID}
	r.applications[applicationID] = record
	if r.members[record.GroupID] == nil {
		r.members[record.GroupID] = make(map[string]inMemoryMembership)
	}
	r.members[record.GroupID][record.ApplicantID] = inMemoryMembership{Role: GroupRoleMember, JoinedAt: now}
	if group, ok := r.groups[record.GroupID]; ok {
		group.Group.MembersCount = len(r.members[record.GroupID])
		r.groups[record.GroupID] = group
	}
	return nil
}

func (r *InMemoryRepository) RejectApplication(_ context.Context, applicationID string, reason *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.applications[applicationID]
	if !ok {
		return ErrGroupNotFound
	}
	if record.Status != ApplicationStatusPending {
		return ErrAlreadyProcessed
	}
	record.Status = ApplicationStatusRejected
	now := time.Now().UTC().Format(time.RFC3339)
	record.HandledAt = &now
	if reason != nil {
		record.Message = reason
	}
	r.applications[applicationID] = record
	return nil
}

func (r *InMemoryRepository) UpdateMySettings(_ context.Context, userID string, req UpdateGroupMySettingsRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	member, ok := r.members[req.GroupID][userID]
	if !ok {
		return ErrNotMember
	}
	if req.IsMuted != nil {
		member.IsMuted = *req.IsMuted
	}
	r.members[req.GroupID][userID] = member
	return nil
}

func (r *InMemoryRepository) LeaveGroup(_ context.Context, userID string, groupID string) error {
	_ = userID
	r.mu.Lock()
	defer r.mu.Unlock()
	group, ok := r.groups[groupID]
	if !ok {
		return ErrGroupNotFound
	}
	if group.MyState == nil || !group.MyState.IsJoined {
		return ErrNotMember
	}
	if group.MyState.Role == GroupRoleOwner {
		return ErrAccessRestricted
	}
	group.MyState = nil
	r.groups[groupID] = group
	return nil
}

func (r *InMemoryRepository) ListMyApplications(_ context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupApplicationItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]GroupApplicationItem, 0)
	for _, record := range r.applications {
		if record.ApplicantID != userID {
			continue
		}
		if status != nil && *status != record.Status {
			continue
		}
		items = append(items, GroupApplicationItem{ApplicationID: record.ApplicationID, Applicant: UserInfo{UserID: record.ApplicantID, Nickname: record.ApplicantID}, Message: record.Message, Status: record.Status, CreatedAt: record.CreatedAt, HandledAt: record.HandledAt, HandledBy: record.HandledBy})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].ApplicationID > items[j].ApplicationID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, "", nil
}

func (r *InMemoryRepository) CancelApplication(_ context.Context, applicationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.applications[applicationID]
	if !ok {
		return ErrGroupNotFound
	}
	record.Status = ApplicationStatusRevoked
	r.applications[applicationID] = record
	return nil
}

func (r *InMemoryRepository) ListMyInvites(_ context.Context, userID string, status *ApplicationStatus, limit int, cursor string) ([]GroupInviteItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]GroupInviteItem, 0)
	for _, record := range r.invites {
		if record.InviteeID != userID {
			continue
		}
		if status != nil && *status != record.Status {
			continue
		}
		group := r.groups[record.GroupID]
		items = append(items, GroupInviteItem{InviteID: record.InviteID, Group: group.Group, Inviter: UserInfo{UserID: record.InviterID, Nickname: record.InviterID}, Message: record.Message, Status: record.Status, CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].InviteID > items[j].InviteID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, "", nil
}

func (r *InMemoryRepository) AcceptInvite(_ context.Context, userID string, inviteID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.invites[inviteID]
	if !ok {
		return ErrGroupNotFound
	}
	if record.Status != ApplicationStatusPending {
		return ErrInviteExpired
	}
	record.Status = ApplicationStatusAccepted
	now := time.Now().UTC().Format(time.RFC3339)
	record.RespondedAt = &now
	r.invites[inviteID] = record
	if r.members[record.GroupID] == nil {
		r.members[record.GroupID] = make(map[string]inMemoryMembership)
	}
	r.members[record.GroupID][userID] = inMemoryMembership{Role: GroupRoleMember, JoinedAt: now}
	if group, ok := r.groups[record.GroupID]; ok {
		group.Group.MembersCount = len(r.members[record.GroupID])
		r.groups[record.GroupID] = group
	}
	return nil
}

func (r *InMemoryRepository) RejectInvite(_ context.Context, inviteID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.invites[inviteID]
	if !ok {
		return ErrGroupNotFound
	}
	if record.Status != ApplicationStatusPending {
		return ErrInviteExpired
	}
	record.Status = ApplicationStatusRejected
	now := time.Now().UTC().Format(time.RFC3339)
	record.RespondedAt = &now
	r.invites[inviteID] = record
	return nil
}

func (r *InMemoryRepository) InviteMember(_ context.Context, userID string, req InviteMemberRequest) (Invitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	group, ok := r.groups[req.GroupID]
	if !ok {
		return Invitation{}, ErrGroupNotFound
	}
	if _, member := r.members[req.GroupID][req.UserID]; member {
		return Invitation{}, ErrAlreadyMember
	}
	for _, record := range r.invites {
		if record.GroupID == req.GroupID && record.InviteeID == req.UserID && record.Status == ApplicationStatusPending {
			return Invitation{InviteID: record.InviteID, ExpiresAt: derefString(record.ExpiresAt)}, nil
		}
	}
	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour).Format(time.RFC3339)
	inviteID := "invite_" + now.Format("20060102150405.000000000")
	record := inMemoryInviteRecord{InviteID: inviteID, GroupID: req.GroupID, InviterID: userID, InviteeID: req.UserID, Status: ApplicationStatusPending, Message: req.Message, CreatedAt: now.Format(time.RFC3339), ExpiresAt: &expiresAt}
	r.invites[inviteID] = record
	group.Group.MembersCount = len(r.members[req.GroupID])
	r.groups[req.GroupID] = group
	return Invitation{InviteID: inviteID, ExpiresAt: expiresAt}, nil
}

func (r *InMemoryRepository) ListSentInvites(_ context.Context, groupID string, status *ApplicationStatus, limit int, cursor string) ([]GroupSentInviteItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return nil, "", ErrGroupNotFound
	}
	items := make([]GroupSentInviteItem, 0)
	for _, record := range r.invites {
		if record.GroupID != groupID {
			continue
		}
		if status != nil && *status != record.Status {
			continue
		}
		items = append(items, GroupSentInviteItem{InviteID: record.InviteID, Invitee: UserInfo{UserID: record.InviteeID, Nickname: record.InviteeID}, Status: record.Status, Message: record.Message, CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt, RespondedAt: record.RespondedAt})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].InviteID > items[j].InviteID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, "", nil
}

func (r *InMemoryRepository) RevokeInvite(_ context.Context, groupID string, inviteID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[groupID]; !ok {
		return ErrGroupNotFound
	}
	record, ok := r.invites[inviteID]
	if !ok {
		return nil
	}
	record.Status = ApplicationStatusRevoked
	now := time.Now().UTC().Format(time.RFC3339)
	record.RespondedAt = &now
	r.invites[inviteID] = record
	return nil
}

func (r *InMemoryRepository) SetMemberRole(_ context.Context, userID string, req SetMemberRoleRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	members, ok := r.members[req.GroupID]
	if !ok {
		return ErrNotMember
	}
	membership, ok := members[req.UserID]
	if !ok {
		return ErrNotMember
	}
	membership.Role = req.Role
	members[req.UserID] = membership
	_ = userID
	return nil
}

func (r *InMemoryRepository) MuteMember(_ context.Context, userID string, req MuteMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	members, ok := r.members[req.GroupID]
	if !ok {
		return ErrNotMember
	}
	membership, ok := members[req.UserID]
	if !ok {
		return ErrNotMember
	}
	membership.IsMuted = true
	mutedUntil := time.Now().UTC().Add(time.Duration(req.DurationMinutes) * time.Minute)
	membership.MutedUntil = mutedUntil.Format(time.RFC3339)
	members[req.UserID] = membership
	_ = userID
	return nil
}

func (r *InMemoryRepository) UnmuteMember(_ context.Context, userID string, req UnmuteMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	members, ok := r.members[req.GroupID]
	if !ok {
		return ErrNotMember
	}
	membership, ok := members[req.UserID]
	if !ok {
		return ErrNotMember
	}
	membership.IsMuted = false
	membership.MutedUntil = ""
	members[req.UserID] = membership
	_ = userID
	return nil
}

func (r *InMemoryRepository) BanMember(_ context.Context, userID string, req BanMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	if r.banned[req.GroupID] == nil {
		r.banned[req.GroupID] = make(map[string]inMemoryBanRecord)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	r.banned[req.GroupID][req.UserID] = inMemoryBanRecord{BannedAt: now, Reason: req.Reason, BannedBy: userID}
	if members, ok := r.members[req.GroupID]; ok {
		delete(members, req.UserID)
		group := r.groups[req.GroupID]
		group.Group.MembersCount = len(members)
		r.groups[req.GroupID] = group
	}
	return nil
}

func (r *InMemoryRepository) UnbanMember(_ context.Context, userID string, req UnbanMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	if banned, ok := r.banned[req.GroupID]; ok {
		delete(banned, req.UserID)
	}
	_ = userID
	return nil
}

func (r *InMemoryRepository) ListBannedMembers(_ context.Context, groupID string, limit int, cursor string) ([]BannedMemberItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return nil, "", ErrGroupNotFound
	}
	_ = cursor
	items := make([]BannedMemberItem, 0, len(r.banned[groupID]))
	for userID, record := range r.banned[groupID] {
		item := BannedMemberItem{
			User:     UserInfo{UserID: userID, Nickname: userID},
			BannedAt: record.BannedAt,
			Reason:   record.Reason,
		}
		if record.BannedBy != "" {
			item.BannedBy = &UserInfo{UserID: record.BannedBy, Nickname: record.BannedBy}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].BannedAt > items[j].BannedAt
	})
	if limit <= 0 {
		limit = 20
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, "", nil
}

func (r *InMemoryRepository) RemoveMember(_ context.Context, userID string, req RemoveMemberRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[req.GroupID]; !ok {
		return ErrGroupNotFound
	}
	members, ok := r.members[req.GroupID]
	if !ok {
		return nil
	}
	if _, ok := members[req.UserID]; !ok {
		return nil
	}
	delete(members, req.UserID)
	group := r.groups[req.GroupID]
	group.Group.MembersCount = len(members)
	r.groups[req.GroupID] = group
	_ = userID
	return nil
}

func (r *InMemoryRepository) TransferOwnership(_ context.Context, userID string, req TransferOwnershipRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	group, ok := r.groups[req.GroupID]
	if !ok {
		return ErrGroupNotFound
	}
	members, ok := r.members[req.GroupID]
	if !ok {
		return ErrNotMember
	}
	membership, ok := members[req.NewOwnerID]
	if !ok {
		return ErrNotMember
	}
	if oldMembership, ok := members[userID]; ok {
		oldMembership.Role = GroupRoleAdmin
		members[userID] = oldMembership
	}
	membership.Role = GroupRoleOwner
	members[req.NewOwnerID] = membership
	group.OwnerID = req.NewOwnerID
	r.groups[req.GroupID] = group
	return nil
}

func (r *InMemoryRepository) cloneGroupForUser(group GroupItem, userID string) GroupItem {
	cloned := group
	if memberships, ok := r.members[group.Group.GroupID]; ok {
		cloned.Group.MembersCount = len(memberships)
		if membership, joined := memberships[userID]; joined {
			state := &MyGroupState{Role: membership.Role, IsJoined: true, UnreadCount: 0, IsMuted: membership.IsMuted, MutedUntil: membership.MutedUntil, IsPinned: false, JoinedAt: membership.JoinedAt}
			cloned.MyState = state
		} else {
			cloned.MyState = nil
		}
	}
	return cloned
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

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
	IsPinned   bool
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
	item := GroupItem{Group: GroupInfo{GroupID: groupID, Name: req.Name, AvatarURL: req.AvatarFileID, MembersCount: 1}, Description: req.Description, Tags: req.Tags, HasPassword: req.Password != "", Location: newRelativeLocation(req.Location), MyState: &MyGroupState{Role: "owner", IsJoined: true, UnreadCount: 0, IsMuted: false, IsPinned: false, JoinedAt: nowRFC3339}, JoinMethod: joinMethod, RequiresApproval: requiresApproval, IsSearchable: isSearchable, MembersSearchable: membersSearchable, MaxMembers: maxMembers, CreatedAt: &nowRFC3339, OwnerID: userID}
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
	if req.MembersSearchable != nil {
		item.MembersSearchable = *req.MembersSearchable
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

func (r *InMemoryRepository) ListMyGroups(_ context.Context, userID string, limit int, pageToken string) ([]GroupItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	if limit > types.MaxPageSize {
		limit = types.MaxPageSize
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

func (r *InMemoryRepository) UpdatePreferences(_ context.Context, userID string, req UpdateGroupPreferencesRequest) error {
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
	if req.IsPinned != nil {
		member.IsPinned = *req.IsPinned
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

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

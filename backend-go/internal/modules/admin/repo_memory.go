package admin

import (
	groupsmodule "hotspot/backend-go/internal/modules/groups"
	usersmodule "hotspot/backend-go/internal/modules/users"

	"sync"
	"time"
)

type roleRecord struct {
	item      AdminRoleItem
	deleted   bool
	updatedAt time.Time
}

type adminGrantRecord struct {
	userID              string
	grantAllPermissions bool
	roleIDs             []string
	grantedAt           time.Time
	revoked             bool
}

type InMemoryRepository struct {
	mu           sync.RWMutex
	roles        map[string]roleRecord
	admins       map[string]adminGrantRecord
	users        map[string]UserInfo
	adminUsers   map[string]AdminUserItem
	groups       map[string]AdminGroupItem
	groupMembers map[string][]string
	reports      map[string]reportHandleRecord
	// seq 保证同纳秒内连续创建的 ID 不碰撞（UnixNano 时间戳在快速循环下会重复），
	// 同时服务 role 与 audit ID 生成。
	seq             uint64
	banStates       map[string]UserBanState
	userSessions    map[string][]string
	auditLogs       map[string]AuditLogEntry
	auditOrder      []string
	deletedRoleIDs  map[string]struct{}
	revokedAdmins   map[string]struct{}
	moments         map[string]momentRecord
	comments        map[string]commentRecord
	groupMessages   map[string]messageRecord
	privateMessages map[string]messageRecord
}

type momentRecord struct {
	ID        string
	Content   string
	AuthorID  string
	CreatedAt time.Time
	Deleted   bool
}

type commentRecord struct {
	ID        string
	Content   string
	AuthorID  string
	MomentID  string
	CreatedAt time.Time
	Deleted   bool
}

type messageRecord struct {
	ID        string
	Content   string
	SenderID  string
	CreatedAt time.Time
	Recalled  bool
}

func NewInMemoryRepository() *InMemoryRepository {
	now := time.Now().UTC()
	aliceCreatedAt := now.Add(-72 * time.Hour).Format(time.RFC3339)
	bobCreatedAt := now.Add(-48 * time.Hour).Format(time.RFC3339)
	cindyCreatedAt := now.Add(-24 * time.Hour).Format(time.RFC3339)
	adminCreatedAt := now.Add(-96 * time.Hour).Format(time.RFC3339)

	adminUsers := map[string]AdminUserItem{
		"admin": {
			User:              UserInfo{UserID: "admin", Nickname: "admin"},
			Status:            UserAccountStatusNormal,
			CreatedAt:         adminCreatedAt,
			LastActiveAt:      now.Add(-time.Hour).Format(time.RFC3339),
			LoginID:           "admin",
			PresenceStatus:    usersmodule.UserPresenceOnline,
			LastOnline:        now.Add(-time.Hour).Format(time.RFC3339),
			RegisterIP:        strPtr("127.0.0.1"),
			RegisterLocation:  "Local",
			LastLoginIP:       strPtr("127.0.0.1"),
			LastLoginLocation: "Local",
		},
		"user_alice": {
			User:              UserInfo{UserID: "user_alice", Nickname: "Alice"},
			Status:            UserAccountStatusNormal,
			CreatedAt:         aliceCreatedAt,
			LastActiveAt:      now.Add(-2 * time.Hour).Format(time.RFC3339),
			LoginID:           "alice",
			PresenceStatus:    usersmodule.UserPresenceOnline,
			LastOnline:        now.Add(-2 * time.Hour).Format(time.RFC3339),
			RegisterIP:        strPtr("10.0.0.2"),
			RegisterLocation:  "Shanghai",
			LastLoginIP:       strPtr("10.0.0.3"),
			LastLoginLocation: "Shanghai",
		},
		"user_bob": {
			User:              UserInfo{UserID: "user_bob", Nickname: "Bob"},
			Status:            UserAccountStatusNormal,
			CreatedAt:         bobCreatedAt,
			LastActiveAt:      now.Add(-3 * time.Hour).Format(time.RFC3339),
			LoginID:           "bob",
			PresenceStatus:    usersmodule.UserPresenceOffline,
			LastOnline:        now.Add(-3 * time.Hour).Format(time.RFC3339),
			RegisterIP:        strPtr("10.0.0.4"),
			RegisterLocation:  "Hangzhou",
			LastLoginIP:       strPtr("10.0.0.4"),
			LastLoginLocation: "Hangzhou",
		},
		"user_cindy": {
			User:              UserInfo{UserID: "user_cindy", Nickname: "Cindy"},
			Status:            UserAccountStatusNormal,
			CreatedAt:         cindyCreatedAt,
			LastActiveAt:      now.Add(-4 * time.Hour).Format(time.RFC3339),
			LoginID:           "cindy",
			PresenceStatus:    usersmodule.UserPresenceOffline,
			LastOnline:        now.Add(-4 * time.Hour).Format(time.RFC3339),
			RegisterIP:        strPtr("10.0.0.5"),
			RegisterLocation:  "Beijing",
			LastLoginIP:       strPtr("10.0.0.5"),
			LastLoginLocation: "Beijing",
		},
	}

	groups := map[string]AdminGroupItem{
		"group_hotspot": {
			Group: groupsmodule.GroupInfo{GroupID: "group_hotspot", Name: "HotSpot", MembersCount: 2},
			Location: groupsmodule.Location{Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 31.2304, Longitude: 121.4737}, LocationName: "Shanghai"},
			CreatedAt:      now.Add(-36 * time.Hour).Format(time.RFC3339),
			Owner:          UserInfo{UserID: "user_alice", Nickname: "Alice"},
			JoinMethod:     groupsmodule.GroupJoinMethod("direct"),
			IsSearchable:   true,
			MaxMembers:     200,
			LastActivityAt: now.Add(-30 * time.Minute).Format(time.RFC3339),
		},
	}

	reports := map[string]reportHandleRecord{
		"report_1": {
			Item: AdminReportItem{
				ReportID:   "report_1",
				TargetType: ReportTargetTypeUser,
				TargetID:   "user_bob",
				Reason:     "spam",
				Status:     ReportStatusPending,
				CreatedAt:  now.Add(-6 * time.Hour).Format(time.RFC3339),
				Reporter:   UserInfo{UserID: "user_alice", Nickname: "Alice"},
			},
			TargetUser: UserInfo{UserID: "user_bob", Nickname: "Bob"},
		},
	}

	users := map[string]UserInfo{}
	for userID, item := range adminUsers {
		users[userID] = item.User
	}

	repo := &InMemoryRepository{
		roles:           make(map[string]roleRecord),
		admins:          make(map[string]adminGrantRecord),
		users:           users,
		adminUsers:      adminUsers,
		groups:          groups,
		groupMembers:    map[string][]string{"group_hotspot": {"user_alice", "user_bob"}},
		reports:         reports,
		banStates:       map[string]UserBanState{"admin": {Status: UserAccountStatusNormal}, "user_alice": {Status: UserAccountStatusNormal}, "user_bob": {Status: UserAccountStatusNormal}, "user_cindy": {Status: UserAccountStatusNormal}},
		userSessions:    map[string][]string{"user_alice": {"sess_user_alice_1"}, "user_bob": {"sess_user_bob_1"}},
		auditLogs:       make(map[string]AuditLogEntry),
		deletedRoleIDs:  make(map[string]struct{}),
		revokedAdmins:   make(map[string]struct{}),
		moments:         make(map[string]momentRecord),
		comments:        make(map[string]commentRecord),
		groupMessages:   make(map[string]messageRecord),
		privateMessages: make(map[string]messageRecord),
	}
	repo.seedAuditLogs()
	return repo
}

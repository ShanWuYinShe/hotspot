package groups

type GroupJoinMethod string

type GroupRole string

type GroupInfo struct {
	GroupID      string `json:"group_id"`
	Name         string `json:"name"`
	AvatarURL    string `json:"avatar_url,omitempty"`
	MembersCount int    `json:"members_count"`
}

type Location struct {
	Point struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"point"`
	LocationName string `json:"location_name"`
}

type MyGroupState struct {
	Role        GroupRole `json:"role"`
	IsJoined    bool      `json:"is_joined"`
	UnreadCount int       `json:"unread_count"`
	IsMuted     bool      `json:"is_muted"`
	IsPinned    bool      `json:"is_pinned"`
	JoinedAt    string    `json:"joined_at,omitempty"`
}

type GroupItem struct {
	Group        GroupInfo       `json:"group"`
	Description  string          `json:"description,omitempty"`
	Tags         []string        `json:"tags,omitempty"`
	HasPassword  bool            `json:"has_password"`
	Location     Location        `json:"location"`
	MyState      *MyGroupState   `json:"my_state,omitempty"`
	JoinMethod   GroupJoinMethod `json:"join_method,omitempty"`
	IsSearchable bool            `json:"is_searchable,omitempty"`
	MaxMembers   int             `json:"max_members,omitempty"`
}

type CreateGroupRequest struct {
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	AvatarFileID string          `json:"avatar_file_id,omitempty"`
	Tags         []string        `json:"tags,omitempty"`
	Password     string          `json:"password,omitempty"`
	JoinMethod   GroupJoinMethod `json:"join_method,omitempty"`
	IsSearchable *bool           `json:"is_searchable,omitempty"`
	MaxMembers   int             `json:"max_members,omitempty"`
	Location     Location        `json:"location"`
}

type GroupOpenTheDoorRequest struct {
	GroupID string `json:"group_id"`
}

type SearchGroupsRequest struct {
	Keyword string `json:"keyword,omitempty"`
}

type PaginatedGroups struct {
	Items      []GroupItem `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

package notifications

type NotificationItem struct {
	NotificationID string         `json:"notification_id"`
	Type           string         `json:"type"`
	IsRead         bool           `json:"is_read"`
	CreatedAt      string         `json:"created_at"`
	Payload        map[string]any `json:"payload"`
}

type PaginatedNotifications struct {
	Items      []NotificationItem `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

type UnreadCountResult struct {
	Total  int            `json:"total"`
	ByType map[string]int `json:"by_type"`
}

type NotificationListRequest struct {
	Type string `json:"type,omitempty"`
}

type MarkAsReadRequest struct {
	NotificationIDs []string `json:"notification_ids"`
}

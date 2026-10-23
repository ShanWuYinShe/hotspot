package groups

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"hotspot/backend-go/internal/platform/idgen"
	"hotspot/backend-go/internal/platform/persistence"
)

// announcementsListHardCap 公告列表硬上限（审查 P2-2）：列表挂匿名端点
// /public/v1/groups/announcements/fetch，此前无 LIMIT，群管理员可无限建公告
// 再放大读取。50 覆盖置顶+常规公告的正常规模；置顶优先排序下截断只影响
// 超长尾部。SQL 侧以字面量写入 queryListGroupAnnouncements 尾部（LIMIT 不能
// 参数化），两处同步维护。每群创建数量上限需 AppConfig 决策，本轮登记不加。
const announcementsListHardCap = 50

func (r *PostgresRepository) CreateAnnouncement(ctx context.Context, creatorID string, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
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
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	// 硬上限见 queryListGroupAnnouncements 尾部 LIMIT（审查 P2-2，与
	// announcementsListHardCap 常量同步维护）。
	rows, err := queryable.Query(ctx, queryListGroupAnnouncements, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]GroupAnnouncementItem, 0)
	for rows.Next() {
		item, err := scanAnnouncementRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetAnnouncement 按归属双键直查单条（repo.go 接口注释；审查 P2-2）。
func (r *PostgresRepository) GetAnnouncement(ctx context.Context, groupID, announcementID string) (GroupAnnouncementItem, bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return GroupAnnouncementItem{}, false, persistence.ErrNotImplemented
	}
	item, err := scanAnnouncementRow(queryable.QueryRow(ctx, queryGetGroupAnnouncement, announcementID, groupID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GroupAnnouncementItem{}, false, nil
		}
		return GroupAnnouncementItem{}, false, err
	}
	return item, true, nil
}

// scanAnnouncementRow 统一公告行映射（List/Get 共用）：created_by 昵称由
// SQL LEFT JOIN users 解析（审查 P2-3，此前占位填裸 user_id）。
func scanAnnouncementRow(scanner interface{ Scan(...any) error }) (GroupAnnouncementItem, error) {
	var item GroupAnnouncementItem
	var createdByUserID, createdByNickname string
	var createdAt time.Time
	var isPinned bool
	if err := scanner.Scan(&item.AnnouncementID, &item.GroupID, &item.Title, &item.Content, &createdByUserID, &createdByNickname, &isPinned, &createdAt); err != nil {
		return GroupAnnouncementItem{}, err
	}
	item.CreatedBy = UserInfo{UserID: createdByUserID, Nickname: createdByNickname}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.IsPinned = isPinned
	return item, nil
}

func (r *PostgresRepository) UpdateAnnouncement(ctx context.Context, groupID string, req UpdateGroupAnnouncementRequest) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	// group_id 双键匹配（IDOR 防护，见 queryUpdateGroupAnnouncement 注释）。
	return queryable.Exec(ctx, queryUpdateGroupAnnouncement, req.AnnouncementID, nilOrString(derefStringPtr(req.Title)), nilOrString(derefStringPtr(req.Content)), req.IsPinned, now, groupID)
}

func (r *PostgresRepository) DeleteAnnouncement(ctx context.Context, groupID string, announcementID string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteGroupAnnouncement, announcementID, time.Now().UTC(), groupID)
}

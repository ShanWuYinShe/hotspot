package comments

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/idgen"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"

	"github.com/jackc/pgx/v5"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateCommentRequest) (CommentItem, error) {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return CommentItem{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	// 写库保留亚秒精度（timestamptz 微秒）：秒截断会让同秒评论只能靠随机主键
	// tie-break，keyset 分页不稳定（审计 P1-1）。对外展示仍按秒格式化。
	createdAtText := now.Format(time.RFC3339)
	commentID, err := idgen.WithPrefix("comment_")
	if err != nil {
		return CommentItem{}, err
	}
	var parentID any
	if req.ParentID != "" {
		parentID = req.ParentID
	}
	mentions := req.Mentions
	if mentions == nil {
		mentions = []string{}
	}
	if err := queryable.Exec(ctx, queryInsertComment, commentID, req.MomentID, userID, parentID, req.Content, now, mentions); err != nil {
		return CommentItem{}, fmt.Errorf("comments.Create insert comment: %w", err)
	}
	var nickname string
	if err := queryable.QueryRow(ctx, queryGetNickname, userID).Scan(&nickname); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			observability.L().Warn("comments.Create: nickname lookup failed, falling back to user_id", "user_id", userID, "error", err)
		}
		nickname = userID
	}
	if nickname == "" {
		nickname = userID
	}
	parentIDPtr := (*string)(nil)
	if req.ParentID != "" {
		parentIDPtr = &req.ParentID
	}
	return CommentItem{
		CommentID:    commentID,
		ParentID:     parentIDPtr,
		MomentID:     req.MomentID,
		User:         UserInfo{UserID: userID, Nickname: nickname},
		Content:      req.Content,
		Mentions:     mentions,
		CreatedAt:    createdAtText,
		RepliesCount: 0,
	}, nil
}

// SoftDelete 原子软删，返回是否实际命中（deleted=false = 并发下已被删除或本就
// 不存在，幂等 no-op，调用方据此跳过瞬间互动计数递减）。
func (r *PostgresRepository) SoftDelete(ctx context.Context, commentID string) (bool, error) {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	var deleted bool
	if err := queryable.QueryRow(ctx, querySoftDeleteComment, commentID).Scan(&deleted); err != nil {
		return false, fmt.Errorf("comments.SoftDelete exec: %w", err)
	}
	return deleted, nil
}

func (r *PostgresRepository) List(ctx context.Context, momentID string, limit int, cursor string) (PaginatedComments, error) {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return PaginatedComments{}, persistence.ErrNotImplemented
	}
	limit = normalizeCommentLimit(limit)
	cursorCreatedAt, cursorCommentID, err := decodeCommentCursor(cursor)
	if err != nil {
		return PaginatedComments{}, err
	}
	rows, err := queryable.Query(ctx, queryListComments, momentID, cursorCreatedAt, cursorCommentID, limit+1)
	if err != nil {
		return PaginatedComments{}, fmt.Errorf("comments.List query: %w", err)
	}
	defer rows.Close()
	items := make([]CommentItem, 0, limit+1)
	// rawCreatedAt 与 items 平行保存列的真实精度时间：response 的 created_at 按
	// 秒展示，但 keyset 游标必须用真实值（微秒）编码，否则同秒行翻页会重复/丢行。
	rawCreatedAt := make([]time.Time, 0, limit+1)
	for rows.Next() {
		var item CommentItem
		var userID, nickname, parentID string
		var createdAt time.Time
		if err := rows.Scan(&item.CommentID, &userID, &nickname, &parentID, &item.Content, &createdAt, &item.RepliesCount, &item.IsDeleted, &item.Mentions); err != nil {
			return PaginatedComments{}, fmt.Errorf("comments.List scan: %w", err)
		}
		item.User = UserInfo{UserID: userID, Nickname: nickname}
		item.ParentID = strPtr(parentID)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		items = append(items, item)
		rawCreatedAt = append(rawCreatedAt, createdAt.UTC())
	}
	if err := rows.Err(); err != nil {
		return PaginatedComments{}, fmt.Errorf("comments.List rows: %w", err)
	}
	if len(items) > limit {
		c := encodeCommentCursor(rawCreatedAt[limit-1].Format(time.RFC3339Nano), items[limit-1].CommentID)
		return PaginatedComments{Items: items[:limit], NextCursor: &c}, nil
	}
	return PaginatedComments{Items: items}, nil
}

func (r *PostgresRepository) Get(ctx context.Context, commentID string) (CommentItem, bool, error) {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return CommentItem{}, false, persistence.ErrNotImplemented
	}
	row := queryable.QueryRow(ctx, queryGetComment, commentID)
	var item CommentItem
	var userID, nickname, parentID string
	var createdAt time.Time
	if err := row.Scan(&item.CommentID, &item.MomentID, &userID, &nickname, &parentID, &item.Content, &createdAt, &item.RepliesCount, &item.IsDeleted, &item.Mentions); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CommentItem{}, false, nil
		}
		return CommentItem{}, false, fmt.Errorf("comments.Get scan: %w", err)
	}
	item.User = UserInfo{UserID: userID, Nickname: nickname}
	item.ParentID = strPtr(parentID)
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return item, true, nil
}

func (r *PostgresRepository) ListReplies(ctx context.Context, commentID string, limit int, cursor string) (PaginatedComments, error) {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return PaginatedComments{}, persistence.ErrNotImplemented
	}
	limit = normalizeCommentLimit(limit)
	cursorCreatedAt, cursorCommentID, err := decodeCommentCursor(cursor)
	if err != nil {
		return PaginatedComments{}, err
	}
	rows, err := queryable.Query(ctx, queryListCommentReplies, commentID, cursorCreatedAt, cursorCommentID, limit+1)
	if err != nil {
		return PaginatedComments{}, fmt.Errorf("comments.ListReplies query: %w", err)
	}
	defer rows.Close()
	items := make([]CommentItem, 0, limit+1)
	rawCreatedAt := make([]time.Time, 0, limit+1)
	for rows.Next() {
		var item CommentItem
		var userID, nickname, parentID string
		var createdAt time.Time
		if err := rows.Scan(&item.CommentID, &userID, &nickname, &parentID, &item.Content, &createdAt, &item.RepliesCount, &item.IsDeleted, &item.Mentions); err != nil {
			return PaginatedComments{}, fmt.Errorf("comments.ListReplies scan: %w", err)
		}
		item.User = UserInfo{UserID: userID, Nickname: nickname}
		item.ParentID = strPtr(parentID)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		items = append(items, item)
		rawCreatedAt = append(rawCreatedAt, createdAt.UTC())
	}
	if err := rows.Err(); err != nil {
		return PaginatedComments{}, fmt.Errorf("comments.ListReplies rows: %w", err)
	}
	if len(items) > limit {
		c := encodeCommentCursor(rawCreatedAt[limit-1].Format(time.RFC3339Nano), items[limit-1].CommentID)
		return PaginatedComments{Items: items[:limit], NextCursor: &c}, nil
	}
	return PaginatedComments{Items: items}, nil
}

// ListTopRepliesByParentIDs fetches up to `limit` newest replies for each of the
// given parent comment IDs in a single query, returning a parentID->replies
// map. It replaces the N+1 of calling ListReplies once per top-level comment.
func (r *PostgresRepository) ListTopRepliesByParentIDs(ctx context.Context, parentIDs []string, limit int) (map[string][]CommentItem, error) {
	result := make(map[string][]CommentItem, len(parentIDs))
	if len(parentIDs) == 0 || limit <= 0 {
		return result, nil
	}
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return result, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListTopRepliesByParentIDs, parentIDs, limit)
	if err != nil {
		return nil, fmt.Errorf("comments.ListTopRepliesByParentIDs query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item CommentItem
		var userID, nickname, parentID string
		var createdAt time.Time
		if err := rows.Scan(&item.CommentID, &userID, &nickname, &parentID, &item.Content, &createdAt, &item.RepliesCount, &item.IsDeleted, &item.Mentions); err != nil {
			return nil, fmt.Errorf("comments.ListTopRepliesByParentIDs scan: %w", err)
		}
		item.User = UserInfo{UserID: userID, Nickname: nickname}
		item.ParentID = strPtr(parentID)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		result[parentID] = append(result[parentID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("comments.ListTopRepliesByParentIDs rows: %w", err)
	}
	return result, nil
}

type commentCursor struct {
	At string `json:"at"`
	ID string `json:"id"`
}

// commentCursorNamespace 是评论分页游标的 AES-GCM 加密命名空间（cursor 包
// 语义：同一 secret 下不同命名空间互不通用，防止跨列表伪造游标）。
const commentCursorNamespace = "comments.list"

// queryable resolves the Queryable to use for this request: an in-flight
// transaction bound to ctx when present, otherwise the shared pool. Returns
// ok=false when persistence is not configured (in-memory mode), in which case
// callers surface ErrNotImplemented.
func queryable(ctx context.Context, factory *persistence.Factory) (persistence.Queryable, bool) {
	return persistence.RepoQueryable(ctx, factory)
}

func normalizeCommentLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > types.MaxPageSize:
		return types.MaxPageSize
	default:
		return limit
	}
}

func encodeCommentCursor(at, id string) string {
	value, err := cursor.Encode(commentCursorNamespace, commentCursor{At: at, ID: id})
	if err != nil {
		return ""
	}
	return value
}

func decodeCommentCursor(cursorValue string) (any, string, error) {
	if cursorValue == "" {
		return nil, "", nil
	}
	var decoded commentCursor
	if err := cursor.Decode(commentCursorNamespace, cursorValue, &decoded); err != nil || decoded.At == "" || decoded.ID == "" {
		return nil, "", ErrInvalidCursor
	}
	at, err := time.Parse(time.RFC3339Nano, decoded.At)
	if err != nil {
		return nil, "", ErrInvalidCursor
	}
	return at.UTC(), decoded.ID, nil
}

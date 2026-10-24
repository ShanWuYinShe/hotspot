package privatechats

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) GetOrCreate(ctx context.Context, userID, targetUserID string) (time.Time, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return time.Time{}, persistence.ErrNotImplemented
	}
	chatKey := key(userID, targetUserID)
	userAID, userBID := orderedChatUsers(userID, targetUserID)
	now := time.Now().UTC()
	if err := queryable.Exec(ctx, queryInsertPrivateChat, chatKey, userAID, userBID, now.Format(time.RFC3339Nano)); err != nil {
		return time.Time{}, err
	}
	var createdAt time.Time
	if err := queryable.QueryRow(ctx, queryGetPrivateChatCreatedAt, chatKey).Scan(&createdAt); err != nil {
		return time.Time{}, err
	}
	return createdAt, nil
}

func (r *PostgresRepository) List(ctx context.Context, userID string, opts ListOptions) ([]ChatState, string, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	// keyset 分页下推（审查 P2-7）：游标编解码收口在 Repository 层，
	// limit+1 探测下一页，SQL 侧按 (is_pinned DESC, last_activity DESC,
	// target_id ASC) 排序并 LIMIT。
	limit := types.NormalizeLimit(opts.Limit)
	cursorPinned, cursorAt, cursorTargetID, err := decodeChatCursor(opts.Cursor)
	if err != nil {
		return nil, "", err
	}
	var pinned *bool
	var at *time.Time
	var targetIDArg *string
	if !cursorAt.IsZero() {
		pinnedValue := cursorPinned
		pinned = &pinnedValue
		atValue := cursorAt.UTC()
		at = &atValue
		targetIDArg = &cursorTargetID
	}
	rows, err := queryable.Query(ctx, queryListPrivateChats, userID, pinned, at, targetIDArg, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]ChatState, 0, limit+1)
	for rows.Next() {
		state, err := scanChatStateWithActivity(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, state)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if len(items) > limit {
		items = items[:limit]
		last := items[limit-1]
		next := encodeChatCursor(last.IsPinned, last.LastActivityAt.UTC().Format(time.RFC3339Nano), last.TargetID)
		return items, next, nil
	}
	return items, "", nil
}

func (r *PostgresRepository) GetState(ctx context.Context, userID, targetUserID string) (ChatState, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return ChatState{}, persistence.ErrNotImplemented
	}
	state, err := scanChatState(queryable.QueryRow(ctx, queryGetPrivateChatState, userID, key(userID, targetUserID)))
	if err != nil {
		return ChatState{}, err
	}
	return state, nil
}

func (r *PostgresRepository) RestoreForUser(ctx context.Context, userID, targetUserID string, visibleSince time.Time) error {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return persistence.ErrNotImplemented
	}
	state, err := r.getStateImpl(ctx, userID, targetUserID)
	if err != nil {
		return err
	}
	ts := visibleSince.UTC()
	state.VisibleSince = &ts
	state.DeletedAt = nil
	return r.upsertStateImpl(ctx, userID, state)
}

func (r *PostgresRepository) UpdateSettings(ctx context.Context, userID, targetUserID string, update UpdateChatSettings) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	// 单条条件 UPDATE：并发 mute + pin 互不覆盖（此前 read-modify-write 整行
	// 覆盖，后到者用旧快照抹掉前者）。
	chatKey := key(userID, targetUserID)
	var isMuted, isPinned any
	if update.IsMuted != nil {
		isMuted = *update.IsMuted
	}
	if update.IsPinned != nil {
		isPinned = *update.IsPinned
	}
	return queryable.Exec(ctx, queryUpdatePrivateChatUserSettings, chatKey, userID, isMuted, isPinned)
}

func (r *PostgresRepository) DeleteForUser(ctx context.Context, userID, targetUserID string, deletedAt *time.Time) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	// 仅更新删除相关列：不回读整行再整行覆盖（delete 与 mute/pin 并发互不干扰）。
	chatKey := key(userID, targetUserID)
	var deletedAtValue any
	if deletedAt != nil {
		deletedAtValue = deletedAt.UTC()
	}
	return queryable.Exec(ctx, queryMarkPrivateChatDeleted, chatKey, userID, deletedAtValue)
}

func (r *PostgresRepository) upsertStateImpl(ctx context.Context, userID string, state ChatState) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	var visibleSince any
	if state.VisibleSince != nil {
		visibleSince = state.VisibleSince.UTC().Format(time.RFC3339Nano)
	}
	var deletedAt any
	if state.DeletedAt != nil {
		deletedAt = state.DeletedAt.UTC().Format(time.RFC3339Nano)
	}
	return queryable.Exec(ctx, queryUpsertPrivateChatUserState, state.ChatKey, userID, state.IsMuted, state.IsPinned, visibleSince, deletedAt)
}

func (r *PostgresRepository) getStateImpl(ctx context.Context, userID, targetUserID string) (ChatState, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return ChatState{}, persistence.ErrNotImplemented
	}
	return scanChatState(queryable.QueryRow(ctx, queryGetPrivateChatState, userID, key(userID, targetUserID)))
}

func orderedChatUsers(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

// scanChatStateWithActivity 在基础列尾部多扫 last_activity_at（列表分页查询
// 的 SQL 尾部列），供排序/游标使用。
func scanChatStateWithActivity(scanner interface{ Scan(...any) error }) (ChatState, error) {
	state, err := scanChatState(scanner, true)
	return state, err
}

func scanChatState(scanner interface{ Scan(...any) error }, withActivity ...bool) (ChatState, error) {
	var state ChatState
	var visibleSince *time.Time
	var deletedAt *time.Time
	dest := []any{&state.ChatKey, &state.TargetID, &state.CreatedAt, &state.IsMuted, &state.IsPinned, &visibleSince, &deletedAt}
	if len(withActivity) > 0 && withActivity[0] {
		dest = append(dest, &state.LastActivityAt)
	}
	if err := scanner.Scan(dest...); err != nil {
		return ChatState{}, err
	}
	if visibleSince != nil {
		ts := visibleSince.UTC()
		state.VisibleSince = &ts
	}
	if deletedAt != nil {
		ts := deletedAt.UTC()
		state.DeletedAt = &ts
	}
	return state, nil
}

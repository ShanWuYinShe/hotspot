package privatechats

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) GetOrCreate(ctx context.Context, userID, targetUserID string) (time.Time, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return time.Time{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
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
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListPrivateChats, userID)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]ChatState, 0)
	for rows.Next() {
		state, err := scanChatState(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, state)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) GetState(ctx context.Context, userID, targetUserID string) (ChatState, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return ChatState{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
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
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
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
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
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
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
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
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
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
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
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

func scanChatState(scanner interface{ Scan(...any) error }) (ChatState, error) {
	var state ChatState
	var visibleSince *time.Time
	var deletedAt *time.Time
	if err := scanner.Scan(&state.ChatKey, &state.TargetID, &state.CreatedAt, &state.IsMuted, &state.IsPinned, &visibleSince, &deletedAt); err != nil {
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

package share

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
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

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateShareRequest) (ShareCreated, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Create(ctx, userID, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return ShareCreated{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	shareID := fmt.Sprintf("share_%d", now.UnixNano())
	shareCode := strings.ToUpper(fmt.Sprintf("S%011d", now.UnixNano()%100000000000))
	createdAt := now.Format(time.RFC3339)
	expiresAt := ""
	var expiresAtValue any
	if req.ExpiresInSeconds > 0 {
		expiresAt = now.Add(time.Duration(req.ExpiresInSeconds) * time.Second).Format(time.RFC3339)
		expiresAtValue = expiresAt
	}
	if err := queryable.Exec(ctx, queryInsertShare, shareID, shareCode, userID, req.ShareType, req.Payload, createdAt, expiresAtValue, false); err != nil {
		return ShareCreated{}, err
	}
	return ShareCreated{ShareID: shareID, ShareCode: shareCode, ExpiresAt: expiresAt}, nil
}

func (r *PostgresRepository) Revoke(ctx context.Context, userID, shareID string) (bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Revoke(ctx, userID, shareID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	if _, ok, err := r.GetByID(ctx, userID, shareID); err != nil {
		return false, err
	} else if !ok {
		return false, nil
	}
	if err := queryable.Exec(ctx, queryRevokeShare, shareID, userID); err != nil {
		return false, err
	}
	return true, nil
}

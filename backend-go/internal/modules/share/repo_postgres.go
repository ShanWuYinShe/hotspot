package share

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"hotspot/backend-go/internal/platform/idgen"
	"time"

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

func (r *PostgresRepository) Create(ctx context.Context, cmd createShareCommand) (ShareCreated, error) {
	queryable, err := r.queryable()
	if err != nil {
		return ShareCreated{}, err
	}
	if err := ensureShareTargetExists(ctx, queryable, cmd.ShareType, cmd.Payload); err != nil {
		return ShareCreated{}, err
	}
	payloadJSON, err := json.Marshal(cmd.Payload)
	if err != nil {
		return ShareCreated{}, err
	}
	now := time.Now().UTC()
	shareID, err := idgen.WithPrefix("share_")
	if err != nil {
		return ShareCreated{}, err
	}
	createdAt := now.Format(time.RFC3339)
	var expiresAt string
	var expiresAtValue any
	if cmd.ExpiresAt != nil {
		expiresAt = cmd.ExpiresAt.UTC().Format(time.RFC3339)
		expiresAtValue = expiresAt
	}
	for attempt := 0; attempt < 5; attempt++ {
		shareCode, err := newShareCode()
		if err != nil {
			return ShareCreated{}, err
		}
		err = queryable.Exec(ctx, queryInsertShare, shareID, shareCode, cmd.CreatorUserID, cmd.ShareType, string(payloadJSON), createdAt, expiresAtValue, false)
		if err == nil {
			return ShareCreated{ShareID: shareID, ShareCode: shareCode, ExpiresAt: types.StrPtr(expiresAt)}, nil
		}
		if persistence.IsUniqueViolation(err) {
			continue
		}
		return ShareCreated{}, err
	}
	return ShareCreated{}, errors.New("share: code generation exhausted")
}

func (r *PostgresRepository) Revoke(ctx context.Context, cmd revokeShareCommand) error {
	queryable, err := r.queryable()
	if err != nil {
		return err
	}
	ownerUserID, revoked, err := r.getShareOwnership(ctx, queryable, cmd.ShareID)
	if err != nil {
		return err
	}
	if ownerUserID == "" {
		return ErrShareNotFound
	}
	if ownerUserID != cmd.CreatorUserID {
		return ErrSharePermission
	}
	if revoked {
		return nil
	}
	if err := queryable.Exec(ctx, queryRevokeShare, cmd.ShareID, cmd.CreatorUserID); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) queryable() (persistence.QueryableDB, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	return queryable, nil
}

func getPayloadID(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return value
}

func ensureShareTargetExists(ctx context.Context, queryable persistence.QueryableDB, shareType string, payload map[string]any) error {
	switch shareType {
	case ShareTypeUser:
		if _, _, _, err := loadSharedUser(ctx, queryable, getPayloadID(payload, "shared_user_id")); err != nil {
			return err
		}
		return nil
	case ShareTypeGroup:
		if _, _, _, _, err := loadSharedGroup(ctx, queryable, getPayloadID(payload, "shared_group_id")); err != nil {
			return err
		}
		return nil
	case ShareTypeMoment:
		if _, _, _, _, _, err := loadSharedMoment(ctx, queryable, getPayloadID(payload, "shared_moment_id")); err != nil {
			return err
		}
		return nil
	case ShareTypeLocation:
		return nil
	default:
		return ErrInvalidShareRequest
	}
}

func loadSharedUser(ctx context.Context, queryable persistence.QueryableDB, userID string) (string, string, string, error) {
	var id, nickname, avatarURL string
	if err := queryable.QueryRow(ctx, queryShareUserExists, userID).Scan(&id, &nickname, &avatarURL); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", ErrShareNotFound
		}
		return "", "", "", err
	}
	return id, nickname, avatarURL, nil
}

func loadSharedGroup(ctx context.Context, queryable persistence.QueryableDB, groupID string) (string, string, string, string, error) {
	var id, name, description, avatarURL string
	if err := queryable.QueryRow(ctx, queryShareGroupExists, groupID).Scan(&id, &name, &description, &avatarURL); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", "", ErrShareNotFound
		}
		return "", "", "", "", err
	}
	return id, name, description, avatarURL, nil
}

func loadSharedMoment(ctx context.Context, queryable persistence.QueryableDB, momentID string) (string, string, string, string, string, error) {
	var id, description, locationName, authorNickname, authorAvatarURL string
	if err := queryable.QueryRow(ctx, queryShareMomentExists, momentID).Scan(&id, &description, &locationName, &authorNickname, &authorAvatarURL); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", "", "", ErrShareNotFound
		}
		return "", "", "", "", "", err
	}
	return id, description, locationName, authorNickname, authorAvatarURL, nil
}

func (r *PostgresRepository) getShareOwnership(ctx context.Context, queryable persistence.QueryableDB, shareID string) (string, bool, error) {
	var ownerUserID string
	var isRevoked bool
	if err := queryable.QueryRow(ctx, queryGetShareByID, shareID).Scan(&shareID, &ownerUserID, &isRevoked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return ownerUserID, isRevoked, nil
}

func newShareCode() (string, error) {
	buf := make([]byte, shareCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	code := make([]byte, shareCodeLength)
	for i := range buf {
		code[i] = shareCodeAlphabet[int(buf[i])%len(shareCodeAlphabet)]
	}
	return string(code), nil
}

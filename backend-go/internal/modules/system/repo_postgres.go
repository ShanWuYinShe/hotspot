package system

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idgen"
	"time"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

var errAnnouncementNotFound = errors.New("announcement not found")

// healthPinger is the minimal subset of a dependency client needed to probe
// liveness from Health(). Both *pgxpool.Pool (via persistence.DB) and
// *redis.Client satisfy a compatible shape; we model them as functions so the
// system package doesn't import redis/pgx directly.
type healthPinger func(ctx context.Context) error

// healthProbeTimeout caps how long a single dependency ping may take, so a
// hung DB or redis doesn't stall the health endpoint (and whatever load
// balancer is polling it).
const healthProbeTimeout = 2 * time.Second

type PostgresRepository struct {
	factory     *persistence.Factory
	redisPinger healthPinger
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

// SetRedisPinger wires an optional redis liveness probe. Bootstrap calls this
// when a redis client is available; when nil (single-instance mode, or tests)
// the health endpoint skips the redis check rather than reporting degraded.
func (r *PostgresRepository) SetRedisPinger(p func(ctx context.Context) error) {
	r.redisPinger = p
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

func (r *PostgresRepository) GetMeta(ctx context.Context) ApiVersionMeta {
	return ApiVersionMeta{Version: "v1", Status: ApiVersionStatusActive}
}

// Health probes the dependencies the service actually depends on (Postgres and,
// when configured, Redis) and reports status per spec: code=0 covers healthy
// and degraded; code=10600 (CodeInternalUnavailable) means unhealthy. Postgres
// is a hard dependency — without it the service cannot serve any request, so a
// ping failure is unhealthy. Redis is a soft dependency — its absence degrades
// WebSocket fanout and resume but the service stays usable, so it is degraded.
// Each probe runs under a 2s timeout so a hung dependency can't stall the
// endpoint. The HTTP-layer response stays 200 so callers read the body to
// distinguish healthy/degraded/unhealthy.
func (r *PostgresRepository) Health(ctx context.Context) (int, string, *HealthStatus) {
	status := SystemHealthHealthy

	if r.factory != nil && r.factory.DB != nil {
		probeCtx, cancel := context.WithTimeout(ctx, healthProbeTimeout)
		err := r.factory.DB.Ping(probeCtx)
		cancel()
		if err != nil {
			observability.L().Warn("health check: postgres ping failed", "error", err)
			return int(httpx.CodeInternalUnavailable), "数据库不可用", nil
		}
	}
	if r.redisPinger != nil {
		probeCtx, cancel := context.WithTimeout(ctx, healthProbeTimeout)
		if err := r.redisPinger(probeCtx); err != nil {
			status = SystemHealthDegraded
			observability.L().Warn("health check: redis ping failed", "error", err)
		}
		cancel()
	}
	return 0, "", &HealthStatus{Status: status, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

func (r *PostgresRepository) GetAnnouncement(ctx context.Context, announcementID string) (AnnouncementRecord, bool, error) {
	queryable, err := r.queryable()
	if err != nil {
		return AnnouncementRecord{}, false, err
	}
	return r.getAnnouncementByID(ctx, queryable, announcementID)
}

func (r *PostgresRepository) CreateAnnouncement(ctx context.Context, input AnnouncementCreateInput) (AnnouncementRecord, error) {
	queryable, err := r.queryable()
	if err != nil {
		return AnnouncementRecord{}, err
	}
	now := time.Now().UTC()
	item := AnnouncementRecord{
		AnnouncementID: idgen.MustWithPrefix("ann_"),
		Title:          input.Title,
		Content:        input.Content,
		Severity:       input.Severity,
		ActionURL:      input.ActionURL,
		CreatedAt:      now,
		CreatedBy:      input.Actor,
		SendPush:       input.SendPush,
		ViewCount:      0,
	}
	var pushSentAt any
	if input.SendPush {
		item.PushSentAt = &now
		pushSentAt = now
	}
	if err := queryable.Exec(ctx, queryInsertAnnouncement,
		item.AnnouncementID,
		item.Title,
		item.Content,
		string(item.Severity),
		nullableAnnouncementString(item.ActionURL),
		item.SendPush,
		pushSentAt,
		item.ViewCount,
		item.CreatedBy.UserID,
		now,
	); err != nil {
		return AnnouncementRecord{}, err
	}
	return item, nil
}

func (r *PostgresRepository) UpdateAnnouncement(ctx context.Context, input AnnouncementUpdateInput) (AnnouncementRecord, error) {
	queryable, err := r.queryable()
	if err != nil {
		return AnnouncementRecord{}, err
	}
	current, found, err := r.getAnnouncementByID(ctx, queryable, input.AnnouncementID)
	if err != nil {
		return AnnouncementRecord{}, err
	}
	if !found || current.IsDeleted() {
		return AnnouncementRecord{}, errAnnouncementNotFound
	}
	if input.Title != nil {
		current.Title = *input.Title
	}
	if input.Content != nil {
		current.Content = *input.Content
	}
	if input.Severity != nil {
		current.Severity = *input.Severity
	}
	if input.ActionURL != nil {
		current.ActionURL = *input.ActionURL
	}
	if err := queryable.Exec(ctx, queryUpdateAnnouncement,
		current.AnnouncementID,
		current.Title,
		current.Content,
		string(current.Severity),
		nullableAnnouncementString(current.ActionURL),
		time.Now().UTC(),
	); err != nil {
		return AnnouncementRecord{}, err
	}
	return current, nil
}

func (r *PostgresRepository) DeleteAnnouncement(ctx context.Context, announcementID string) error {
	queryable, err := r.queryable()
	if err != nil {
		return err
	}
	current, found, err := r.getAnnouncementByID(ctx, queryable, announcementID)
	if err != nil {
		return err
	}
	if !found {
		return errAnnouncementNotFound
	}
	if current.IsDeleted() {
		return nil
	}
	return queryable.Exec(ctx, queryDeleteAnnouncement, announcementID, time.Now().UTC())
}

func nullableAnnouncementString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func scanAnnouncement(scanner interface{ Scan(...any) error }, includeDeleted bool) (AnnouncementRecord, error) {
	var item AnnouncementRecord
	var severity string
	var pushSentAt *time.Time
	if includeDeleted {
		var deletedAt *time.Time
		if err := scanner.Scan(
			&item.AnnouncementID,
			&item.Title,
			&item.Content,
			&severity,
			&item.ActionURL,
			&item.CreatedAt,
			&item.CreatedBy.UserID,
			&item.CreatedBy.Nickname,
			&item.CreatedBy.AvatarURL,
			&item.SendPush,
			&pushSentAt,
			&item.ViewCount,
			&deletedAt,
		); err != nil {
			return AnnouncementRecord{}, err
		}
		item.DeletedAt = deletedAt
	} else {
		if err := scanner.Scan(
			&item.AnnouncementID,
			&item.Title,
			&item.Content,
			&severity,
			&item.ActionURL,
			&item.CreatedAt,
			&item.CreatedBy.UserID,
			&item.CreatedBy.Nickname,
			&item.CreatedBy.AvatarURL,
			&item.SendPush,
			&pushSentAt,
			&item.ViewCount,
		); err != nil {
			return AnnouncementRecord{}, err
		}
	}
	item.Severity = AnnounceLevel(severity)
	if pushSentAt != nil {
		utc := pushSentAt.UTC()
		item.PushSentAt = &utc
	}
	item.CreatedAt = item.CreatedAt.UTC()
	return item, nil
}

func (r *PostgresRepository) getAnnouncementByID(ctx context.Context, queryable persistence.QueryableDB, announcementID string) (AnnouncementRecord, bool, error) {
	item, err := scanAnnouncement(queryable.QueryRow(ctx, queryAnnouncementByID, announcementID), true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AnnouncementRecord{}, false, nil
		}
		return AnnouncementRecord{}, false, err
	}
	return item, true, nil
}

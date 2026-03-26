package system

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"hotspot/backend-go/internal/platform/persistence"
)

var errAnnouncementNotFound = errors.New("announcement not found")

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

func (r *PostgresRepository) queryable() (persistence.QueryableDB, bool) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, false
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	return queryable, ok
}

func (r *PostgresRepository) GetMeta(ctx context.Context) ApiVersionMeta {
	if repo, ok := r.fallback.(interface {
		GetMeta(context.Context) ApiVersionMeta
	}); ok {
		return repo.GetMeta(ctx)
	}
	return ApiVersionMeta{Version: "v1", Status: ApiVersionStatusActive}
}

func (r *PostgresRepository) Health(ctx context.Context) (int, string, *HealthStatus) {
	if repo, ok := r.fallback.(interface {
		Health(context.Context) (int, string, *HealthStatus)
	}); ok {
		return repo.Health(ctx)
	}
	return 0, "", &HealthStatus{Status: SystemHealthHealthy, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

func (r *PostgresRepository) GetAnnouncement(ctx context.Context, announcementID string) (AnnouncementRecord, bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return r.fallback.GetAnnouncement(ctx, announcementID)
	}
	return r.getAnnouncementByID(ctx, queryable, announcementID)
}

func (r *PostgresRepository) CreateAnnouncement(ctx context.Context, input AnnouncementCreateInput) (AnnouncementRecord, error) {
	queryable, ok := r.queryable()
	if !ok {
		return r.fallback.CreateAnnouncement(ctx, input)
	}
	now := time.Now().UTC()
	item := AnnouncementRecord{
		AnnouncementID: fmt.Sprintf("ann_%d", now.UnixNano()),
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
	queryable, ok := r.queryable()
	if !ok {
		return r.fallback.UpdateAnnouncement(ctx, input)
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
	queryable, ok := r.queryable()
	if !ok {
		return r.fallback.DeleteAnnouncement(ctx, announcementID)
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
	item.Severity = AnnouncementSeverity(severity)
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

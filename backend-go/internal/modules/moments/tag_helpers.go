package moments

import (
	"context"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

func normalizeMomentTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		normalized := strings.TrimSpace(tag)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func normalizeMomentMentions(mentions []string) []string {
	if len(mentions) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(mentions))
	result := make([]string, 0, len(mentions))
	for _, mention := range mentions {
		normalized := strings.TrimSpace(mention)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func filterAllowedMomentTags(ctx context.Context, queryable persistence.Queryable, tags []string) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	rows, err := queryable.Query(ctx, queryFilterAllowedMomentTags, tags)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	allowed := make([]string, 0, len(tags))
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		allowed = append(allowed, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return allowed, nil
}

func insertMomentTags(ctx context.Context, queryable persistence.Queryable, momentID string, tags []string, createdAt time.Time) error {
	allowed, err := filterAllowedMomentTags(ctx, queryable, tags)
	if err != nil {
		return err
	}
	for _, tag := range allowed {
		if err := queryable.Exec(ctx, queryInsertMomentTagAssignment, momentID, tag, createdAt); err != nil {
			return err
		}
	}
	return nil
}

func replaceMomentMentions(ctx context.Context, queryable persistence.Queryable, momentID string, mentions []string, createdAt time.Time) error {
	if err := queryable.Exec(ctx, queryDeleteMomentMentionsByMomentID, momentID); err != nil {
		return err
	}
	for _, mention := range normalizeMomentMentions(mentions) {
		if err := queryable.Exec(ctx, queryInsertMomentMention, momentID, mention, createdAt); err != nil {
			return err
		}
	}
	return nil
}

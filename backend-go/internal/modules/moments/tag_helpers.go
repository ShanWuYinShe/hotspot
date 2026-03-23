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

func insertMomentTags(ctx context.Context, queryable persistence.QueryableDB, momentID string, tags []string, createdAt time.Time) error {
	for _, tag := range tags {
		if err := queryable.Exec(ctx, queryInsertMomentTagAssignment, momentID, tag, createdAt); err != nil {
			return err
		}
	}
	return nil
}

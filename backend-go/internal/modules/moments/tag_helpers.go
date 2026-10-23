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

// insertMomentTags 过滤被屏蔽标签后逐条落库，返回实际落库的 allowed 集合。
// blocked tag 口径收口于此（Create 与 Update 共用）：请求含被屏蔽标签时整体
// 返回 ErrBlockedTag（→ 10400，spec api_spec_moment.md create 错误码；update
// 的 tags 替换与 create 同族校验），由外层事务回滚，不允许「部分允许 tag
// 落库」的半屏蔽语义——此前 Update 路径静默丢弃被屏蔽 tag、Create 却整体
// 拒绝，同一校验两种口径。
func insertMomentTags(ctx context.Context, queryable persistence.Queryable, momentID string, tags []string, createdAt time.Time) ([]string, error) {
	allowed, err := filterAllowedMomentTags(ctx, queryable, tags)
	if err != nil {
		return nil, err
	}
	if len(allowed) != len(tags) {
		return nil, ErrBlockedTag
	}
	for _, tag := range allowed {
		if err := queryable.Exec(ctx, queryInsertMomentTagAssignment, momentID, tag, createdAt); err != nil {
			return nil, err
		}
	}
	return allowed, nil
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

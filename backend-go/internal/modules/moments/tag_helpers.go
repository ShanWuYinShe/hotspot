package moments

import (
	"context"
	"fmt"
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
	if len(allowed) == 0 {
		return nil, nil
	}
	// 单条多值 INSERT 替代逐 tag Exec：此前 ≤10 个标签 = 事务内 ≤10 次往返
	// （审查 P2-15，参照 insertMomentImages 的批量写法）。ON CONFLICT 与
	// 逐条写入的幂等语义一致（moment_tags 主键 (moment_id, tag_name)）。
	var sb strings.Builder
	sb.WriteString("INSERT INTO moment_tags (moment_id, tag_name, created_at) VALUES ")
	args := make([]any, 0, 2+len(allowed))
	args = append(args, momentID, createdAt)
	for i, tag := range allowed {
		if i > 0 {
			sb.WriteString(", ")
		}
		// $1=momentID, $2=createdAt，其后每个 tag 一个占位符
		sb.WriteString(fmt.Sprintf("($1, $%d, $2)", i+3))
		args = append(args, tag)
	}
	sb.WriteString(" ON CONFLICT (moment_id, tag_name) DO NOTHING")
	if err := queryable.Exec(ctx, sb.String(), args...); err != nil {
		return nil, err
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

// containsFold 是大小写不敏感子串判定：与 Postgres 的 ILIKE '%kw%' 语义对齐
// （契约测试发现内存实现此前用 strings.Contains 区分大小写，搜索命中集与真实库不同）。
func containsFold(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

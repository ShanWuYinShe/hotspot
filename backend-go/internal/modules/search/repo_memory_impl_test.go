package search

// 本文件是**测试专用占位实现，不保证语义等价**（审查 P2-26）：返回固定
// 捏造条目用于构造不依赖 DB 的单元测试，不做真实检索/分页/隐私过滤。
// module.go 的装配恒使用 NewPostgresRepository（DB 未配置时也无降级到本
// 实现的路径），故生产不可达。禁止在降级/生产路径引用。
import "context"

type InMemoryRepository struct{}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{}
}

func (r *InMemoryRepository) Suggest(_ context.Context, req SearchSuggestRequest) (SearchSuggestResponse, error) {
	items := []SearchSuggestItem{}
	if req.Prefix == "" {
		return SearchSuggestResponse{Items: items}, nil
	}
	allowed := normalizeSuggestTypes(req.Types)
	if allowed["tag"] {
		items = append(items, SearchSuggestItem{Type: "tag", Payload: map[string]any{"name": req.Prefix, "usage_count": 1}})
	}
	if allowed["user"] {
		items = append(items, SearchSuggestItem{Type: "user", Payload: map[string]any{"user_id": "user_" + req.Prefix, "nickname": req.Prefix}})
	}
	if allowed["group"] {
		items = append(items, SearchSuggestItem{Type: "group", Payload: map[string]any{"group_id": "group_" + req.Prefix, "name": req.Prefix, "members_count": 1}})
	}
	if allowed["moment"] {
		items = append(items, SearchSuggestItem{Type: "moment", Payload: map[string]any{"moment_id": "moment_" + req.Prefix, "description_preview": req.Prefix, "author": map[string]any{"user_id": "user_" + req.Prefix, "nickname": req.Prefix}}})
	}
	return SearchSuggestResponse{Items: items}, nil
}

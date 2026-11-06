package admin

import "context"

// fakeMomentTagStore 是 admin 消费的 moments 标签能力测试桩（moment_tags 归属
// moments 模块，admin 经接口消费，P0-4）。
type fakeMomentTagStore struct {
	refs    map[string]struct{}
	deleted []string
}

func newFakeMomentTagStore(refs ...string) *fakeMomentTagStore {
	store := &fakeMomentTagStore{refs: make(map[string]struct{}, len(refs))}
	for _, ref := range refs {
		store.refs[ref] = struct{}{}
	}
	return store
}

func (s *fakeMomentTagStore) HasTagAssignments(_ context.Context, tagName string) (bool, error) {
	_, ok := s.refs[tagName]
	return ok, nil
}

func (s *fakeMomentTagStore) DeleteTagAssignments(_ context.Context, tagName string) error {
	delete(s.refs, tagName)
	s.deleted = append(s.deleted, tagName)
	return nil
}

// newBackendMomentTagStore 让 fake 与 adminBackendStub.momentTags 共享同一 map，
// 便于断言「软屏蔽保留关联 / 硬删删除关联」。
func newBackendMomentTagStore(backend *adminBackendStub) *fakeMomentTagStore {
	if backend.momentTags == nil {
		backend.momentTags = make(map[string]struct{})
	}
	return &fakeMomentTagStore{refs: backend.momentTags}
}

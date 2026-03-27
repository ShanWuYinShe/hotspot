package groups

import (
	"context"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

type Service struct {
	repo Repository
}

func NewService() *Service {
	return NewServiceWithRepo(NewInMemoryRepository())
}

func NewServiceWithRepo(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateGroupRequest) (GroupItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Create(ctx, principal.UserID, req)
}

func (s *Service) OpenTheDoor(ctx context.Context, groupID string) (GroupItem, bool, error) {
	return s.repo.Get(ctx, groupID)
}

func (s *Service) Search(ctx context.Context, keyword string) (PaginatedGroups, error) {
	return s.repo.Search(ctx, keyword)
}

func (s *Service) Update(ctx context.Context, req UpdateGroupRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if group.MyState == nil || group.MyState.Role != GroupRoleOwner {
		return ErrPermissionDenied
	}
	return s.repo.Update(ctx, principal.UserID, req)
}

func (s *Service) Dismiss(ctx context.Context, req DismissGroupRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if group.MyState == nil || group.MyState.Role != GroupRoleOwner {
		return ErrPermissionDenied
	}
	return s.repo.Dismiss(ctx, principal.UserID)
}

func (s *Service) ListMembers(ctx context.Context, req GroupMembersRequest) ([]GroupMemberItem, string, error) {
	_, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", ErrGroupNotFound
	}
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListMembers(ctx, req.GroupID, req.Role, limit, req.Cursor)
}

func (s *Service) SearchMembers(ctx context.Context, req SearchGroupMembersRequest) ([]GroupMemberItem, string, error) {
	_, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", ErrGroupNotFound
	}
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.SearchMembers(ctx, req.GroupID, req.Keyword, req.Role, limit, req.Cursor)
}

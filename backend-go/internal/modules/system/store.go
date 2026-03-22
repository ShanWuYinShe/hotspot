package system

import (
	"context"
	"time"
)

type Store interface {
	GetMeta(context.Context) ApiVersionMeta
	Health(context.Context) (int, string, *HealthStatus)
	ListAnnouncements(context.Context) SystemAnnouncementList
}

type InMemoryStore struct{}

func NewInMemoryStore() *InMemoryStore { return &InMemoryStore{} }

func (s *InMemoryStore) GetMeta(context.Context) ApiVersionMeta {
	return ApiVersionMeta{
		Version: "v1",
		Status:  ApiVersionStatusActive,
	}
}

func (s *InMemoryStore) Health(context.Context) (int, string, *HealthStatus) {
	return 0, "", &HealthStatus{
		Status:    SystemHealthHealthy,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func (s *InMemoryStore) ListAnnouncements(context.Context) SystemAnnouncementList {
	return SystemAnnouncementList{Items: []SystemAnnouncementItem{}}
}

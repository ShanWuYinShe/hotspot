package realtime

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"hotspot/backend-go/internal/platform/actor"
)

// 审计 perf P2-2 收口：typing actor 30s 进程内缓存——同 user 的连续 typing 帧
// 只打一次 GetActorInfo；不同 user 各查一次；TTL 过期后重新查询；provider 出错
// 不缓存（下一帧重试成功）。
func TestResolveTypingActorCached(t *testing.T) {
	var calls atomic.Int64
	provider := actorCountingProvider{calls: &calls}
	h := NewHandler(nil, nil, nil)
	h.SetActorProvider(provider)

	first := h.resolveTypingActor(context.Background(), "user_1")
	second := h.resolveTypingActor(context.Background(), "user_1")
	third := h.resolveTypingActor(context.Background(), "user_2")

	if calls.Load() != 2 {
		t.Fatalf("GetActorInfo calls: got %d, want 2（同 user 命中缓存，不同 user 各查一次）", calls.Load())
	}
	if first["nickname"] != "nick-user_1" || second["nickname"] != "nick-user_1" {
		t.Fatalf("cached actor mismatch: %#v / %#v", first, second)
	}
	if third["user_id"] != "user_2" {
		t.Fatalf("second user actor mismatch: %#v", third)
	}
}

// 错误路径不落缓存：provider 报错时回退兜底 map，紧接着的成功查询重新打 provider。
func TestResolveTypingActorErrorNotCached(t *testing.T) {
	var calls atomic.Int64
	provider := actorErrThenOKProvider{calls: &calls}
	h := NewHandler(nil, nil, nil)
	h.SetActorProvider(provider)

	fallback := h.resolveTypingActor(context.Background(), "user_err")
	if fallback["nickname"] != "user_err" {
		t.Fatalf("error should fall back to userID nickname, got %#v", fallback)
	}

	ok := h.resolveTypingActor(context.Background(), "user_err")
	if ok["nickname"] != "nick-user_err" {
		t.Fatalf("retry after error should succeed, got %#v", ok)
	}
	if calls.Load() != 2 {
		t.Fatalf("error must not be cached: got %d calls, want 2", calls.Load())
	}
}

type actorCountingProvider struct {
	calls *atomic.Int64
}

func (p actorCountingProvider) GetActorInfo(_ context.Context, userID string) (actor.Info, error) {
	p.calls.Add(1)
	return actor.Info{UserID: userID, Nickname: "nick-" + userID}, nil
}

type actorErrThenOKProvider struct {
	calls *atomic.Int64
}

func (p actorErrThenOKProvider) GetActorInfo(_ context.Context, userID string) (actor.Info, error) {
	n := p.calls.Add(1)
	if n == 1 {
		return actor.Info{}, errors.New("db down")
	}
	return actor.Info{UserID: userID, Nickname: "nick-" + userID}, nil
}

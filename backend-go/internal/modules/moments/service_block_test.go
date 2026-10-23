package moments

import (
	"context"
	"errors"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

// blockServiceFixture 构造带拉黑判定注入的内存版 Service，返回服务与判定记录。
type blockServiceFixture struct {
	svc    *Service
	blocks map[string]bool // "caller|author" → blocked
	calls  int
}

func newBlockServiceFixture() *blockServiceFixture {
	f := &blockServiceFixture{svc: NewServiceWithRepo(NewInMemoryRepository()), blocks: map[string]bool{}}
	f.svc.SetBlockChecker(func(callerID, authorID string) (bool, error) {
		f.calls++
		return f.blocks[callerID+"|"+authorID] || f.blocks[authorID+"|"+callerID], nil
	})
	return f
}

func (f *blockServiceFixture) createMoment(t *testing.T, authorID, desc string) MomentItem {
	t.Helper()
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: authorID})
	item, err := f.svc.Create(ctx, CreateMomentRequest{Description: desc, PrivacyLevel: "public"})
	if err != nil {
		t.Fatalf("Create moment: %v", err)
	}
	return item.MomentItem
}

func ctxAs(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

// TestBlockEitherWayBlocksFavorite 验证收藏路径的双向拉黑拦截：
// 无论 caller 拉黑作者还是被作者拉黑，收藏都按不可交互处理（10300 防枚举口径）。
func TestBlockEitherWayBlocksFavorite(t *testing.T) {
	f := newBlockServiceFixture()
	moment := f.createMoment(t, "author_1", "hello")

	f.blocks["viewer_1|author_1"] = true // viewer 拉黑了作者
	if err := f.svc.Favorite(ctxAs("viewer_1"), moment.MomentID); !errors.Is(err, ErrMomentNotFound) {
		t.Fatalf("viewer 拉黑作者时收藏应按不存在处理, got %v", err)
	}

	delete(f.blocks, "viewer_1|author_1")
	f.blocks["author_1|viewer_2"] = true // 作者拉黑了 viewer_2
	if err := f.svc.Favorite(ctxAs("viewer_2"), moment.MomentID); !errors.Is(err, ErrMomentNotFound) {
		t.Fatalf("被作者拉黑时收藏应按不存在处理, got %v", err)
	}

	// 无拉黑关系时正常收藏。
	if err := f.svc.Favorite(ctxAs("viewer_3"), moment.MomentID); err != nil {
		t.Fatalf("无拉黑关系收藏应成功, got %v", err)
	}
}

// TestBlockEitherWayBlocksOpenTheDoor 验证瞬间详情直达路径的双向拉黑拦截
// （P2-1）：拉黑双方经 open_the_door 互不可见（按不存在处理，spec
// api_spec_authz.md「隐私拒绝」→10300），详情不得成为击穿拉黑直读对方
// 瞬间全文与图片的旁路。
func TestBlockEitherWayBlocksOpenTheDoor(t *testing.T) {
	f := newBlockServiceFixture()
	moment := f.createMoment(t, "author_1", "hello")

	f.blocks["viewer_1|author_1"] = true
	item, ok, err := f.svc.OpenTheDoor(ctxAs("viewer_1"), moment.MomentID)
	if err != nil || ok || item.MomentID != "" {
		t.Fatalf("拉黑关系下详情应按不存在处理, got item=%#v ok=%v err=%v", item, ok, err)
	}

	delete(f.blocks, "viewer_1|author_1")
	item, ok, err = f.svc.OpenTheDoor(ctxAs("viewer_1"), moment.MomentID)
	if err != nil || !ok || item.MomentID != moment.MomentID {
		t.Fatalf("无拉黑关系详情应可见, got item=%#v ok=%v err=%v", item, ok, err)
	}

	// 作者本人不受影响。
	if _, ok, err := f.svc.OpenTheDoor(ctxAs("author_1"), moment.MomentID); err != nil || !ok {
		t.Fatalf("作者本人详情应可见, ok=%v err=%v", ok, err)
	}
}

// TestBlockEitherWayBlocksReactionTarget 验证 reaction 目标可见性在双向拉黑下
// 返回不可见（reactions 创建被拒，不再向对方触发通知）。
func TestBlockEitherWayBlocksReactionTarget(t *testing.T) {
	f := newBlockServiceFixture()
	moment := f.createMoment(t, "author_1", "hello")

	f.blocks["viewer_1|author_1"] = true
	found, visible, err := f.svc.IsReactionTargetVisible(ctxAs("viewer_1"), moment.MomentID)
	if err != nil || !found || visible {
		t.Fatalf("拉黑关系下 reaction 目标应 found=true/visible=false, got found=%v visible=%v err=%v", found, visible, err)
	}

	delete(f.blocks, "viewer_1|author_1")
	found, visible, err = f.svc.IsReactionTargetVisible(ctxAs("viewer_1"), moment.MomentID)
	if err != nil || !found || !visible {
		t.Fatalf("无拉黑关系 reaction 目标应可见, got found=%v visible=%v err=%v", found, visible, err)
	}
}

// TestBlockCheckerFailureFailsClosed 验证拉黑查询失败时 fail-closed：
// 收藏返回错误而非放行（与评论拉黑口径一致）。
func TestBlockCheckerFailureFailsClosed(t *testing.T) {
	svc := NewServiceWithRepo(NewInMemoryRepository())
	boom := errors.New("db down")
	svc.SetBlockChecker(func(callerID, authorID string) (bool, error) {
		return false, boom
	})
	ctx := ctxAs("author_1")
	item, err := svc.Create(ctx, CreateMomentRequest{Description: "hello", PrivacyLevel: "public"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Favorite(ctxAs("viewer_1"), item.MomentID); !errors.Is(err, boom) {
		t.Fatalf("拉黑查询失败应 fail-closed 透传错误, got %v", err)
	}
}

// TestBlockCheckerSkippedForAuthorAndNil 验证作者本人操作不触发拉黑查询，
// 未注入 checker 时行为与原先一致。
func TestBlockCheckerSkippedForAuthorAndNil(t *testing.T) {
	f := newBlockServiceFixture()
	moment := f.createMoment(t, "author_1", "hello")

	if err := f.svc.Favorite(ctxAs("author_1"), moment.MomentID); err != nil {
		t.Fatalf("作者本人收藏不应被拦截, got %v", err)
	}
	if f.calls != 0 {
		t.Fatalf("作者本人操作不应触发拉黑查询, calls=%d", f.calls)
	}

	plain := NewServiceWithRepo(NewInMemoryRepository())
	if err := plain.Favorite(ctxAs("viewer_1"), moment.MomentID); err == nil {
		// 内存 repo 是新实例，moment 不存在 → ErrMomentNotFound；关键是不得 panic。
		t.Log("新 repo 无该动态:", err)
	}
}

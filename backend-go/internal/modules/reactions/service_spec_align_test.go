package reactions

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/realtime"
)

// stubActorProvider resolves actor info for buildActor tests.
type stubActorProvider struct {
	infos map[string]actor.Info
	err   error
}

func (s stubActorProvider) GetActorInfo(_ context.Context, userID string) (actor.Info, error) {
	if s.err != nil {
		return actor.Info{}, s.err
	}
	if info, ok := s.infos[userID]; ok {
		return info, nil
	}
	return actor.Info{UserID: userID, Nickname: userID}, nil
}

// --- Set 对 message 目标:统一 reaction_added 通知 + ReactionAdded WS 送达作者 ---

func TestSetReactMessageTargetNotifiesAndBroadcasts(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	defer realtime.ResetHub()
	ownerSink := &realtimeSinkStub{}
	hub.RegisterUser("owner_msg", "sess_owner", ownerSink)

	notif := &recordingNotifier{}
	msgProv := stubTargetProvider{kind: ReactTargetMessage, exists: map[string]bool{"msg1": true}, owners: map[string]string{"msg1": "owner_msg"}}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{msgProv}, notif, stubActorProvider{infos: map[string]actor.Info{
		"u_bob": {UserID: "u_bob", Nickname: "Bob", AvatarURL: "https://avatar.example/bob.png"},
	}})

	if err := svc.Set(ctxWithUser("u_bob"), SetReactionRequest{TargetType: ReactTargetMessage, TargetID: "msg1", Emoji: "👍", Active: true}); err != nil {
		t.Fatalf("Set message: %v", err)
	}

	// 通知分派:message 目标同样走统一 reaction_added(契约移除 message_liked 等旧类型)。
	notif.mu.Lock()
	if len(notif.reactionAddedCalls) != 1 {
		notif.mu.Unlock()
		t.Fatalf("expected one reaction_added notification, got %+v", notif.reactionAddedCalls)
	}
	call := notif.reactionAddedCalls[0]
	notif.mu.Unlock()
	if call.targetUserID != "owner_msg" || call.targetType != "message" || call.targetID != "msg1" || call.emoji != "👍" {
		t.Fatalf("unexpected notification call: %+v", call)
	}

	// WS ReactionAdded 送达资源作者,payload 完整。
	if len(ownerSink.events) != 1 || ownerSink.events[0].Type != EventTypeReactionAdded {
		t.Fatalf("expected ReactionAdded on owner sink, got %+v", ownerSink.events)
	}
	payload, ok := ownerSink.events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", ownerSink.events[0].Payload)
	}
	if payload["target_type"] != "message" || payload["target_id"] != "msg1" || payload["emoji"] != "👍" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	// 非自反应:payload 必须携带 notification_id(客户端深链用)。
	if payload["notify_id"] != "notif_1" {
		t.Fatalf("expected notification_id=notif_1, got %+v", payload)
	}
	actorInfo, ok := payload["actor"].(map[string]any)
	if !ok {
		t.Fatalf("expected actor map, got %T", payload["actor"])
	}
	if actorInfo["user_id"] != "u_bob" || actorInfo["nickname"] != "Bob" || actorInfo["avatar_url"] != "https://avatar.example/bob.png" {
		t.Fatalf("unexpected actor: %+v", actorInfo)
	}
}

func TestSetReactSelfReactionSkipsNotificationAndBroadcast(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	defer realtime.ResetHub()
	ownerSink := &realtimeSinkStub{}
	hub.RegisterUser("u_bob", "sess_bob", ownerSink)

	notif := &recordingNotifier{}
	prov := stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}, owners: map[string]string{"m1": "u_bob"}}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{prov}, notif, nil)

	if err := svc.Set(ctxWithUser("u_bob"), SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: true}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if len(ownerSink.events) != 0 {
		t.Fatalf("self-reaction must not broadcast ReactionAdded, got %+v", ownerSink.events)
	}
	notif.mu.Lock()
	defer notif.mu.Unlock()
	if len(notif.reactionAddedCalls) != 0 {
		t.Fatalf("self-reaction must not notify owner, got %+v", notif.reactionAddedCalls)
	}
}

// --- resolveTarget 分支:三类目标存在/不存在 ---

func TestResolveTargetBranches(t *testing.T) {
	momentProv := stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}, owners: map[string]string{"m1": "owner_m"}}
	commentProv := stubTargetProvider{kind: ReactTargetComment, exists: map[string]bool{}, owners: map[string]string{}}
	messageProv := stubTargetProvider{kind: ReactTargetMessage, exists: map[string]bool{"msg1": true}, owners: map[string]string{"msg1": "owner_msg"}}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{momentProv, commentProv, messageProv}, nil, nil)

	// moment 存在 → found + owner
	ctx2, found, err := svc.resolveTarget(context.Background(), ReactTargetMoment, "m1")
	if err != nil || !found || ctx2.OwnerUserID != "owner_m" {
		t.Fatalf("moment resolve: found=%v owner=%q err=%v", found, ctx2.OwnerUserID, err)
	}
	// comment 不存在 → found=false 无错误
	if _, found, err := svc.resolveTarget(context.Background(), ReactTargetComment, "c_missing"); err != nil || found {
		t.Fatalf("comment missing: found=%v err=%v", found, err)
	}
	// message 存在 → found + owner
	ctx2, found, err = svc.resolveTarget(context.Background(), ReactTargetMessage, "msg1")
	if err != nil || !found || ctx2.OwnerUserID != "owner_msg" {
		t.Fatalf("message resolve: found=%v owner=%q err=%v", found, ctx2.OwnerUserID, err)
	}
	// 未注册 kind 的 provider(moment 存在,查 comment)→ 视为存在(降级)
	onlyMoment := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{momentProv}, nil, nil)
	ctx2, found, err = onlyMoment.resolveTarget(context.Background(), ReactTargetComment, "c1")
	if err != nil || !found {
		t.Fatalf("unregistered kind: found=%v err=%v", found, err)
	}
	// providers == nil → 视为存在但无 owner
	svc2 := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	ctx2, found, err = svc2.resolveTarget(context.Background(), ReactTargetMoment, "x")
	if err != nil || !found || ctx2.OwnerUserID != "" {
		t.Fatalf("nil providers: found=%v owner=%q err=%v", found, ctx2.OwnerUserID, err)
	}
}

// --- emoji 白名单(spec api_spec_reaction.md:emoji 不在白名单返回 10000) ---

func TestSetRejectsEmojiOutsideWhitelist(t *testing.T) {
	source := appconfig.NewStaticSource(config.Config{}) // 默认白名单:👍❤️😂😮😢🙏
	prov := stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}, owners: map[string]string{"m1": "owner"}}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{prov}, nil, nil)
	svc.SetAppConfig(source)

	err := svc.Set(ctxWithUser("u_bob"), SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "💩", Active: true})
	if !errors.Is(err, ErrInvalidEmoji) {
		t.Fatalf("expected ErrInvalidEmoji for emoji outside whitelist, got %v", err)
	}
	// 白名单 emoji 正常通过
	if err := svc.Set(ctxWithUser("u_bob"), SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: true}); err != nil {
		t.Fatalf("whitelisted emoji must succeed: %v", err)
	}
	// unreact 是幂等移除,不受白名单限制
	if err := svc.Set(ctxWithUser("u_bob"), SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "💩", Active: false}); err != nil {
		t.Fatalf("unreact must not be whitelist-gated: %v", err)
	}
}

func TestHandlerSetRejectsEmojiOutsideWhitelist(t *testing.T) {
	source := appconfig.NewStaticSource(config.Config{})
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	svc.SetAppConfig(source)
	h := NewHandler(svc)

	rr := do(t, h.Set, "u", `{"target_type":"moment","target_id":"m1","emoji":"💩","active":true}`)
	if code := apiCode(t, rr.Body.String()); code != 10000 {
		t.Fatalf("expected 10000 for emoji outside whitelist, got %d", code)
	}
}

// --- buildActor:provider 失败回退 ---

func TestBuildActorFallsBackWhenProviderFails(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, stubActorProvider{err: errors.New("boom")})
	a := svc.buildActor(context.Background(), "u_x")
	if a["user_id"] != "u_x" || a["nickname"] != "u_x" {
		t.Fatalf("expected fallback actor {user_id:u_x,nickname:u_x}, got %+v", a)
	}
}

// --- ListReactedTargets(backs users/list_reacted_moments) ---

func TestListReactedTargetsBasic(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	if _, err := svc.repo.React(context.Background(), ReactTargetMoment, "m1", "u1", "👍"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.repo.React(context.Background(), ReactTargetMoment, "m2", "u1", "❤️"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.repo.React(context.Background(), ReactTargetMoment, "m3", "u2", "👍"); err != nil {
		t.Fatal(err) // 其他用户的反应不计入
	}
	if _, err := svc.repo.React(context.Background(), ReactTargetComment, "c1", "u1", "👍"); err != nil {
		t.Fatal(err) // 其他目标类型不计入
	}

	ids, next, err := svc.ListReactedTargets(ctxWithUser("u1"), ReactTargetMoment, 20, "")
	if err != nil {
		t.Fatalf("ListReactedTargets: %v", err)
	}
	if len(ids) != 2 || next != "" {
		t.Fatalf("expected 2 ids and no cursor, got %v next=%q", ids, next)
	}
	if ids[0] != "m1" && ids[0] != "m2" {
		t.Fatalf("unexpected ids: %v", ids)
	}

	// 分页:limit=1 返回一个 + next_cursor
	ids, next, err = svc.ListReactedTargets(ctxWithUser("u1"), ReactTargetMoment, 1, "")
	if err != nil {
		t.Fatalf("ListReactedTargets page: %v", err)
	}
	if len(ids) != 1 || next == "" {
		t.Fatalf("expected 1 id + cursor, got %v next=%q", ids, next)
	}
	// 第二页
	ids, next, err = svc.ListReactedTargets(ctxWithUser("u1"), ReactTargetMoment, 1, next)
	if err != nil {
		t.Fatalf("ListReactedTargets page2: %v", err)
	}
	if len(ids) != 1 || next != "" {
		t.Fatalf("expected 1 id and no cursor on last page, got %v next=%q", ids, next)
	}
}

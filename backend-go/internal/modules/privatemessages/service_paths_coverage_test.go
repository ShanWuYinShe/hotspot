package privatemessages

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
)

// 本文件补齐 service 层的错误路径/权限分支/配额边界：媒体与回复读取失败、
// share 校验、配额咨询锁与计数失败、陌生人配额批量化失败回退、以及 admin
// 撤回与账户注销级联的注入失败分支。

// covServiceRepo 在既有 stubMsgRepo 之上按需注入单方法返回值（复用其默认语义）。
type covServiceRepo struct {
	*stubMsgRepo
	mediaResult map[string]any
	mediaFound  bool
	mediaErr    error
	replyResult map[string]any
	replyFound  bool
	replyErr    error
	lockErr     error

	participantsSender   string
	participantsReceiver string
	participantsFound    bool
	participantsErr      error
	recallAdminHit       bool
	recallAdminErr       error
}

func (r *covServiceRepo) GetMediaForMessage(context.Context, string, string) (map[string]any, bool, error) {
	return r.mediaResult, r.mediaFound, r.mediaErr
}

func (r *covServiceRepo) GetMessageForReply(context.Context, string, string) (map[string]any, bool, error) {
	return r.replyResult, r.replyFound, r.replyErr
}

func (r *covServiceRepo) LockStrangerQuota(context.Context, string, string) error {
	if r.lockErr != nil {
		r.ops = append(r.ops, "lock_failed")
		return r.lockErr
	}
	return r.stubMsgRepo.LockStrangerQuota(context.Background(), "", "")
}

func (r *covServiceRepo) MessageParticipants(context.Context, string) (string, string, bool, error) {
	return r.participantsSender, r.participantsReceiver, r.participantsFound, r.participantsErr
}

func (r *covServiceRepo) RecallAsAdmin(context.Context, string) (bool, error) {
	return r.recallAdminHit, r.recallAdminErr
}

// covUserReader 在既有 stubUserReader 之上注入黑名单/互关批量查询错误。
type covUserReader struct {
	*stubUserReader
	blockErr             error
	relationshipBatchErr error
}

func (u *covUserReader) BlockStateBetween(context.Context, string, string) (bool, bool, error) {
	if u.blockErr != nil {
		return false, false, u.blockErr
	}
	return u.stubUserReader.BlockStateBetween(context.Background(), "", "")
}

func (u *covUserReader) RelationshipBatch(ctx context.Context, userID string, targetUserIDs []string) (map[string]users.Relationship, error) {
	if u.relationshipBatchErr != nil {
		return nil, u.relationshipBatchErr
	}
	return u.stubUserReader.RelationshipBatch(ctx, userID, targetUserIDs)
}

func covAllowedUsers() *stubUserReader {
	return &stubUserReader{
		found:    true,
		item:     users.UserItem{Relation: &users.Relationship{}},
		settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	}
}

// covShareValidatorStub 是 share_code 校验器的可控替身。
type covShareValidatorStub struct{ err error }

func (v covShareValidatorStub) ValidateShareCode(context.Context, string) error { return v.err }

func TestServiceSendMediaCheckErrorPropagates(t *testing.T) {
	boom := errors.New("media db down")
	repo := &covServiceRepo{stubMsgRepo: &stubMsgRepo{}, mediaErr: boom}
	svc := NewServiceWithDeps(repo, covAllowedUsers())

	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "image", FileID: "file_1"})
	if !errors.Is(err, boom) {
		t.Fatalf("expected media lookup error to propagate, got %v", err)
	}
}

func TestServiceSendShareValidationFailure(t *testing.T) {
	repo := &covServiceRepo{stubMsgRepo: &stubMsgRepo{sendResult: MessageSentResult{MessageID: "pm1"}}, mediaFound: true}
	svc := NewServiceWithDeps(repo, covAllowedUsers())
	svc.SetShareValidator(covShareValidatorStub{err: errors.New("share revoked")})

	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "share", ShareCode: "sc_1"})
	if !errors.Is(err, ErrInvalidShareCode) {
		t.Fatalf("expected ErrInvalidShareCode, got %v", err)
	}
	if len(repo.ops) != 0 {
		t.Fatalf("invalid share must abort before repo write, got ops %v", repo.ops)
	}
}

func TestServiceSendValidShareCodePasses(t *testing.T) {
	repo := &covServiceRepo{stubMsgRepo: &stubMsgRepo{sendResult: MessageSentResult{MessageID: "pm1"}}}
	svc := NewServiceWithDeps(repo, covAllowedUsers())
	svc.SetShareValidator(covShareValidatorStub{})

	if _, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "share", ShareCode: "sc_1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceSendQuotaLockErrorAborts(t *testing.T) {
	boom := errors.New("advisory lock failed")
	repo := &covServiceRepo{stubMsgRepo: &stubMsgRepo{sendResult: MessageSentResult{MessageID: "pm1"}}, lockErr: boom}
	svc := NewServiceWithDeps(repo, covAllowedUsers())

	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "hi"})
	if !errors.Is(err, boom) {
		t.Fatalf("expected lock error to propagate, got %v", err)
	}
	if repo.sendResult.MessageID == "" || len(repo.ops) != 1 || repo.ops[0] != "lock_failed" {
		t.Fatalf("lock failure must abort before send, got ops %v", repo.ops)
	}
}

func TestServiceSendQuotaCountErrorAborts(t *testing.T) {
	boom := errors.New("count failed")
	repo := &covServiceRepo{stubMsgRepo: &stubMsgRepo{countErr: boom, sendResult: MessageSentResult{MessageID: "pm1"}}}
	svc := NewServiceWithDeps(repo, covAllowedUsers())
	svc.SetAppConfig(quotaSource{quota: 5})

	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "hi"})
	if !errors.Is(err, boom) {
		t.Fatalf("expected quota count error to propagate, got %v", err)
	}
}

// 回复/媒体在发送成功后回填进 WS 负载（真实数据源命中时）。
func TestServiceSendBuildsReplyAndMediaPayload(t *testing.T) {
	repo := &covServiceRepo{
		stubMsgRepo: &stubMsgRepo{sendResult: MessageSentResult{MessageID: "pm1", SentAt: "2026-03-30T10:00:00Z"}},
		mediaResult: map[string]any{"url": "media/storage/file_1.png"},
		mediaFound:  true,
		replyResult: map[string]any{"message_id": "pm_reply"},
		replyFound:  true,
	}
	svc := NewServiceWithDeps(repo, covAllowedUsers())

	if _, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{
		TargetUserID:     "u2",
		Type:             "image",
		FileID:           "file_1",
		ReplyToMessageID: "pm_reply",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// buildReplyTo / buildMedia 是发送负载的唯一来源：直接校验其返回。
	if reply := svc.buildReplyTo(context.Background(), "pm_reply", "u1"); reply == nil || reply["message_id"] != "pm_reply" {
		t.Fatalf("unexpected reply payload: %#v", reply)
	}
	if media := svc.buildMedia(context.Background(), "file_1", "u1"); media == nil || media["url"] != "media/storage/file_1.png" {
		t.Fatalf("unexpected media payload: %#v", media)
	}
}

func TestServiceRunInTxUsesInjectedRunner(t *testing.T) {
	repo := &stubMsgRepo{sendResult: MessageSentResult{MessageID: "pm1"}}
	svc := NewServiceWithDeps(repo, covAllowedUsers())

	ran := false
	svc.SetTxRunner(func(ctx context.Context, fn func(context.Context) error) error {
		ran = true
		return fn(ctx)
	})
	if _, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "hi"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ran {
		t.Fatal("injected tx runner must be used")
	}
}

func TestServiceRecallMessageByIDAsAdminBranches(t *testing.T) {
	t.Run("participants error", func(t *testing.T) {
		boom := errors.New("participants query failed")
		repo := &covServiceRepo{stubMsgRepo: &stubMsgRepo{}, participantsErr: boom}
		hit, err := NewServiceWithRepo(repo).RecallMessageByIDAsAdmin(errorCtx(), "admin_1", "pm_1")
		if hit || !errors.Is(err, boom) {
			t.Fatalf("hit=%v err=%v, want false/participants error", hit, err)
		}
	})
	t.Run("message not found", func(t *testing.T) {
		repo := &covServiceRepo{stubMsgRepo: &stubMsgRepo{}, participantsFound: false}
		hit, err := NewServiceWithRepo(repo).RecallMessageByIDAsAdmin(errorCtx(), "admin_1", "pm_missing")
		if hit || err != nil {
			t.Fatalf("hit=%v err=%v, want false/nil", hit, err)
		}
	})
	t.Run("recall as admin error", func(t *testing.T) {
		boom := errors.New("recall failed")
		repo := &covServiceRepo{stubMsgRepo: &stubMsgRepo{}, participantsFound: true, recallAdminErr: boom}
		hit, err := NewServiceWithRepo(repo).RecallMessageByIDAsAdmin(errorCtx(), "admin_1", "pm_1")
		if hit || !errors.Is(err, boom) {
			t.Fatalf("hit=%v err=%v, want false/recall error", hit, err)
		}
	})
	t.Run("already recalled", func(t *testing.T) {
		repo := &covServiceRepo{stubMsgRepo: &stubMsgRepo{}, participantsFound: true, recallAdminHit: false}
		hit, err := NewServiceWithRepo(repo).RecallMessageByIDAsAdmin(errorCtx(), "admin_1", "pm_1")
		if hit || err != nil {
			t.Fatalf("hit=%v err=%v, want false/nil for already recalled", hit, err)
		}
	})
}

// errorCtx 返回带 principal 的普通 ctx（admin 撤回不需要 hub）。
func errorCtx() context.Context { return msgCtx("admin_1") }

func TestServiceEnsureCanSendBranchCoverage(t *testing.T) {
	ctx := msgCtx("u1")

	t.Run("no user reader wired", func(t *testing.T) {
		svc := NewServiceWithDeps(&stubMsgRepo{}, nil)
		quota, err := svc.ensureCanSend(ctx, "u1", "u2")
		if quota || err != nil {
			t.Fatalf("quota=%v err=%v, want false/nil when users unwired", quota, err)
		}
	})
	t.Run("target lookup error", func(t *testing.T) {
		boom := errors.New("users db down")
		svc := NewServiceWithDeps(&stubMsgRepo{}, &stubUserReader{userErr: boom})
		if _, err := svc.ensureCanSend(ctx, "u1", "u2"); !errors.Is(err, boom) {
			t.Fatalf("expected lookup error, got %v", err)
		}
	})
	t.Run("block state error", func(t *testing.T) {
		boom := errors.New("block query failed")
		svc := NewServiceWithDeps(&stubMsgRepo{}, &covUserReader{stubUserReader: covAllowedUsers(), blockErr: boom})
		if _, err := svc.ensureCanSend(ctx, "u1", "u2"); !errors.Is(err, boom) {
			t.Fatalf("expected block state error, got %v", err)
		}
	})
	t.Run("settings error", func(t *testing.T) {
		boom := errors.New("settings query failed")
		svc := NewServiceWithDeps(&stubMsgRepo{}, &stubUserReader{found: true, settingsErr: boom})
		if _, err := svc.ensureCanSend(ctx, "u1", "u2"); !errors.Is(err, boom) {
			t.Fatalf("expected settings error, got %v", err)
		}
	})
	t.Run("relationship error", func(t *testing.T) {
		boom := errors.New("relationship query failed")
		svc := NewServiceWithDeps(&stubMsgRepo{}, &stubUserReader{found: true, settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone}, relationErr: boom})
		if _, err := svc.ensureCanSend(ctx, "u1", "u2"); !errors.Is(err, boom) {
			t.Fatalf("expected relationship error, got %v", err)
		}
	})
	t.Run("friends permission requires mutual follow", func(t *testing.T) {
		svc := NewServiceWithDeps(&stubMsgRepo{}, &stubUserReader{
			found:    true,
			settings: users.UserSettings{AllowPrivateChat: users.ChatPermFriends},
		})
		_, err := svc.ensureCanSend(ctx, "u1", "u2")
		assertForbiddenReason(t, err, httpx.ReasonPrivacySetting)
	})
	t.Run("friends permission allows mutual follow", func(t *testing.T) {
		svc := NewServiceWithDeps(&stubMsgRepo{}, &stubUserReader{
			found:     true,
			settings:  users.UserSettings{AllowPrivateChat: users.ChatPermFriends},
			relations: map[string]users.Relationship{"u2": {IsFollowing: true, IsFollowedBy: true}},
		})
		quota, err := svc.ensureCanSend(ctx, "u1", "u2")
		if err != nil || quota {
			t.Fatalf("mutual follow under friends-only must pass without stranger quota, got quota=%v err=%v", quota, err)
		}
	})
}

func TestServiceEnforceStrangerQuotaBranchCoverage(t *testing.T) {
	ctx := msgCtx("u1")

	t.Run("negative quota treated as zero", func(t *testing.T) {
		svc := NewServiceWithDeps(&stubMsgRepo{}, covAllowedUsers())
		svc.SetAppConfig(quotaSource{quota: -3})
		assertForbiddenReason(t, svc.enforceStrangerQuota(ctx, "u1"), httpx.ReasonStrangerQuotaExceeded)
	})
	t.Run("count error propagates", func(t *testing.T) {
		boom := errors.New("count failed")
		svc := NewServiceWithDeps(&stubMsgRepo{countErr: boom}, covAllowedUsers())
		svc.SetAppConfig(quotaSource{quota: 5})
		if err := svc.enforceStrangerQuota(ctx, "u1"); !errors.Is(err, boom) {
			t.Fatalf("expected count error, got %v", err)
		}
	})
	t.Run("self target excluded", func(t *testing.T) {
		svc := NewServiceWithDeps(&stubMsgRepo{counts: map[string]int{"u1": 9, "u2": 1}}, covAllowedUsers())
		svc.SetAppConfig(quotaSource{quota: 5})
		if err := svc.enforceStrangerQuota(ctx, "u1"); err != nil {
			t.Fatalf("self target must not consume stranger quota, got %v", err)
		}
	})
	t.Run("batch relationship failure falls back to per-target", func(t *testing.T) {
		reader := &covUserReader{
			stubUserReader: &stubUserReader{
				found:     true,
				settings:  users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
				relations: map[string]users.Relationship{"u2": {}},
			},
			relationshipBatchErr: errors.New("batch failed"),
		}
		svc := NewServiceWithDeps(&stubMsgRepo{counts: map[string]int{"u2": 5}}, reader)
		svc.SetAppConfig(quotaSource{quota: 5})
		assertForbiddenReason(t, svc.enforceStrangerQuota(ctx, "u1"), httpx.ReasonStrangerQuotaExceeded)
	})
	t.Run("batch relationship failure fallback keeps mutual targets out", func(t *testing.T) {
		reader := &covUserReader{
			stubUserReader: &stubUserReader{
				found:     true,
				settings:  users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
				relations: map[string]users.Relationship{"u2": {IsFollowing: true, IsFollowedBy: true}},
			},
			relationshipBatchErr: errors.New("batch failed"),
		}
		svc := NewServiceWithDeps(&stubMsgRepo{counts: map[string]int{"u2": 5}}, reader)
		svc.SetAppConfig(quotaSource{quota: 5})
		if err := svc.enforceStrangerQuota(ctx, "u1"); err != nil {
			t.Fatalf("mutual follows must not count toward stranger quota, got %v", err)
		}
	})
}

func TestServiceResolveUserInfoIncludesAvatar(t *testing.T) {
	avatar := "media/avatar/u1.png"
	reader := &stubUserReader{
		found: true,
		item:  users.UserItem{User: users.UserInfo{UserID: "u1", Nickname: "Alice", AvatarURL: &avatar}},
	}
	svc := NewServiceWithDeps(&stubMsgRepo{}, reader)

	info := svc.resolveUserInfo(context.Background(), "u1")
	if info["nickname"] != "Alice" || info["avatar_url"] != avatar {
		t.Fatalf("unexpected user info payload: %#v", info)
	}

	// 未接线 users 时只返回 user_id（降级不报错）。
	if info := NewServiceWithDeps(&stubMsgRepo{}, nil).resolveUserInfo(context.Background(), "u1"); info["user_id"] != "u1" || len(info) != 1 {
		t.Fatalf("unexpected degraded user info: %#v", info)
	}
}

// --- user_purge ---

func TestServicePurgeUserDataNonPurgerRepoIsNoop(t *testing.T) {
	svc := NewServiceWithRepo(NewInMemoryRepository())
	if err := svc.PurgeUserData(context.Background(), "u1"); err != nil {
		t.Fatalf("non-purger repo must be a no-op, got %v", err)
	}
}

func TestPostgresPurgeUserDataFailsClosed(t *testing.T) {
	repo := NewPostgresRepository(&persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     privateMessageNonQueryableDB{},
	})
	if err := repo.PurgeUserData(context.Background(), "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

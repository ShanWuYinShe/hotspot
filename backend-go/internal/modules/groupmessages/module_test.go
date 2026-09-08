package groupmessages

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/go-chi/chi/v5"
)

func TestGroupMessagesModuleMount(t *testing.T) {
	fac := persistence.NewFactory(config.Config{})
	mod := NewModule(fac, nil, nil, nil)
	if mod == nil || mod.Service == nil || mod.handler == nil {
		t.Fatal("expected non-nil groupmessages module")
	}

	r := chi.NewRouter()
	mod.Mount(r)
}

func TestInMemoryRepositoryGetMessageReportSnapshot(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()

	// 1. 查找不存在的消息
	_, found, err := repo.GetMessageReportSnapshot(ctx, "gm_not_exist")
	if err != nil || found {
		t.Fatalf("expected not found for non-existent message, got found=%v, err=%v", found, err)
	}

	// 2. 发送普通文本消息并读取快照
	res, err := repo.Send(ctx, SendGroupMessageRequest{
		GroupID:      "grp_snap_1",
		SenderUserID: "usr_sender",
		Type:         "text",
		Content:      "敏感内容证据",
	})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	snap, found, err := repo.GetMessageReportSnapshot(ctx, res.MessageID)
	if err != nil || !found {
		t.Fatalf("expected found snapshot, got found=%v, err=%v", found, err)
	}
	if snap.GroupID != "grp_snap_1" || snap.Type != "text" || snap.Content != "敏感内容证据" || snap.FileID != "" {
		t.Fatalf("unexpected snapshot values: %+v", snap)
	}

	// 3. 构造带媒体附件的消息快照
	repo.mu.Lock()
	repo.items["grp_snap_1"] = append(repo.items["grp_snap_1"], GroupMessageItem{
		MessageID:    "gm_media_test",
		SenderUserID: "usr_sender",
		Type:         "image",
		Media:        map[string]any{"file_id": "file_evidence_999"},
		SentAt:       "2026-09-08T12:00:00Z",
	})
	repo.mu.Unlock()

	mediaSnap, found, err := repo.GetMessageReportSnapshot(ctx, "gm_media_test")
	if err != nil || !found {
		t.Fatalf("expected found media snapshot, got %v", err)
	}
	if mediaSnap.FileID != "file_evidence_999" || mediaSnap.Type != "image" {
		t.Fatalf("unexpected media snapshot: %+v", mediaSnap)
	}
}

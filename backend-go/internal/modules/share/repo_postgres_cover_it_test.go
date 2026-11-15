package share

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// 真实库 IT（-short 跳过）：moment 分享封面来源的产品决策——封面用该瞬间第一张
// 「已完成」媒体（按 position），无图的纯文字瞬间回退作者头像；仓储只带出
// storage key（CoverURL 仍是头像），Service 注入 resolver 后覆盖为签名 URL；
// 非 public / 已删瞬间仍按不存在处理（不泄露内容）。
//
// 库由调用方准备（CI 用 hotspot_it；本地用自建临时库并用后 DROP），本用例内的
// 数据全部写入 BeginScope 事务并随 t.Cleanup 回滚，不留残留行。
const (
	queryInsertITMediaUploadSession = `
INSERT INTO media_upload_sessions (upload_id, user_id, storage_key, file_name, declared_file_size, declared_mime_type, category, status, file_id, created_at)
VALUES ($1, $2, $3, $4, 1024, 'image/jpeg', 'moment_image', $5, $6, NOW())`

	queryInsertITMomentImage = `
INSERT INTO moment_images (moment_id, file_id, position, created_at)
VALUES ($1, $2, $3, NOW())`
)

func itQueryableDB(t *testing.T, factory *persistence.Factory) persistence.QueryableDB {
	t.Helper()
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	return qdb
}

// insertITMomentImage 插入一条媒体上传会话并挂到瞬间的指定 position 上；status
// 传 completed 之外的取值用于验证「未完成上传不能成为封面」。
func insertITMomentImage(t *testing.T, factory *persistence.Factory, momentID, userID, uploadID, storageKey, status string, position int) {
	t.Helper()
	qdb := itQueryableDB(t, factory)
	fileID := "it_file_" + uploadID
	if err := qdb.Exec(context.Background(), queryInsertITMediaUploadSession, uploadID, userID, storageKey, uploadID+".jpg", status, fileID); err != nil {
		t.Fatalf("insert media upload session %s: %v", uploadID, err)
	}
	if err := qdb.Exec(context.Background(), queryInsertITMomentImage, momentID, fileID, position); err != nil {
		t.Fatalf("attach moment image %s: %v", uploadID, err)
	}
}

func TestPostgresITMomentShareCover(t *testing.T) {
	testdb.SkipIfShort(t)
	testdb.Open(t)
	txFactory, cleanup := testdb.BeginScope(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	const (
		creator       = "it_share_cover_creator"
		avatarURL     = "https://example.com/it_share_cover_avatar.png"
		coverMomentID = "it_share_cover_moment_images"
		textMomentID  = "it_share_cover_moment_text"
		signedCover   = "https://storage.example.com/it-signed-cover-first.jpg"
	)
	qdb := itQueryableDB(t, txFactory)
	testdb.InsertUser(t, txFactory, creator, "封面作者")
	if err := qdb.Exec(ctx, `UPDATE users SET avatar_url = $1 WHERE user_id = $2`, avatarURL, creator); err != nil {
		t.Fatalf("set creator avatar: %v", err)
	}

	withImages := testdb.InsertMoment(t, txFactory, coverMomentID, creator)
	textOnly := testdb.InsertMoment(t, txFactory, textMomentID, creator)
	// position 0 的上传尚未完成：封面必须跳过它取 position 1 的已完成媒体。
	insertITMomentImage(t, txFactory, withImages, creator, "it_cover_pending", "uploads/it/cover-pending.jpg", "pending", 0)
	insertITMomentImage(t, txFactory, withImages, creator, "it_cover_first", "uploads/it/cover-first.jpg", "completed", 1)
	insertITMomentImage(t, txFactory, withImages, creator, "it_cover_second", "uploads/it/cover-second.jpg", "completed", 2)

	repo := NewPostgresRepository(txFactory)
	imageShare, err := repo.Create(ctx, createShareCommand{CreatorUserID: creator, ShareType: ShareTypeMoment, Payload: map[string]any{"shared_moment_id": withImages}})
	if err != nil {
		t.Fatalf("create moment share with images: %v", err)
	}
	textShare, err := repo.Create(ctx, createShareCommand{CreatorUserID: creator, ShareType: ShareTypeMoment, Payload: map[string]any{"shared_moment_id": textOnly}})
	if err != nil {
		t.Fatalf("create text moment share: %v", err)
	}

	// 单条 Get：封面 key 取第一张已完成媒体（跳过未完成上传），仓储层 cover_url
	// 仍先填作者头像兜底。
	detail, err := repo.Get(ctx, getShareQuery{ShareCode: imageShare.ShareCode})
	if err != nil {
		t.Fatalf("get moment share with images: %v", err)
	}
	if detail.Preview == nil {
		t.Fatalf("带图瞬间 preview 不得为空: %#v", detail)
	}
	if detail.Preview.coverStorageKey != "uploads/it/cover-first.jpg" {
		t.Fatalf("封面 key = %q，期望第一张已完成媒体 uploads/it/cover-first.jpg", detail.Preview.coverStorageKey)
	}
	if detail.Preview.CoverURL == nil || *detail.Preview.CoverURL != avatarURL {
		t.Fatalf("仓储层 cover_url 应是作者头像兜底，实际 %#v", detail.Preview.CoverURL)
	}
	textDetail, err := repo.Get(ctx, getShareQuery{ShareCode: textShare.ShareCode})
	if err != nil {
		t.Fatalf("get text moment share: %v", err)
	}
	if textDetail.Preview == nil || textDetail.Preview.coverStorageKey != "" {
		t.Fatalf("纯文字瞬间只应有头像兜底、无封面 key: %#v", textDetail.Preview)
	}

	// 列表批量路径：与单条同口径（「我的分享」列表也展示封面）。
	page, err := repo.List(ctx, listSharesQuery{CreatorUserID: creator, Limit: 20})
	if err != nil {
		t.Fatalf("list shares: %v", err)
	}
	byID := make(map[string]ShareListItem, len(page.Items))
	for _, item := range page.Items {
		byID[item.ShareID] = item
	}
	if item := byID[imageShare.ShareID]; item.Preview == nil || item.Preview.coverStorageKey != "uploads/it/cover-first.jpg" {
		t.Fatalf("列表带图瞬间封面 key = %#v", item.Preview)
	}
	if item := byID[textShare.ShareID]; item.Preview == nil || item.Preview.coverStorageKey != "" {
		t.Fatalf("列表纯文字瞬间不得带封面 key: %#v", item.Preview)
	}

	// Service 注入 resolver：cover_url 变为签名 URL；纯文字瞬间不触发签名。
	resolver := &shareStubCoverResolver{url: signedCover}
	svc := NewServiceWithRepo(repo)
	svc.SetCoverResolver(resolver)
	detail, err = svc.Get(ctx, GetShareRequest{ShareCode: imageShare.ShareCode})
	if err != nil {
		t.Fatalf("service get moment share: %v", err)
	}
	if detail.Preview == nil || detail.Preview.CoverURL == nil || *detail.Preview.CoverURL != signedCover {
		t.Fatalf("service cover_url = %#v，期望签名 URL %q", detail.Preview, signedCover)
	}
	textDetail, err = svc.Get(ctx, GetShareRequest{ShareCode: textShare.ShareCode})
	if err != nil {
		t.Fatalf("service get text moment share: %v", err)
	}
	if textDetail.Preview == nil || textDetail.Preview.CoverURL == nil || *textDetail.Preview.CoverURL != avatarURL {
		t.Fatalf("纯文字瞬间 cover_url 应回退头像: %#v", textDetail.Preview)
	}
	if len(resolver.keys) != 1 || resolver.keys[0] != "uploads/it/cover-first.jpg" {
		t.Fatalf("resolver 入参 = %#v，期望仅封面 key 一次", resolver.keys)
	}

	page, err = svc.List(shareContextWithUser(creator), ListSharesRequest{})
	if err != nil {
		t.Fatalf("service list shares: %v", err)
	}
	for _, item := range page.Items {
		if item.ShareID != imageShare.ShareID {
			continue
		}
		if item.Preview == nil || item.Preview.CoverURL == nil || *item.Preview.CoverURL != signedCover {
			t.Fatalf("列表 cover_url = %#v，期望签名 URL", item.Preview)
		}
	}

	// 非 public / 已删瞬间：创建分享按不存在拒绝（10300 防枚举，与既有语义一
	// 致）；已创建的分享在目标不可访问后 preview 置空、分享本身仍可读。
	if err := qdb.Exec(ctx, `UPDATE moments SET privacy_level = 'private' WHERE moment_id = $1`, withImages); err != nil {
		t.Fatalf("make moment private: %v", err)
	}
	if _, err := repo.Create(ctx, createShareCommand{CreatorUserID: creator, ShareType: ShareTypeMoment, Payload: map[string]any{"shared_moment_id": withImages}}); !errors.Is(err, ErrShareNotFound) {
		t.Fatalf("私密瞬间创建分享 err = %v，期望 ErrShareNotFound", err)
	}
	detail, err = repo.Get(ctx, getShareQuery{ShareCode: imageShare.ShareCode})
	if err != nil {
		t.Fatalf("get share of private moment: %v", err)
	}
	if detail.Preview != nil {
		t.Fatalf("目标转私密后 preview 必须置空: %#v", detail.Preview)
	}

	if err := qdb.Exec(ctx, `UPDATE moments SET privacy_level = 'public', is_deleted = TRUE WHERE moment_id = $1`, withImages); err != nil {
		t.Fatalf("soft delete moment: %v", err)
	}
	if _, err := repo.Create(ctx, createShareCommand{CreatorUserID: creator, ShareType: ShareTypeMoment, Payload: map[string]any{"shared_moment_id": withImages}}); !errors.Is(err, ErrShareNotFound) {
		t.Fatalf("已删瞬间创建分享 err = %v，期望 ErrShareNotFound", err)
	}
	detail, err = repo.Get(ctx, getShareQuery{ShareCode: imageShare.ShareCode})
	if err != nil {
		t.Fatalf("get share of deleted moment: %v", err)
	}
	if detail.Preview != nil {
		t.Fatalf("目标已删后 preview 必须置空: %#v", detail.Preview)
	}
}

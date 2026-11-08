package privatemessages

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// 本文件补齐 PG 仓储在真实库难以稳定制造的失败/边界分支：单查询错误、
// 并发撤回竞态（ErrNoRows）、行扫描失败、rows.Err 尾错、空结果，以及
// 「连接不可用即 fail-closed」的全部入口。成功语义由真实库 IT/契约测试覆盖。

// covScanErrRows 让第一次 Scan 返回错误（模拟列类型不匹配等扫描失败）。
type covScanErrRows struct {
	err  error
	done bool
}

func (r *covScanErrRows) Next() bool {
	if r.done {
		return false
	}
	r.done = true
	return true
}
func (r *covScanErrRows) Scan(...any) error { return r.err }
func (r *covScanErrRows) Close() error      { return nil }
func (r *covScanErrRows) Err() error        { return nil }

// covTailErrRows 空结果但 rows.Err 非 nil（模拟迭代中途断连）。
type covTailErrRows struct{ err error }

func (r *covTailErrRows) Next() bool        { return false }
func (r *covTailErrRows) Scan(...any) error { return nil }
func (r *covTailErrRows) Close() error      { return nil }
func (r *covTailErrRows) Err() error        { return r.err }

// covRepoBackend 在既有 privateMessageBackendStub 之上按 SQL 覆写 QueryRow/
// Query 返回值；未覆写的 SQL 继续走 stub 的既有语义。
type covRepoBackend struct {
	*privateMessageBackendStub
	rows    map[string]persistence.Row
	queries map[string]persistence.Rows
}

func (b *covRepoBackend) QueryRow(ctx context.Context, query string, args ...any) persistence.Row {
	if row, ok := b.rows[query]; ok {
		return row
	}
	return b.privateMessageBackendStub.QueryRow(ctx, query, args...)
}

func (b *covRepoBackend) Query(ctx context.Context, query string, args ...any) (persistence.Rows, error) {
	if rows, ok := b.queries[query]; ok {
		return rows, nil
	}
	return b.privateMessageBackendStub.Query(ctx, query, args...)
}

func emptyCovStub() *privateMessageBackendStub {
	return &privateMessageBackendStub{
		items:      map[string][]privateMessageStubRecord{},
		readStates: map[string]map[string]privateMessageReadState{},
		execErrs:   map[string]error{},
		queryErrs:  map[string]error{},
	}
}

func newCovRepo(stub *privateMessageBackendStub, rows map[string]persistence.Row, queries map[string]persistence.Rows) *PostgresRepository {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &covRepoBackend{privateMessageBackendStub: stub, rows: rows, queries: queries}
	return NewPostgresRepository(factory)
}

func newEmptyCovRepo() *PostgresRepository { return newCovRepo(emptyCovStub(), nil, nil) }

func covChatKey() string { return conversationKey("user_alice", "user_bob") }

// covSoloRepo 返回「DB 不可查询」的仓储：所有入口必须 fail-closed。
func covSoloRepo() *PostgresRepository {
	return NewPostgresRepository(&persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     privateMessageNonQueryableDB{},
	})
}

// --- Send ---

type covFailingChatEnsurer struct{ err error }

func (e covFailingChatEnsurer) EnsureChat(context.Context, string, string, string, time.Time) error {
	return e.err
}

func TestPostgresSendChatEnsurerFailureAborts(t *testing.T) {
	repo := newEmptyCovRepo()
	repo.SetChatEnsurer(covFailingChatEnsurer{err: errors.New("chat write failed")})

	_, err := repo.Send(context.Background(), "user_alice", "user_bob", SendPrivateMessageRequest{Type: "text", Content: "hi"})
	if err == nil || err.Error() != "chat write failed" {
		t.Fatalf("expected chat ensurer error, got %v", err)
	}
}

func TestPostgresSendRequiresChatEnsurer(t *testing.T) {
	repo := newEmptyCovRepo()

	_, err := repo.Send(context.Background(), "user_alice", "user_bob", SendPrivateMessageRequest{Type: "text", Content: "hi"})
	if !errors.Is(err, errChatEnsurerNotWired) {
		t.Fatalf("expected errChatEnsurerNotWired, got %v", err)
	}
}

// --- Update ---

func TestPostgresUpdateRejectsNonAuthorAndNonEditable(t *testing.T) {
	chatKey := covChatKey()
	newRepo := func(record privateMessageStubRecord) *PostgresRepository {
		stub := emptyCovStub()
		stub.items[chatKey] = []privateMessageStubRecord{record}
		return newCovRepo(stub, nil, nil)
	}

	// 非作者编辑 → ErrMessagePermissionDenied（message_id 存在且为 text）。
	repo := newRepo(privateMessageStubRecord{MessageID: "pm_1", ChatKey: chatKey, SenderID: "user_bob", Type: "text", SentAt: "2026-03-30T10:00:00Z"})
	if _, err := repo.Update(context.Background(), "user_alice", UpdatePrivateMessageRequest{MessageID: "pm_1", Content: "x"}); !errors.Is(err, ErrMessagePermissionDenied) {
		t.Fatalf("expected ErrMessagePermissionDenied, got %v", err)
	}

	// 非 text 类型不可编辑。
	repo = newRepo(privateMessageStubRecord{MessageID: "pm_2", ChatKey: chatKey, SenderID: "user_alice", Type: "image", SentAt: "2026-03-30T10:00:00Z"})
	if _, err := repo.Update(context.Background(), "user_alice", UpdatePrivateMessageRequest{MessageID: "pm_2", Content: "x"}); !errors.Is(err, ErrMessageNotEditable) {
		t.Fatalf("expected ErrMessageNotEditable for non-text, got %v", err)
	}

	// 已撤回不可编辑。
	repo = newRepo(privateMessageStubRecord{MessageID: "pm_3", ChatKey: chatKey, SenderID: "user_alice", Type: "text", RecalledAt: "2026-03-30T10:01:00Z", SentAt: "2026-03-30T10:00:00Z"})
	if _, err := repo.Update(context.Background(), "user_alice", UpdatePrivateMessageRequest{MessageID: "pm_3", Content: "x"}); !errors.Is(err, ErrMessageNotEditable) {
		t.Fatalf("expected ErrMessageNotEditable for recalled, got %v", err)
	}
}

// Update 的 UPDATE...RETURNING 与元数据读取之间被并发撤回：ErrNoRows 必须
// 映射为「不可编辑」而不是广播假编辑事件；其它 DB 错误原样上抛。
func TestPostgresUpdateConcurrentRecallAndDBErrors(t *testing.T) {
	chatKey := covChatKey()
	newRepo := func(row persistence.Row) *PostgresRepository {
		stub := emptyCovStub()
		stub.items[chatKey] = []privateMessageStubRecord{
			{MessageID: "pm_1", ChatKey: chatKey, SenderID: "user_alice", Type: "text", Content: "original", SentAt: "2026-03-30T10:00:00Z"},
		}
		return newCovRepo(stub, map[string]persistence.Row{queryUpdatePrivateMessage: row}, nil)
	}

	t.Run("concurrent recall maps to not editable", func(t *testing.T) {
		repo := newRepo(testdb.RowValues{Err: pgx.ErrNoRows})
		if _, err := repo.Update(context.Background(), "user_alice", UpdatePrivateMessageRequest{MessageID: "pm_1", Content: "x"}); !errors.Is(err, ErrMessageNotEditable) {
			t.Fatalf("expected ErrMessageNotEditable, got %v", err)
		}
	})

	t.Run("update db error propagates", func(t *testing.T) {
		boom := errors.New("update failed")
		repo := newRepo(testdb.RowValues{Err: boom})
		if _, err := repo.Update(context.Background(), "user_alice", UpdatePrivateMessageRequest{MessageID: "pm_1", Content: "x"}); !errors.Is(err, boom) {
			t.Fatalf("expected update error, got %v", err)
		}
	})
}

// --- Recall ---

func TestPostgresRecallRejectsNonAuthorRecalledAndExpired(t *testing.T) {
	chatKey := covChatKey()
	newRepo := func(record privateMessageStubRecord) *PostgresRepository {
		stub := emptyCovStub()
		stub.items[chatKey] = []privateMessageStubRecord{record}
		return newCovRepo(stub, nil, nil)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	// 非作者撤回 → 10200。
	repo := newRepo(privateMessageStubRecord{MessageID: "pm_1", ChatKey: chatKey, SenderID: "user_bob", Type: "text", SentAt: now})
	if _, err := repo.Recall(context.Background(), "user_alice", 2*time.Minute, RecallPrivateMessageRequest{MessageID: "pm_1"}); !errors.Is(err, ErrMessagePermissionDenied) {
		t.Fatalf("expected ErrMessagePermissionDenied, got %v", err)
	}

	// 已撤回 → 不可重复撤回。
	repo = newRepo(privateMessageStubRecord{MessageID: "pm_2", ChatKey: chatKey, SenderID: "user_alice", Type: "text", RecalledAt: now, SentAt: now})
	if _, err := repo.Recall(context.Background(), "user_alice", 2*time.Minute, RecallPrivateMessageRequest{MessageID: "pm_2"}); !errors.Is(err, ErrMessageNotRecallable) {
		t.Fatalf("expected ErrMessageNotRecallable for recalled, got %v", err)
	}

	// 时间窗 0 = 撤回禁用。
	repo = newRepo(privateMessageStubRecord{MessageID: "pm_3", ChatKey: chatKey, SenderID: "user_alice", Type: "text", SentAt: now})
	if _, err := repo.Recall(context.Background(), "user_alice", 0, RecallPrivateMessageRequest{MessageID: "pm_3"}); !errors.Is(err, ErrMessageNotRecallable) {
		t.Fatalf("expected ErrMessageNotRecallable for zero window, got %v", err)
	}
}

// sent_at 无法解析时 fail-closed：不得把异常时间当成「已超时」或「未超时」放行。
func TestPostgresRecallRejectsUnparsableSentAt(t *testing.T) {
	chatKey := covChatKey()
	stub := emptyCovStub()
	stub.items[chatKey] = []privateMessageStubRecord{
		{MessageID: "pm_1", ChatKey: chatKey, SenderID: "user_alice", Type: "text", SentAt: "not-a-timestamp"},
	}
	repo := newCovRepo(stub, nil, nil)

	if _, err := repo.Recall(context.Background(), "user_alice", 2*time.Minute, RecallPrivateMessageRequest{MessageID: "pm_1"}); !errors.Is(err, ErrMessageNotRecallable) {
		t.Fatalf("expected ErrMessageNotRecallable, got %v", err)
	}
}

func TestPostgresRecallConcurrentAndDBErrors(t *testing.T) {
	chatKey := covChatKey()
	newRepo := func(row persistence.Row) *PostgresRepository {
		stub := emptyCovStub()
		stub.items[chatKey] = []privateMessageStubRecord{
			{MessageID: "pm_1", ChatKey: chatKey, SenderID: "user_alice", Type: "text", SentAt: time.Now().UTC().Format(time.RFC3339Nano)},
		}
		return newCovRepo(stub, map[string]persistence.Row{queryRecallPrivateMessage: row}, nil)
	}

	t.Run("concurrent recall maps to not recallable", func(t *testing.T) {
		repo := newRepo(testdb.RowValues{Err: pgx.ErrNoRows})
		if _, err := repo.Recall(context.Background(), "user_alice", 2*time.Minute, RecallPrivateMessageRequest{MessageID: "pm_1"}); !errors.Is(err, ErrMessageNotRecallable) {
			t.Fatalf("expected ErrMessageNotRecallable, got %v", err)
		}
	})

	t.Run("recall db error propagates", func(t *testing.T) {
		boom := errors.New("recall failed")
		repo := newRepo(testdb.RowValues{Err: boom})
		if _, err := repo.Recall(context.Background(), "user_alice", 2*time.Minute, RecallPrivateMessageRequest{MessageID: "pm_1"}); !errors.Is(err, boom) {
			t.Fatalf("expected recall error, got %v", err)
		}
	})
}

// --- MarkAsRead ---

// 空会话：无消息可标记 → 返回空位点（与内存实现一致），不当作错误。
func TestPostgresMarkAsReadEmptySession(t *testing.T) {
	repo := newCovRepo(emptyCovStub(), map[string]persistence.Row{
		queryMarkAllPrivateMessagesReadForChat: testdb.RowValues{Err: pgx.ErrNoRows},
	}, nil)

	meta, err := repo.MarkAsRead(context.Background(), "user_alice", MarkPrivateMessagesReadRequest{TargetUserID: "user_bob"})
	if err != nil {
		t.Fatalf("empty session must not error, got %v", err)
	}
	if meta.TargetUserID != "user_bob" || meta.MessageID != "" {
		t.Fatalf("unexpected empty meta: %#v", meta)
	}
}

func TestPostgresMarkAsReadDBErrors(t *testing.T) {
	t.Run("latest message lookup failure", func(t *testing.T) {
		boom := errors.New("read state lookup failed")
		repo := newCovRepo(emptyCovStub(), map[string]persistence.Row{
			queryMarkAllPrivateMessagesReadForChat: testdb.RowValues{Err: boom},
		}, nil)
		if _, err := repo.MarkAsRead(context.Background(), "user_alice", MarkPrivateMessagesReadRequest{TargetUserID: "user_bob"}); !errors.Is(err, boom) {
			t.Fatalf("expected lookup error, got %v", err)
		}
	})

	t.Run("read cursor upsert failure", func(t *testing.T) {
		boom := errors.New("upsert failed")
		chatKey := covChatKey()
		stub := emptyCovStub()
		stub.items[chatKey] = []privateMessageStubRecord{
			{MessageID: "pm_1", ChatKey: chatKey, SenderID: "user_bob", Type: "text", SentAt: "2026-03-30T10:00:00Z"},
		}
		stub.execErrs[queryUpsertPrivateMessageReadState] = boom
		repo := newCovRepo(stub, nil, nil)
		if _, err := repo.MarkAsRead(context.Background(), "user_alice", MarkPrivateMessagesReadRequest{TargetUserID: "user_bob", UpToMessageID: "pm_1"}); !errors.Is(err, boom) {
			t.Fatalf("expected upsert error, got %v", err)
		}
	})
}

// --- 单条查询：失败/空/命中 ---

func TestPostgresGetMessageMetaFailsClosed(t *testing.T) {
	if _, err := covSoloRepo().getMessageMetaImpl(context.Background(), "pm_1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestPostgresGetMessageForReplyBranches(t *testing.T) {
	if _, found, err := covSoloRepo().GetMessageForReply(context.Background(), "pm_1", "user_alice"); !errors.Is(err, persistence.ErrNotImplemented) || found {
		t.Fatalf("expected ErrNotImplemented, got found=%v err=%v", found, err)
	}

	repo := newCovRepo(emptyCovStub(), map[string]persistence.Row{
		queryGetPrivateMessageForReply: testdb.RowValues{Err: pgx.ErrNoRows},
	}, nil)
	if result, found, err := repo.GetMessageForReply(context.Background(), "pm_1", "user_alice"); err != nil || found || result != nil {
		t.Fatalf("missing message must yield nil,false,nil; got %#v found=%v err=%v", result, found, err)
	}
}

func TestPostgresGetMessageForReplyIncludesShareCode(t *testing.T) {
	rows := map[string]persistence.Row{
		queryGetPrivateMessageForReply: testdb.RowValues{Values: []any{"user_bob", "Bob", "看这个", "SHARECODE01"}},
	}
	repo := newCovRepo(emptyCovStub(), rows, nil)

	result, found, err := repo.GetMessageForReply(context.Background(), "pm_1", "user_alice")
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if result["user_id"] != "user_bob" || result["nickname"] != "Bob" || result["content"] != "看这个" || result["share_code"] != "SHARECODE01" {
		t.Fatalf("unexpected reply payload: %#v", result)
	}

	// share_code 为空时不写该键（契约：可选字段）。
	rows[queryGetPrivateMessageForReply] = testdb.RowValues{Values: []any{"user_bob", "Bob", "文本", ""}}
	result, found, err = repo.GetMessageForReply(context.Background(), "pm_2", "user_alice")
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if _, ok := result["share_code"]; ok {
		t.Fatalf("empty share_code must be omitted: %#v", result)
	}
}

func TestPostgresGetMediaForMessageBranches(t *testing.T) {
	if _, found, err := covSoloRepo().GetMediaForMessage(context.Background(), "file_1", "user_alice"); !errors.Is(err, persistence.ErrNotImplemented) || found {
		t.Fatalf("expected ErrNotImplemented, got found=%v err=%v", found, err)
	}

	rows := map[string]persistence.Row{
		queryGetMediaFileForMessage: testdb.RowValues{Err: pgx.ErrNoRows},
	}
	repo := newCovRepo(emptyCovStub(), rows, nil)
	if result, found, err := repo.GetMediaForMessage(context.Background(), "file_1", "user_alice"); err != nil || found || result != nil {
		t.Fatalf("missing media must yield nil,false,nil; got %#v found=%v err=%v", result, found, err)
	}

	rows[queryGetMediaFileForMessage] = testdb.RowValues{Values: []any{"file_1", "media/storage/file_1.png", "file_1.png", 1024, "image/png"}}
	result, found, err := repo.GetMediaForMessage(context.Background(), "file_1", "user_alice")
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if result["url"] != "media/storage/file_1.png" || result["file_name"] != "file_1.png" || result["file_size"] != 1024 || result["mime_type"] != "image/png" {
		t.Fatalf("unexpected media payload: %#v", result)
	}
}

func TestPostgresRecallAsAdminBranches(t *testing.T) {
	if hit, err := covSoloRepo().RecallAsAdmin(context.Background(), "pm_1"); !errors.Is(err, persistence.ErrNotImplemented) || hit {
		t.Fatalf("expected ErrNotImplemented, got hit=%v err=%v", hit, err)
	}

	rows := map[string]persistence.Row{
		queryRecallPrivateMessageAsAdmin: testdb.RowValues{Err: pgx.ErrNoRows},
	}
	repo := newCovRepo(emptyCovStub(), rows, nil)
	if hit, err := repo.RecallAsAdmin(context.Background(), "pm_missing"); err != nil || hit {
		t.Fatalf("already recalled must be false,nil; got hit=%v err=%v", hit, err)
	}

	rows[queryRecallPrivateMessageAsAdmin] = testdb.RowValues{Values: []any{"pm_1"}}
	if hit, err := repo.RecallAsAdmin(context.Background(), "pm_1"); err != nil || !hit {
		t.Fatalf("expected successful recall, got hit=%v err=%v", hit, err)
	}
}

func TestPostgresMessageParticipantsBranches(t *testing.T) {
	if _, _, found, err := covSoloRepo().MessageParticipants(context.Background(), "pm_1"); !errors.Is(err, persistence.ErrNotImplemented) || found {
		t.Fatalf("expected ErrNotImplemented, got found=%v err=%v", found, err)
	}

	rows := map[string]persistence.Row{
		queryPrivateMessageParticipants: testdb.RowValues{Err: pgx.ErrNoRows},
	}
	repo := newCovRepo(emptyCovStub(), rows, nil)
	if _, _, found, err := repo.MessageParticipants(context.Background(), "pm_missing"); err != nil || found {
		t.Fatalf("missing message must be false,nil; got found=%v err=%v", found, err)
	}

	rows[queryPrivateMessageParticipants] = testdb.RowValues{Values: []any{"user_alice", "user_bob"}}
	sender, receiver, found, err := repo.MessageParticipants(context.Background(), "pm_1")
	if err != nil || !found || sender != "user_alice" || receiver != "user_bob" {
		t.Fatalf("participants = %q/%q found=%v err=%v", sender, receiver, found, err)
	}
}

func TestPostgresMessageExistsBranches(t *testing.T) {
	if found, sender, err := covSoloRepo().MessageExists(context.Background(), "pm_1"); !errors.Is(err, persistence.ErrNotImplemented) || found || sender != "" {
		t.Fatalf("expected ErrNotImplemented, got found=%v err=%v", found, err)
	}

	rows := map[string]persistence.Row{
		queryPrivateMessageExists: testdb.RowValues{Err: pgx.ErrNoRows},
	}
	repo := newCovRepo(emptyCovStub(), rows, nil)
	if found, _, err := repo.MessageExists(context.Background(), "pm_missing"); err != nil || found {
		t.Fatalf("missing message must be false,nil; got found=%v err=%v", found, err)
	}

	rows[queryPrivateMessageExists] = testdb.RowValues{Values: []any{"user_alice"}}
	found, sender, err := repo.MessageExists(context.Background(), "pm_1")
	if err != nil || !found || sender != "user_alice" {
		t.Fatalf("exists = %v/%q err=%v", found, sender, err)
	}
}

func TestPostgresGetMessageReportSnapshotBranches(t *testing.T) {
	if _, found, err := covSoloRepo().GetMessageReportSnapshot(context.Background(), "pm_1"); !errors.Is(err, persistence.ErrNotImplemented) || found {
		t.Fatalf("expected ErrNotImplemented, got found=%v err=%v", found, err)
	}

	rows := map[string]persistence.Row{
		queryGetPrivateMessageSnapshot: testdb.RowValues{Err: pgx.ErrNoRows},
	}
	repo := newCovRepo(emptyCovStub(), rows, nil)
	if _, found, err := repo.GetMessageReportSnapshot(context.Background(), "pm_missing"); err != nil || found {
		t.Fatalf("missing message must be false,nil; got found=%v err=%v", found, err)
	}

	rows[queryGetPrivateMessageSnapshot] = testdb.RowValues{Values: []any{
		"user_alice", "user_bob", "text", "举报内容", "", "2026-03-30T10:00:00.000000Z",
	}}
	snapshot, found, err := repo.GetMessageReportSnapshot(context.Background(), "pm_1")
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if snapshot.SenderUserID != "user_alice" || snapshot.TargetUserID != "user_bob" || snapshot.Content != "举报内容" {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
}

func TestPostgresCountMessagesByTargetBranches(t *testing.T) {
	if _, err := covSoloRepo().CountMessagesByTarget(context.Background(), "user_alice", time.Now()); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}

	boom := errors.New("count query failed")
	failing := emptyCovStub()
	failing.queryErrs[queryCountPrivateMessagesByTarget] = boom
	if _, err := newCovRepo(failing, nil, nil).CountMessagesByTarget(context.Background(), "user_alice", time.Now()); !errors.Is(err, boom) {
		t.Fatalf("expected query error, got %v", err)
	}

	scanRepo := newCovRepo(emptyCovStub(), nil, map[string]persistence.Rows{
		queryCountPrivateMessagesByTarget: &covScanErrRows{err: boom},
	})
	if _, err := scanRepo.CountMessagesByTarget(context.Background(), "user_alice", time.Now()); !errors.Is(err, boom) {
		t.Fatalf("expected scan error, got %v", err)
	}

	tailRepo := newCovRepo(emptyCovStub(), nil, map[string]persistence.Rows{
		queryCountPrivateMessagesByTarget: &covTailErrRows{err: boom},
	})
	if _, err := tailRepo.CountMessagesByTarget(context.Background(), "user_alice", time.Now()); !errors.Is(err, boom) {
		t.Fatalf("expected rows.Err propagation, got %v", err)
	}

	countsRepo := newCovRepo(emptyCovStub(), nil, map[string]persistence.Rows{
		queryCountPrivateMessagesByTarget: &testdb.RowsValues{Rows: [][]any{
			{"user_bob", 2},
			{"user_carol", 1},
		}},
	})
	counts, err := countsRepo.CountMessagesByTarget(context.Background(), "user_alice", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if counts["user_bob"] != 2 || counts["user_carol"] != 1 {
		t.Fatalf("unexpected counts: %#v", counts)
	}
}

func TestPostgresLockStrangerQuotaBranches(t *testing.T) {
	if err := covSoloRepo().LockStrangerQuota(context.Background(), "user_alice", "2026-03-30"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}

	boom := errors.New("advisory lock failed")
	stub := emptyCovStub()
	stub.execErrs[queryLockStrangerQuota] = boom
	err := newCovRepo(stub, nil, nil).LockStrangerQuota(context.Background(), "user_alice", "2026-03-30")
	if !errors.Is(err, boom) {
		t.Fatalf("expected wrapped lock error, got %v", err)
	}
	if err.Error() != "privatemessages.LockStrangerQuota: advisory lock failed" {
		t.Fatalf("unexpected error text: %v", err)
	}
}

// --- 读路径 ---

func TestPostgresLastBranches(t *testing.T) {
	boom := errors.New("history query failed")
	failing := emptyCovStub()
	failing.queryErrs[queryPrivateMessagesHistory] = boom
	if _, err := newCovRepo(failing, nil, nil).Last(context.Background(), "user_alice", "user_bob", nil); !errors.Is(err, boom) {
		t.Fatalf("expected history error, got %v", err)
	}

	last, err := newEmptyCovRepo().Last(context.Background(), "user_alice", "user_bob", nil)
	if err != nil || last != nil {
		t.Fatalf("empty conversation must yield nil,nil; got %#v err=%v", last, err)
	}
}

func TestPostgresUnreadCountBranches(t *testing.T) {
	if _, err := covSoloRepo().UnreadCount(context.Background(), "user_alice", "user_bob", nil); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}

	rows := map[string]persistence.Row{
		queryUnreadPrivateMessageCount: testdb.RowValues{Values: []any{3}},
	}
	repo := newCovRepo(emptyCovStub(), rows, nil)
	deletedAfter := time.Now().UTC().Add(-time.Hour)
	count, err := repo.UnreadCount(context.Background(), "user_alice", "user_bob", &deletedAfter)
	if err != nil || count != 3 {
		t.Fatalf("count=%d err=%v", count, err)
	}

	boom := errors.New("count failed")
	rows[queryUnreadPrivateMessageCount] = testdb.RowValues{Err: boom}
	if _, err := repo.UnreadCount(context.Background(), "user_alice", "user_bob", nil); !errors.Is(err, boom) {
		t.Fatalf("expected count error, got %v", err)
	}
}

func TestPostgresLastByChatKeysBranches(t *testing.T) {
	ctx := context.Background()
	key := covChatKey()

	if got, err := newEmptyCovRepo().LastByChatKeys(ctx, "user_alice", nil, nil); err != nil || len(got) != 0 {
		t.Fatalf("empty key list must yield empty map; got %#v err=%v", got, err)
	}
	if _, err := covSoloRepo().LastByChatKeys(ctx, "user_alice", []string{key}, nil); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}

	boom := errors.New("batch last failed")
	failing := emptyCovStub()
	failing.queryErrs[queryLastPrivateMessagesByChatKeys] = boom
	if _, err := newCovRepo(failing, nil, nil).LastByChatKeys(ctx, "user_alice", []string{key}, nil); !errors.Is(err, boom) {
		t.Fatalf("expected query error, got %v", err)
	}

	scanRepo := newCovRepo(emptyCovStub(), nil, map[string]persistence.Rows{
		queryLastPrivateMessagesByChatKeys: &covScanErrRows{err: boom},
	})
	if _, err := scanRepo.LastByChatKeys(ctx, "user_alice", []string{key}, nil); !errors.Is(err, boom) {
		t.Fatalf("expected scan error, got %v", err)
	}

	tailRepo := newCovRepo(emptyCovStub(), nil, map[string]persistence.Rows{
		queryLastPrivateMessagesByChatKeys: &covTailErrRows{err: boom},
	})
	if _, err := tailRepo.LastByChatKeys(ctx, "user_alice", []string{key}, nil); !errors.Is(err, boom) {
		t.Fatalf("expected rows.Err propagation, got %v", err)
	}
}

// 批量末条命中媒体文件时回填完整 MessageMedia；媒体查询失败必须上抛
// （不得返回缺 media 的半成品会话列表）。
func TestPostgresLastByChatKeysBackfillsMedia(t *testing.T) {
	key := covChatKey()
	width, height := 800, 600
	// 每次调用都返回新的 Rows（RowsValues 是有状态的游标）。
	newRepo := func(stub *privateMessageBackendStub) *PostgresRepository {
		return newCovRepo(stub, nil, map[string]persistence.Rows{
			queryLastPrivateMessagesByChatKeys: &testdb.RowsValues{Rows: [][]any{
				{key, "pm_1", "image", "", "file_1", "", "user_alice", "Alice", "", "2026-03-30T10:00:00Z"},
			}},
		})
	}

	stub := emptyCovStub()
	stub.mediaSessions = map[string][]any{
		"file_1": {"file_1", "media/storage/file_1.png", "file_1.png", 1024, "image/png", "media/thumb/file_1.png", &width, &height, (*int)(nil)},
	}
	got, err := newRepo(stub).LastByChatKeys(context.Background(), "user_alice", []string{key}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	item, ok := got[key]
	if !ok || item.Media == nil || item.Media.URL != "media/storage/file_1.png" {
		t.Fatalf("expected media backfill, got %#v", got)
	}
	if item.Media.Width == nil || *item.Media.Width != 800 || item.Media.MimeType == nil || *item.Media.MimeType != "image/png" {
		t.Fatalf("expected full media contract fields, got %#v", item.Media)
	}

	// 媒体查询失败必须上抛：不得返回缺 media 的半成品会话列表。
	boom := errors.New("media query failed")
	failStub := emptyCovStub()
	failStub.queryErrs = map[string]error{queryMediaFilesByIDs: boom}
	if _, err := newRepo(failStub).LastByChatKeys(context.Background(), "user_alice", []string{key}, nil); !errors.Is(err, boom) {
		t.Fatalf("expected media query error, got %v", err)
	}
}

func TestPostgresMediaByFileIDsErrors(t *testing.T) {
	boom := errors.New("media query failed")
	failing := emptyCovStub()
	failing.queryErrs[queryMediaFilesByIDs] = boom
	repo := newCovRepo(failing, nil, nil)

	queryable, ok := persistence.RepoQueryable(context.Background(), repo.factory)
	if !ok {
		t.Fatal("expected queryable test backend")
	}
	if _, err := repo.mediaByFileIDs(context.Background(), queryable, []string{"file_1"}); !errors.Is(err, boom) {
		t.Fatalf("expected query error, got %v", err)
	}

	scanRepo := newCovRepo(emptyCovStub(), nil, map[string]persistence.Rows{
		queryMediaFilesByIDs: &covScanErrRows{err: boom},
	})
	queryable, ok = persistence.RepoQueryable(context.Background(), scanRepo.factory)
	if !ok {
		t.Fatal("expected queryable test backend")
	}
	if _, err := scanRepo.mediaByFileIDs(context.Background(), queryable, []string{"file_1"}); !errors.Is(err, boom) {
		t.Fatalf("expected scan error, got %v", err)
	}
}

func TestPostgresUnreadCountsByChatKeysBranches(t *testing.T) {
	ctx := context.Background()
	key := covChatKey()

	if got, err := newEmptyCovRepo().UnreadCountsByChatKeys(ctx, "user_alice", nil, nil); err != nil || len(got) != 0 {
		t.Fatalf("empty key list must yield empty map; got %#v err=%v", got, err)
	}
	if _, err := covSoloRepo().UnreadCountsByChatKeys(ctx, "user_alice", []string{key}, nil); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}

	boom := errors.New("unread batch failed")
	failing := emptyCovStub()
	failing.queryErrs[queryUnreadPrivateMessageCountsByChatKeys] = boom
	if _, err := newCovRepo(failing, nil, nil).UnreadCountsByChatKeys(ctx, "user_alice", []string{key}, nil); !errors.Is(err, boom) {
		t.Fatalf("expected query error, got %v", err)
	}

	scanRepo := newCovRepo(emptyCovStub(), nil, map[string]persistence.Rows{
		queryUnreadPrivateMessageCountsByChatKeys: &covScanErrRows{err: boom},
	})
	if _, err := scanRepo.UnreadCountsByChatKeys(ctx, "user_alice", []string{key}, nil); !errors.Is(err, boom) {
		t.Fatalf("expected scan error, got %v", err)
	}

	okRepo := newCovRepo(emptyCovStub(), nil, map[string]persistence.Rows{
		queryUnreadPrivateMessageCountsByChatKeys: &testdb.RowsValues{Rows: [][]any{{key, 4}}},
	})
	counts, err := okRepo.UnreadCountsByChatKeys(ctx, "user_alice", []string{key}, map[string]*time.Time{key: nil})
	if err != nil || counts[key] != 4 {
		t.Fatalf("counts=%#v err=%v", counts, err)
	}
}

func TestPostgresHistorySinceScanAndMediaErrors(t *testing.T) {
	boom := errors.New("history scan failed")
	deletedAfter := time.Now().UTC().Add(-time.Hour)

	scanRepo := newCovRepo(emptyCovStub(), nil, map[string]persistence.Rows{
		queryPrivateMessagesHistory: &covScanErrRows{err: boom},
	})
	if _, err := scanRepo.History(context.Background(), "user_alice", ListPrivateMessagesRequest{TargetUserID: "user_bob"}); !errors.Is(err, boom) {
		t.Fatalf("expected scan error, got %v", err)
	}

	tailRepo := newCovRepo(emptyCovStub(), nil, map[string]persistence.Rows{
		queryPrivateMessagesHistory: &covTailErrRows{err: boom},
	})
	if _, err := tailRepo.Recent(context.Background(), "user_alice", "user_bob", 5, &deletedAfter); !errors.Is(err, boom) {
		t.Fatalf("expected rows.Err propagation, got %v", err)
	}

	// 页面含媒体但批量回填失败 → 整体上抛（不返回缺 media 的半成品页）。
	mediaRepo := newCovRepo(emptyCovStub(), nil, map[string]persistence.Rows{
		queryPrivateMessagesHistory: &testdb.RowsValues{Rows: [][]any{{
			"pm_1", "image", "", "file_1", "", "user_alice", "Alice", "",
			"", "", "", "", "", "", "", "", "2026-03-30T10:00:00Z", "", "", true,
		}}},
		queryMediaFilesByIDs: &covScanErrRows{err: boom},
	})
	if _, err := mediaRepo.History(context.Background(), "user_bob", ListPrivateMessagesRequest{TargetUserID: "user_alice"}); !errors.Is(err, boom) {
		t.Fatalf("expected media backfill error, got %v", err)
	}
}

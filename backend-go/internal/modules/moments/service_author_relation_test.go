package moments

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// fakeFollowRelationReader 记录批量查询参数并返回预置关系。
type fakeFollowRelationReader struct {
	gotViewer  string
	gotAuthors []string
	result     map[string]UserRelationView
	err        error
}

func (f *fakeFollowRelationReader) AuthorRelations(_ context.Context, viewerID string, authorIDs []string) (map[string]UserRelationView, error) {
	f.gotViewer = viewerID
	f.gotAuthors = append([]string(nil), authorIDs...)
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func relItems() []MomentItem {
	return []MomentItem{
		{MomentID: "m1", User: UserInfo{UserID: "u1", Nickname: "u1"}},
		{MomentID: "m2", User: UserInfo{UserID: "u2", Nickname: "u2"}},
		{MomentID: "m3", User: UserInfo{UserID: "u2", Nickname: "u2"}}, // 同一作者重复出现
	}
}

// 批量填充：只命中关系表里有的作者，且作者 id 先去重。
func TestFillAuthorRelationsFillsOnlyKnownAuthors(t *testing.T) {
	reader := &fakeFollowRelationReader{
		result: map[string]UserRelationView{
			"u2": {IsFollowing: true, IsFollowedBy: false},
		},
	}
	svc := NewServiceWithRepo(&stubMomentRepo{})
	svc.SetFollowRelationReader(reader)

	items := relItems()
	svc.fillAuthorRelations(context.Background(), items, "viewer")

	if reader.gotViewer != "viewer" {
		t.Fatalf("viewerID 未透传: %q", reader.gotViewer)
	}
	if len(reader.gotAuthors) != 2 { // u1, u2（u2 去重）
		t.Fatalf("作者去重失效: %v", reader.gotAuthors)
	}
	if items[0].AuthorRelation != nil {
		t.Fatalf("未关注的作者不应有关系: %+v", items[0].AuthorRelation)
	}
	for _, i := range []int{1, 2} {
		if items[i].AuthorRelation == nil || !items[i].AuthorRelation.IsFollowing {
			t.Fatalf("第 %d 项应有 is_following=true: %+v", i, items[i].AuthorRelation)
		}
	}
}

// 匿名（viewerID 空）不查询、不填充。
func TestFillAuthorRelationsSkipsAnonymous(t *testing.T) {
	reader := &fakeFollowRelationReader{result: map[string]UserRelationView{"u1": {IsFollowing: true}}}
	svc := NewServiceWithRepo(&stubMomentRepo{})
	svc.SetFollowRelationReader(reader)

	items := relItems()
	svc.fillAuthorRelations(context.Background(), items, "")
	if reader.gotViewer != "" || len(reader.gotAuthors) != 0 {
		t.Fatalf("匿名不应发起查询: viewer=%q authors=%v", reader.gotViewer, reader.gotAuthors)
	}
	for i := range items {
		if items[i].AuthorRelation != nil {
			t.Fatalf("匿名不应有关系: %+v", items[i].AuthorRelation)
		}
	}
}

// 读取失败不能让列表失败：静默保持 null。
func TestFillAuthorRelationsSwallowsError(t *testing.T) {
	reader := &fakeFollowRelationReader{err: errors.New("boom")}
	svc := NewServiceWithRepo(&stubMomentRepo{})
	svc.SetFollowRelationReader(reader)

	items := relItems()
	svc.fillAuthorRelations(context.Background(), items, "viewer") // 不得 panic
	for i := range items {
		if items[i].AuthorRelation != nil {
			t.Fatalf("出错时应保持 null: %+v", items[i].AuthorRelation)
		}
	}
}

// 序列化契约：author_relation 是**恒定键**（nil 时输出 null，而不是缺省），
// 且恒在 user 之外（不污染共享 UserInfo）。
func TestMomentItemAuthorRelationSerialization(t *testing.T) {
	raw, err := json.Marshal(MomentItem{MomentID: "m1", User: UserInfo{UserID: "u1"}})
	if err != nil {
		t.Fatal(err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatal(err)
	}
	if v, ok := asMap["author_relation"]; !ok || v != nil {
		t.Fatalf("恒定键 author_relation 应为 null: %s", raw)
	}
	if _, leaked := asMap["user"].(map[string]any)["relation"]; leaked {
		t.Fatalf("relation 不得出现在共享 UserInfo 上: %s", raw)
	}

	rel := UserRelationView{IsFollowing: true, IsFollowedBy: true}
	raw, err = json.Marshal(MomentItem{MomentID: "m1", User: UserInfo{UserID: "u1"}, AuthorRelation: &rel})
	if err != nil {
		t.Fatal(err)
	}
	asMap = map[string]any{}
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatal(err)
	}
	relMap, ok := asMap["author_relation"].(map[string]any)
	if !ok || relMap["is_following"] != true || relMap["is_followed_by"] != true {
		t.Fatalf("author_relation 形状不符: %s", raw)
	}
}

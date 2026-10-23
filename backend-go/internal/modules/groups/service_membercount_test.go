package groups

import (
	"testing"

	"hotspot/backend-go/internal/platform/realtime"
)

// TestJoinGroupBroadcastsRefreshedMemberCount 防回归（审查 P2-5）：
// GroupMemberJoined 的 member_count 取入群后的权威计数（memberCountAfter-
// Mutation 重读），而非「变更前快照 + 1」——并发入群/退群时快照+1 会把
// 错误值广播给全群。stub 的 Get 恒返回 MembersCount=5：旧实现广播 6，
// 新实现广播 5。
func TestJoinGroupBroadcastsRefreshedMemberCount(t *testing.T) {
	hub := withTestHub(t)
	repo := &stubGroupsRepo{
		getResult:  GroupItem{Group: GroupInfo{GroupID: "g1", MembersCount: 5}},
		getFound:   true,
		joinResult: EnterGroupResult{Status: EnterGroupStatusJoined},
	}
	svc := NewServiceWithRepo(repo)
	sink := &groupRealtimeSink{}
	hub.SetGroupSubscriptions([]string{"g1"}, sink)

	if _, err := svc.JoinGroup(grpCtx("u9"), EnterGroupRequest{GroupID: "g1"}); err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if len(sink.events) != 1 || sink.events[0].Type != realtime.EventTypeGroupMemberJoined {
		t.Fatalf("unexpected events: %#v", sink.events)
	}
	payload := decodePayload(t, sink.events[0].Payload)
	if got, ok := payload["member_count"].(float64); !ok || got != 5 {
		t.Fatalf("expected refreshed member_count=5, got %#v", payload["member_count"])
	}
}

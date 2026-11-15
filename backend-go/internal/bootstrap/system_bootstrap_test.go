package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/modules/system"
)

// stubNotifyUnreadCounter 是 notifyUnreadCounter 的桩：只返回预置的未读汇总，
// 使装配层单测无需构造完整 notify.Service / 仓储。
type stubNotifyUnreadCounter struct {
	result notify.UnreadSummaryResult
	err    error
}

func (s stubNotifyUnreadCounter) UnreadCount(context.Context) (notify.UnreadSummaryResult, error) {
	return s.result, s.err
}

// stubChatUnreadCounter 是 chatUnreadCounter 的桩：私聊/群聊两个域的聚合查询
// 分别注入，便于断言「单域失败只降级该域」与 total_badge 公式。
type stubChatUnreadCounter struct {
	count int
	err   error
}

func (s stubChatUnreadCounter) CountUnreadForUser(context.Context) (int, error) {
	return s.count, s.err
}

// notifyCategoryKeys 是 spec api_spec_types.md 定义的 NotifyCategory 全集。
var notifyCategoryKeys = []string{"social", "interaction", "group", "report", "nearby", "system"}

// 审查 C5：system/bootstrap 的 notify_by_type 必须覆盖 NotifyCategory 全部 6 类
// （此前只写 social/interaction/group/system 四键，report/nearby 静默丢失）；
// 第 81 轮遗留：私聊/群聊未读接上真实聚合值后，total_badge 按 spec 公式对三域
// 求和（此前聊天项恒 0，公式退化为通知总数）。
func TestResolveUnreadSummaryCoversAllNotifyCategories(t *testing.T) {
	resolver := systemBootstrapResolver{
		notify: stubNotifyUnreadCounter{result: notify.UnreadSummaryResult{
			Total:  9,
			ByType: notify.UnreadCountByType{Social: 1, Interaction: 2, Group: 1, Report: 2, Nearby: 1, System: 2},
		}},
		privateChats: stubChatUnreadCounter{count: 2},
		groupChats:   stubChatUnreadCounter{count: 3},
	}

	summary := resolver.ResolveUnreadSummary(context.Background())

	if summary.NotifyTotal != 9 {
		t.Fatalf("notify_total = %d want 9", summary.NotifyTotal)
	}
	wantByType := map[string]int{"social": 1, "interaction": 2, "group": 1, "report": 2, "nearby": 1, "system": 2}
	if !reflect.DeepEqual(summary.NotifyByType, wantByType) {
		t.Fatalf("notify_by_type = %#v want %#v", summary.NotifyByType, wantByType)
	}
	// 聊天未读取各自聚合查询的真实值，不再恒 0（spec api_spec_system.md）。
	if summary.PrivateChatUnreadCount != 2 || summary.GroupChatUnreadCount != 3 {
		t.Fatalf("chat unread = private %d / group %d, want 2/3", summary.PrivateChatUnreadCount, summary.GroupChatUnreadCount)
	}
	if summary.TotalBadge != 14 {
		t.Fatalf("total_badge = %d want 14 (9+2+3)", summary.TotalBadge)
	}
}

// notify 查询失败时降级为全零摘要，但 notify_by_type 仍须是带齐 6 个分类键的
// 对象：spec 的 notify_by_type 是 Record<NotifyCategory, number>，nil map 会
// 序列化成 null，客户端按分类取角标时直接失效。
func TestResolveUnreadSummaryKeepsCategoryKeysWhenNotifyFails(t *testing.T) {
	resolver := systemBootstrapResolver{
		notify:       stubNotifyUnreadCounter{err: errors.New("notify unavailable")},
		privateChats: stubChatUnreadCounter{count: 2},
		groupChats:   stubChatUnreadCounter{count: 3},
	}

	summary := resolver.ResolveUnreadSummary(context.Background())

	if summary.NotifyByType == nil {
		t.Fatal("notify_by_type must be an object with 6 keys, got nil")
	}
	if len(summary.NotifyByType) != len(notifyCategoryKeys) {
		t.Fatalf("notify_by_type = %#v want keys %v", summary.NotifyByType, notifyCategoryKeys)
	}
	for _, category := range notifyCategoryKeys {
		if count, ok := summary.NotifyByType[category]; !ok || count != 0 {
			t.Fatalf("notify_by_type[%q] = %d, ok = %v; want 0", category, count, ok)
		}
	}
	// 通知域失败只降级通知部分：聊天两域各自独立聚合，仍取真实值。
	if summary.NotifyTotal != 0 {
		t.Fatalf("notify_total = %d want 0 on notify failure", summary.NotifyTotal)
	}
	if summary.PrivateChatUnreadCount != 2 || summary.GroupChatUnreadCount != 3 {
		t.Fatalf("chat unread must not degrade with notify: %#v", summary)
	}
	if summary.TotalBadge != 5 {
		t.Fatalf("total_badge = %d want 5 (0+2+3)", summary.TotalBadge)
	}
}

// 单个聊天域聚合查询失败时只把该域降级为 0（Warn），另一域与 total_badge 公式
// 照常；provider 未注入（无 DB 装配/旧单测路径）同样按 0 处理，冷启动摘要不因
// 聊天聚合失败而不可用。
func TestResolveUnreadSummaryChatUnreadDegradesIndependently(t *testing.T) {
	resolver := systemBootstrapResolver{
		notify:       stubNotifyUnreadCounter{result: notify.UnreadSummaryResult{Total: 4}},
		privateChats: stubChatUnreadCounter{err: errors.New("private chat aggregate failed")},
		groupChats:   stubChatUnreadCounter{count: 4},
	}
	summary := resolver.ResolveUnreadSummary(context.Background())
	if summary.PrivateChatUnreadCount != 0 {
		t.Fatalf("private_chat_unread_count = %d want 0 on failure", summary.PrivateChatUnreadCount)
	}
	if summary.GroupChatUnreadCount != 4 || summary.TotalBadge != 8 {
		t.Fatalf("group unread = %d total_badge = %d, want 4/8", summary.GroupChatUnreadCount, summary.TotalBadge)
	}

	// 两个 provider 均未注入：聊天项保持 0（公式退化为通知总数）。
	bare := systemBootstrapResolver{notify: stubNotifyUnreadCounter{result: notify.UnreadSummaryResult{Total: 4}}}
	summary = bare.ResolveUnreadSummary(context.Background())
	if summary.PrivateChatUnreadCount != 0 || summary.GroupChatUnreadCount != 0 || summary.TotalBadge != 4 {
		t.Fatalf("nil chat providers must degrade to zero chat counts: %#v", summary)
	}
}

// total_badge 公式（spec api_spec_system.md）= notify_total +
// private_chat_unread_count + group_chat_unread_count，聊天聚合值必须计入
// （F10：各域之和、允许同一内容重复计数）。
func TestTotalBadgeSumsNotifyAndChatDomains(t *testing.T) {
	if got := totalBadge(4, 2, 3); got != 9 {
		t.Fatalf("totalBadge(4, 2, 3) = %d want 9", got)
	}
}

// 序列化契约：前端 types.ts 的 UnreadSummary 是固定字段 + 6 键 Record，
// JSON 必须带齐全部字段与分类键（缺键即破契约）。
func TestResolveUnreadSummaryJSONContract(t *testing.T) {
	resolver := systemBootstrapResolver{
		notify: stubNotifyUnreadCounter{result: notify.UnreadSummaryResult{
			Total:  2,
			ByType: notify.UnreadCountByType{Report: 1, Nearby: 1},
		}},
		privateChats: stubChatUnreadCounter{count: 6},
		groupChats:   stubChatUnreadCounter{count: 7},
	}
	raw, err := json.Marshal(resolver.ResolveUnreadSummary(context.Background()))
	if err != nil {
		t.Fatalf("marshal unread summary: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal unread summary: %v", err)
	}
	for _, key := range []string{"notify_total", "notify_by_type", "private_chat_unread_count", "group_chat_unread_count", "total_badge"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("JSON missing field %q: %s", key, raw)
		}
	}
	var byType map[string]int
	if err := json.Unmarshal(fields["notify_by_type"], &byType); err != nil {
		t.Fatalf("unmarshal notify_by_type: %v", err)
	}
	if len(byType) != len(notifyCategoryKeys) {
		t.Fatalf("notify_by_type = %#v want keys %v", byType, notifyCategoryKeys)
	}
	for _, category := range notifyCategoryKeys {
		if _, ok := byType[category]; !ok {
			t.Fatalf("notify_by_type missing category %q: %#v", category, byType)
		}
	}
	// 聊天未读字段必须是聚合真实值（number），且 total_badge 为三域之和。
	var summary system.UnreadSummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatalf("unmarshal unread summary as contract type: %v", err)
	}
	if summary.PrivateChatUnreadCount != 6 || summary.GroupChatUnreadCount != 7 || summary.TotalBadge != 15 {
		t.Fatalf("unexpected chat unread/total_badge: %#v", summary)
	}
}

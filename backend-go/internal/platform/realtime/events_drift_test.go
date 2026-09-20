package realtime

// 跨端 WS 事件集合一致性守卫（契约防漂移）。
//
// 后端 EventType* 常量是权威实现（对齐 spec api_spec_websocket.md 的
// ServerEventType 枚举）；frontend/src/lib/ws/types.ts 的 ServerEventType
// 联合与 harmonyapp .../WebSocketTypes.ets 的 ServerEventType 枚举各自手写。
// 历史上前端把 ContentMentioned 误写为 MessageMentioned（spec/后端均为前者），
// Web 端从此永远收不到瞬间/评论 @ 通知——三端各错各的，只能靠 review 发现。
// 本测试解析两个客户端源文件中的事件清单，与后端常量集合做严格相等断言，
// 把这类漂移变成 CI 拦截。事件改名/新增时：先改 spec 与后端，再同步两端，
// 本测试红即提醒尚未同步。

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// authoritativeEvents 后端权威事件集合。SubscriptionRevoked 仅用于 redis
// bridge 的跨实例控制通道（revoke:/ 前缀私有主题），永不下发客户端，
// 不在 spec ServerEventType 枚举内，故从对比中排除。
func authoritativeEvents() map[string]bool {
	events := []string{
		EventTypeConnectionEstablished, EventTypePong, EventTypeError,
		EventTypeSubscribeAck, EventTypeSubscriptionsSnapshot, EventTypeSessionRevoked,
		EventTypeUserOnline, EventTypeUserOffline,
		EventTypeNewFollower, EventTypeMutualFollow, EventTypeNearbyNewMoment,
		EventTypeMomentCommented, EventTypeCommentReply, EventTypeContentMentioned,
		EventTypeMomentUpdated, EventTypeMomentDeleted,
		EventTypePrivateMessageNew, EventTypePrivateMessageEdited, EventTypePrivateMessageRecalled,
		EventTypePrivateMessageRead, EventTypePrivateTyping,
		EventTypeGroupMessageNew, EventTypeGroupMessageEdited, EventTypeGroupMessageRecalled,
		EventTypeGroupMessageReadCountUpdated, EventTypeGroupMessageMentioned, EventTypeGroupTyping,
		EventTypeGroupMemberJoined, EventTypeGroupMemberLeft, EventTypeGroupMemberRemoved,
		EventTypeGroupMemberRoleChanged, EventTypeGroupMemberMuted, EventTypeGroupMemberUnmuted,
		EventTypeGroupInviteReceived, EventTypeGroupInviteRevoked, EventTypeGroupInviteRejected,
		EventTypeGroupInviteExpired, EventTypeGroupInviteResponse,
		EventTypeGroupApplicationReceived, EventTypeGroupApplicationCancelled,
		EventTypeGroupApplicationExpired, EventTypeGroupApplicationRejected, EventTypeGroupApplicationResponse,
		EventTypeGroupAnnouncementCreated, EventTypeGroupUpdated, EventTypeGroupDismissed,
		EventTypeReactionAdded, EventTypeReactionRemoved,
		EventTypeReportResolved, EventTypeReportDismissed,
		EventTypeUserProfileUpdated, EventTypeSystemAnnouncement, EventTypeAccountBanned,
	}
	set := make(map[string]bool, len(events))
	for _, e := range events {
		if set[e] {
			panic("duplicate event constant in authoritative list: " + e)
		}
		set[e] = true
	}
	return set
}

// repoRoot 定位仓库根（本文件位于 backend-go/internal/platform/realtime/，
// 向上四级）。用 runtime 路径而非相对 cwd，保证 `go test ./...` 与 IDE
// 单测两种入口都成立。
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法获取测试文件路径")
	}
	root := filepath.Dir(thisFile)
	for i := 0; i < 4; i++ {
		root = filepath.Dir(root)
	}
	// 防呆：找不到标识文件说明目录层级假设失效，报清晰错误而非解析空文件
	if _, err := os.Stat(filepath.Join(root, "spec", "api_spec_websocket.md")); err != nil {
		t.Fatalf("仓库根定位失败（%s）：%v", root, err)
	}
	return root
}

// extractBlock 从源码中截取 anchor 之后到 terminator 首次出现的片段。
func extractBlock(t *testing.T, src, anchor, terminator string) string {
	t.Helper()
	start := strings.Index(src, anchor)
	if start < 0 {
		t.Fatalf("源码中找不到锚点 %q（客户端事件清单结构变更时同步更新本测试的解析）", anchor)
	}
	seg := src[start:]
	end := strings.Index(seg, terminator)
	if end < 0 {
		t.Fatalf("锚点 %q 之后找不到终止符 %q", anchor, terminator)
	}
	return seg[:end]
}

var (
	feUnionEntry = regexp.MustCompile(`\|\s*'([A-Za-z]+)'`)
	hosEnumEntry = regexp.MustCompile(`=\s*'([A-Za-z]+)'`)
)

// readClientSource 读取客户端源文件；缺失视为测试环境不完整，直接失败
// （CI 检出完整仓库，本地缺失说明仓库不完整）。
func readClientSource(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("读取客户端源文件 %s 失败：%v", rel, err)
	}
	return string(b)
}

// assertSetEqual 对比权威集合与客户端集合，双向差异以多行形式一次性报出，
// 便于一次看到全部待同步项。
func assertSetEqual(t *testing.T, client string, got, want map[string]bool) {
	t.Helper()
	var missingInClient, extraInClient []string
	for e := range want {
		if !got[e] {
			missingInClient = append(missingInClient, e)
		}
	}
	for e := range got {
		if !want[e] {
			extraInClient = append(extraInClient, e)
		}
	}
	if len(missingInClient) == 0 && len(extraInClient) == 0 {
		return
	}
	sort.Strings(missingInClient)
	sort.Strings(extraInClient)
	t.Fatalf("WS 事件集合漂移（%s 与后端不一致，请对照 spec/api_spec_websocket.md 同步）:\n  客户端缺失: %v\n  客户端多出: %v",
		client, missingInClient, extraInClient)
}

func TestServerEventTypeSetsMatchFrontend(t *testing.T) {
	src := readClientSource(t, filepath.Join("frontend", "src", "lib", "ws", "types.ts"))
	block := extractBlock(t, src, "export type ServerEventType", ";")
	got := map[string]bool{}
	for _, m := range feUnionEntry.FindAllStringSubmatch(block, -1) {
		got[m[1]] = true
	}
	if len(got) == 0 {
		t.Fatal("前端 ServerEventType 解析结果为空——union 书写结构变更时同步更新解析")
	}
	assertSetEqual(t, "frontend/src/lib/ws/types.ts", got, authoritativeEvents())
}

func TestServerEventTypeSetsMatchHarmony(t *testing.T) {
	src := readClientSource(t, filepath.Join("harmonyapp", "entry", "src", "main", "ets", "core", "types", "WebSocketTypes.ets"))
	block := extractBlock(t, src, "export enum ServerEventType", "\n}")
	got := map[string]bool{}
	for _, m := range hosEnumEntry.FindAllStringSubmatch(block, -1) {
		got[m[1]] = true
	}
	if len(got) == 0 {
		t.Fatal("鸿蒙 ServerEventType 解析结果为空——enum 书写结构变更时同步更新解析")
	}
	assertSetEqual(t, "harmonyapp .../WebSocketTypes.ets", got, authoritativeEvents())
}

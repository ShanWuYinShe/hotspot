package groups

import (
	"context"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
)

func repeatRunes(n int) string { return strings.Repeat("字", n) }

// 契约（spec api_spec_base.md「通用字段约束」/「群组字段约束」/「公告字段约束」）
// 此前只校验非空，长度上限缺失。以下用例逐字段锁定边界（按 rune 计数）。
func TestValidateCreateGroupRequestBounds(t *testing.T) {
	validLocation := func() Location {
		return Location{LocationName: "南天门", Point: GeoPoint{Latitude: 35.1, Longitude: 112.6}}
	}
	valid := CreateGroupRequest{Name: "登山队", Location: validLocation()}
	if apiErr := validateCreateGroupRequest(valid); apiErr != nil {
		t.Fatalf("minimal request should pass: %v", apiErr)
	}

	cases := []struct {
		name string
		req  CreateGroupRequest
	}{
		{"名称超长", CreateGroupRequest{Name: repeatRunes(groupNameMaxRunes + 1), Location: validLocation()}},
		{"描述超长", CreateGroupRequest{Name: "g", Description: repeatRunes(groupDescriptionMaxRunes + 1), Location: validLocation()}},
		{"标签超数量", CreateGroupRequest{Name: "g", Tags: make([]string, groupTagsMaxCount+1), Location: validLocation()}},
		{"标签超长", CreateGroupRequest{Name: "g", Tags: []string{repeatRunes(groupTagMaxRunes + 1)}, Location: validLocation()}},
		{"地点超长", CreateGroupRequest{Name: "g", Location: Location{LocationName: repeatRunes(groupLocationNameMaxRunes + 1)}}},
		{"密码过短", CreateGroupRequest{Name: "g", Password: "1234567", Location: validLocation()}},
		{"密码过长", CreateGroupRequest{Name: "g", Password: repeatRunes(groupPasswordMaxRunes + 1), Location: validLocation()}},
		{"人数下限", CreateGroupRequest{Name: "g", MaxMembers: 1, Location: validLocation()}},
		{"人数上限", CreateGroupRequest{Name: "g", MaxMembers: groupMaxMembersMax + 1, Location: validLocation()}},
	}
	for _, tc := range cases {
		if apiErr := validateCreateGroupRequest(tc.req); apiErr == nil {
			t.Fatalf("%s: expected invalid", tc.name)
		}
	}

	// 边界值合法：名 50 / 描述 500 / 标签 10 个各 30 / 地点 100 / 密码 8 与 32 /
	// 人数 2 与 2000。
	boundary := CreateGroupRequest{
		Name:        repeatRunes(groupNameMaxRunes),
		Description: repeatRunes(groupDescriptionMaxRunes),
		Tags:        []string{repeatRunes(groupTagMaxRunes)},
		Password:    repeatRunes(groupPasswordMinRunes),
		MaxMembers:  groupMaxMembersMax,
		Location:    Location{LocationName: repeatRunes(groupLocationNameMaxRunes), Point: GeoPoint{Latitude: 35.1, Longitude: 112.6}},
	}
	if apiErr := validateCreateGroupRequest(boundary); apiErr != nil {
		t.Fatalf("boundary request should pass: %v", apiErr)
	}
}

func TestValidateUpdateGroupRequestBounds(t *testing.T) {
	if apiErr := validateUpdateGroupRequest(UpdateGroupRequest{GroupID: "g1"}); apiErr != nil {
		t.Fatalf("empty update should pass: %v", apiErr)
	}
	cases := []struct {
		name string
		req  UpdateGroupRequest
	}{
		{"名称超长", UpdateGroupRequest{GroupID: "g1", Name: repeatRunes(groupNameMaxRunes + 1)}},
		{"描述超长", UpdateGroupRequest{GroupID: "g1", Description: repeatRunes(groupDescriptionMaxRunes + 1)}},
		{"标签超数量", UpdateGroupRequest{GroupID: "g1", Tags: make([]string, groupTagsMaxCount+1)}},
		{"密码过短", UpdateGroupRequest{GroupID: "g1", Password: strPtr("1234567")}},
		{"人数非法", UpdateGroupRequest{GroupID: "g1", MaxMembers: groupIntPtr(1)}},
	}
	for _, tc := range cases {
		if apiErr := validateUpdateGroupRequest(tc.req); apiErr == nil {
			t.Fatalf("%s: expected invalid", tc.name)
		}
	}
	okName := repeatRunes(groupNameMaxRunes)
	okDesc := repeatRunes(groupDescriptionMaxRunes)
	okPwd := repeatRunes(groupPasswordMaxRunes)
	okMax := groupMaxMembersMin
	if apiErr := validateUpdateGroupRequest(UpdateGroupRequest{
		GroupID: "g1", Name: okName, Description: okDesc, Password: &okPwd, MaxMembers: &okMax,
	}); apiErr != nil {
		t.Fatalf("boundary update should pass: %v", apiErr)
	}
}

func TestValidateGroupAnnouncementContentBounds(t *testing.T) {
	title := repeatRunes(groupAnnouncementTitleMax)
	content := repeatRunes(groupAnnouncementContentMax)
	if apiErr := validateGroupAnnouncementContent(&title, &content); apiErr != nil {
		t.Fatalf("boundary announcement should pass: %v", apiErr)
	}
	longTitle := repeatRunes(groupAnnouncementTitleMax + 1)
	if apiErr := validateGroupAnnouncementContent(&longTitle, nil); apiErr == nil {
		t.Fatal("title over limit should be rejected")
	}
	longContent := repeatRunes(groupAnnouncementContentMax + 1)
	if apiErr := validateGroupAnnouncementContent(nil, &longContent); apiErr == nil {
		t.Fatal("content over limit should be rejected")
	}
	// 非空由 Handler 校验（required），Service 只负责长度上限。
	if apiErr := validateGroupAnnouncementContent(nil, nil); apiErr != nil {
		t.Fatalf("nil fields (no change) should pass: %v", apiErr)
	}
}

func TestValidateGroupMessageBounds(t *testing.T) {
	ok := repeatRunes(groupInviteMessageMaxRunes)
	if apiErr := validateGroupMessage(&ok); apiErr != nil {
		t.Fatalf("boundary message should pass: %v", apiErr)
	}
	long := repeatRunes(groupInviteMessageMaxRunes + 1)
	if apiErr := validateGroupMessage(&long); apiErr == nil {
		t.Fatal("message over limit should be rejected")
	}
	if apiErr := validateGroupMessage(nil); apiErr != nil {
		t.Fatalf("nil message should pass: %v", apiErr)
	}
}

// 服务入口接线：校验必须在任何 repo/DB 访问之前生效（nil repo 也能安全返回 10000）。
func TestGroupServiceValidatesBeforeRepo(t *testing.T) {
	svc := &Service{}
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_1"})

	assertInvalid := func(label string, err error) {
		t.Helper()
		apiErr, ok := err.(*httpx.APIError)
		if !ok || apiErr.Code != httpx.CodeInvalidRequest {
			t.Fatalf("%s: expected 10000, got %v", label, err)
		}
	}

	_, err := svc.Create(ctx, CreateGroupRequest{Name: repeatRunes(groupNameMaxRunes + 1)})
	assertInvalid("create", err)

	_, err = svc.Update(ctx, UpdateGroupRequest{GroupID: "g1", Description: repeatRunes(groupDescriptionMaxRunes + 1)})
	assertInvalid("update", err)

	_, err = svc.CreateAnnouncement(ctx, CreateGroupAnnouncementRequest{GroupID: "g1", Title: "t", Content: repeatRunes(groupAnnouncementContentMax + 1)})
	assertInvalid("create_announcement", err)

	_, err = svc.UpdateAnnouncement(ctx, UpdateGroupAnnouncementRequest{GroupID: "g1", Title: strPtr(repeatRunes(groupAnnouncementTitleMax + 1))})
	assertInvalid("update_announcement", err)

	_, err = svc.InviteMember(ctx, InviteMemberRequest{GroupID: "g1", UserID: "u2", Message: strPtr(repeatRunes(groupInviteMessageMaxRunes + 1))})
	assertInvalid("invite", err)

	_, err = svc.JoinGroup(ctx, EnterGroupRequest{GroupID: "g1", Message: strPtr(repeatRunes(groupJoinMessageMaxRunes + 1))})
	assertInvalid("join_group", err)
}

func groupIntPtr(n int) *int { return &n }

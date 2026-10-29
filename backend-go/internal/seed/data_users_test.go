package seed

import (
	"regexp"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/geo"
)

// data_users_test.go — 用户/账号/关系/S2 一致性。

func TestSeedUsersAndAccounts(t *testing.T) {
	data := buildSeedData()
	if len(data.users) != 10 {
		t.Fatalf("用户数 = %d，期望 10", len(data.users))
	}
	userIDs := map[string]userSpec{}
	for _, user := range data.users {
		if _, ok := userIDs[user.UserID]; ok {
			t.Fatalf("用户 ID 重复: %s", user.UserID)
		}
		if !strings.HasPrefix(user.UserID, userIDPrefix) {
			t.Errorf("用户 ID 缺少固定前缀: %s", user.UserID)
		}
		if user.Nickname == "" || user.LocationName == "" {
			t.Errorf("用户 %s 缺少昵称/位置名", user.UserID)
		}
		userIDs[user.UserID] = user
	}
	for _, wantID := range []string{
		"user_seed_lumos", "user_seed_admin", "user_seed_privacy", "user_seed_banned",
		"user_seed_sophia", "user_seed_marco", "user_seed_mika", "user_seed_chen",
		"user_seed_lin", "user_seed_xiaoman",
	} {
		if _, ok := userIDs[wantID]; !ok {
			t.Errorf("缺少用户 %s", wantID)
		}
	}
	if got := userIDs["user_seed_lumos"].Nickname; got != "鹿鸣" {
		t.Errorf("lumos 昵称 = %q，期望 鹿鸣", got)
	}
	if got := userIDs["user_seed_admin"].Nickname; got != "巡检员" {
		t.Errorf("admin 昵称 = %q，期望 巡检员", got)
	}
	privacy := userIDs["user_seed_privacy"].Settings
	if privacy.PrivacyLevel != "private" || privacy.LocationVisible || privacy.PresenceVisible || privacy.Searchable {
		t.Errorf("隐私用户设置不符合规格: %+v", privacy)
	}
	banned := userIDs["user_seed_banned"]
	if banned.Status != "banned" || banned.BannedReason == "" || banned.BannedDays <= 0 {
		t.Errorf("封禁用户状态不完整: %+v", banned)
	}
	if !userIDs["user_seed_lumos"].Settings.NearbyMomentNotify {
		t.Errorf("lumos 应开启 nearby_moment_notify（覆盖附近新瞬间通知链路）")
	}

	// 账号：每个用户都有登录账号；两个 QA 账号映射正确。
	accountByUser := map[string]accountSpec{}
	emailLike := 0
	for _, account := range data.accounts {
		accountByUser[account.UserID] = account
		if strings.Contains(account.LoginID, "@") && len(account.LoginID) >= 3 && len(account.LoginID) <= 64 {
			emailLike++
		}
	}
	if emailLike != len(data.accounts) {
		t.Errorf("存在非邮箱型 login_id: %d/%d", emailLike, len(data.accounts))
	}
	if got := accountByUser["user_seed_lumos"].LoginID; got != "qa@hotspot.local" {
		t.Errorf("QA 账号 login_id = %q", got)
	}
	if got := accountByUser["user_seed_admin"].LoginID; got != "admin@hotspot.local" {
		t.Errorf("管理员账号 login_id = %q", got)
	}
	// 口令长度/字符契约（Auth 校验：8-128 且含字母+数字）。
	hasLetter := regexp.MustCompile("[A-Za-z]")
	hasDigit := regexp.MustCompile("[0-9]")
	for _, plain := range []string{defaultQAPassword, defaultAdminPassword, legacyAccountPassword, groupSeedPassword} {
		if len(plain) < 8 || len(plain) > 128 {
			t.Errorf("口令长度不符合 8-128: %q", plain)
		}
		if !hasLetter.MatchString(plain) || !hasDigit.MatchString(plain) {
			t.Errorf("口令必须同时包含字母与数字: %q", plain)
		}
	}
}

// TestSeedUserS2Cells 用户坐标与 S2 cell 必须真实计算（否则附近/地图为空）。
func TestSeedUserS2Cells(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	rows := rowsOfTable(t, plan, "users")
	columns := columnsOfTable(t, plan, "users")
	latIndex := columnIndex(t, columns, "latitude")
	lngIndex := columnIndex(t, columns, "longitude")
	s2Index := columnIndex(t, columns, "s2_cell_id")
	if len(rows) != 10 {
		t.Fatalf("users 行数 = %d", len(rows))
	}
	for _, row := range rows {
		point := geo.Point{Latitude: mustFloat(t, row[latIndex]), Longitude: mustFloat(t, row[lngIndex])}
		want := geo.LeafCellID(point)
		s2, ok := row[s2Index].(int64)
		if !ok {
			t.Fatalf("用户 s2_cell_id 参数类型 = %T，期望 int64", row[s2Index])
		}
		if want == 0 || s2 != want {
			t.Errorf("用户 %s s2_cell_id = %d，期望 %d（geo.LeafCellID 口径）", mustString(t, row[0]), s2, want)
		}
	}
}

func TestSeedRelations(t *testing.T) {
	data := buildSeedData()
	following := map[string]map[string]bool{}
	followers := map[string]map[string]bool{}
	blocked := map[string]map[string]bool{}
	for _, relation := range data.relations {
		if relation.Following {
			if following[relation.From] == nil {
				following[relation.From] = map[string]bool{}
			}
			following[relation.From][relation.To] = true
			if followers[relation.To] == nil {
				followers[relation.To] = map[string]bool{}
			}
			followers[relation.To][relation.From] = true
		}
		if relation.Blocked {
			if blocked[relation.From] == nil {
				blocked[relation.From] = map[string]bool{}
			}
			blocked[relation.From][relation.To] = true
		}
	}
	for _, wantID := range []string{"user_seed_sophia", "user_seed_marco", "user_seed_mika", "user_seed_xiaoman", "user_seed_chen", "user_seed_admin"} {
		if !following["user_seed_lumos"][wantID] {
			t.Errorf("lumos 应关注 %s", wantID)
		}
	}
	if following["user_seed_lumos"]["user_seed_lin"] {
		t.Errorf("lumos 不应关注 lin（lin 单向关注 lumos）")
	}
	for _, friendID := range []string{"user_seed_sophia", "user_seed_marco", "user_seed_mika", "user_seed_xiaoman", "user_seed_admin"} {
		if !following["user_seed_lumos"][friendID] || !following[friendID]["user_seed_lumos"] {
			t.Errorf("lumos 与 %s 应为互关", friendID)
		}
	}
	if !followers["user_seed_lumos"]["user_seed_lin"] {
		t.Errorf("lin 应关注 lumos（粉丝）")
	}
	if !blocked["user_seed_lumos"]["user_seed_privacy"] {
		t.Errorf("lumos 应拉黑 privacy")
	}
}

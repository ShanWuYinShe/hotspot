package seed

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/geo"
)

// data_moments_test.go — 瞬间规格一致性：作者轮换、图片桶、位置距离、计数列、
// 特殊状态（软删/私密）、S2 口径、标签与 emoji 白名单。

func TestMomentAuthorRotation(t *testing.T) {
	data := buildSeedData()
	if len(data.moments) != 24 {
		t.Fatalf("瞬间数 = %d，期望 24", len(data.moments))
	}
	wantAuthors := map[string]int{
		"user_seed_lumos": 6, "user_seed_sophia": 4, "user_seed_marco": 4,
		"user_seed_mika": 3, "user_seed_chen": 2, "user_seed_lin": 2,
		"user_seed_xiaoman": 2, "user_seed_admin": 1,
	}
	gotAuthors := map[string]int{}
	for index, moment := range data.moments {
		wantID := momentIDPrefix + twoDigits(index+1)
		if moment.MomentID != wantID {
			t.Errorf("瞬间第 %d 条 ID = %s，期望 %s", index+1, moment.MomentID, wantID)
		}
		gotAuthors[moment.AuthorID]++
	}
	for author, want := range wantAuthors {
		if gotAuthors[author] != want {
			t.Errorf("作者 %s 瞬间数 = %d，期望 %d", author, gotAuthors[author], want)
		}
	}
}

// TestMomentImageBuckets 图片桶：0 图 10 条、1 图 8 条、2 图 4 条、3 图 2 条（22 图位）。
func TestMomentImageBuckets(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	data := buildSeedData()
	counts := map[int]int{}
	totalImages := 0
	for _, moment := range data.moments {
		counts[moment.ImageCount]++
		totalImages += moment.ImageCount
	}
	if counts[0] != 10 || counts[1] != 8 || counts[2] != 4 || counts[3] != 2 {
		t.Fatalf("图片桶 = %v，期望 0:10 / 1:8 / 2:4 / 3:2", counts)
	}
	if totalImages != 22 {
		t.Fatalf("瞬间图片总数 = %d，期望 22", totalImages)
	}
	if got := len(rowsOfTable(t, plan, "moment_images")); got != totalImages {
		t.Fatalf("moment_images 行数 = %d，期望 %d", got, totalImages)
	}
}

func TestMomentLocationsWithinRange(t *testing.T) {
	for _, moment := range buildSeedData().moments {
		point := geo.Point{Latitude: moment.Latitude, Longitude: moment.Longitude}
		if !geo.IsValidPoint(point) {
			t.Errorf("瞬间 %s 坐标非法: %+v", moment.MomentID, point)
			continue
		}
		distance := geo.DistanceMeters(lumosPoint, point)
		if distance < 100 || distance > 3000 {
			t.Errorf("瞬间 %s 距 lumos %dm，期望 100–3000m", moment.MomentID, distance)
		}
	}
}

// TestMomentCounterColumns 计数列必须与真实行数一致。
func TestMomentCounterColumns(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	data := buildSeedData()

	reactionCounts := map[string]int{}
	for _, row := range rowsOfTable(t, plan, "reactions") {
		if mustString(t, row[0]) == "moment" {
			reactionCounts[mustString(t, row[1])]++
		}
	}
	commentCounts := map[string]int{}
	for _, row := range rowsOfTable(t, plan, "comments") {
		commentCounts[mustString(t, row[1])]++
	}
	shareCounts := map[string]int{}
	for _, share := range data.shares {
		if share.ShareType == "moment" {
			shareCounts[mustString(t, share.Payload["shared_moment_id"])]++
		}
	}

	rows := rowsOfTable(t, plan, "moments")
	columns := columnsOfTable(t, plan, "moments")
	idIndex := columnIndex(t, columns, "moment_id")
	likesIndex := columnIndex(t, columns, "likes_count")
	commentsIndex := columnIndex(t, columns, "comments_count")
	sharesIndex := columnIndex(t, columns, "shares_count")
	for _, row := range rows {
		momentID := mustString(t, row[idIndex])
		if got, want := mustInt(t, row[likesIndex]), reactionCounts[momentID]; got != want {
			t.Errorf("瞬间 %s likes_count = %d，reactions 行数 = %d", momentID, got, want)
		}
		if got, want := mustInt(t, row[commentsIndex]), commentCounts[momentID]; got != want {
			t.Errorf("瞬间 %s comments_count = %d，comments 行数 = %d", momentID, got, want)
		}
		if got, want := mustInt(t, row[sharesIndex]), shareCounts[momentID]; got != want {
			t.Errorf("瞬间 %s shares_count = %d，shares 行数 = %d", momentID, got, want)
		}
	}
}

// TestCommentRepliesAndLikes 评论 replies_count / likes_count 与子行一致，且存在二级回复。
func TestCommentRepliesAndLikes(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	rows := rowsOfTable(t, plan, "comments")
	columns := columnsOfTable(t, plan, "comments")
	idIndex := columnIndex(t, columns, "comment_id")
	parentIndex := columnIndex(t, columns, "parent_id")
	repliesIndex := columnIndex(t, columns, "replies_count")
	likesIndex := columnIndex(t, columns, "likes_count")

	replyCounts := map[string]int{}
	for _, row := range rows {
		if parent, ok := row[parentIndex].(string); ok && parent != "" {
			replyCounts[parent]++
		}
	}
	likeCounts := map[string]int{}
	for _, row := range rowsOfTable(t, plan, "comment_likes") {
		likeCounts[mustString(t, row[0])]++
	}
	replyTotal := 0
	for _, row := range rows {
		commentID := mustString(t, row[idIndex])
		if got, want := mustInt(t, row[repliesIndex]), replyCounts[commentID]; got != want {
			t.Errorf("评论 %s replies_count = %d，子回复 = %d", commentID, got, want)
		}
		if got, want := mustInt(t, row[likesIndex]), likeCounts[commentID]; got != want {
			t.Errorf("评论 %s likes_count = %d，comment_likes 行 = %d", commentID, got, want)
		}
		replyTotal += replyCounts[commentID]
	}
	if replyTotal == 0 {
		t.Fatalf("评论中没有二级回复（规格要求含 2 级回复）")
	}
}

// TestMomentSpecialStates 软删 / 私密 / favorites / mentions 覆盖。
func TestMomentSpecialStates(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	rows := rowsOfTable(t, plan, "moments")
	columns := columnsOfTable(t, plan, "moments")
	idIndex := columnIndex(t, columns, "moment_id")
	descriptionIndex := columnIndex(t, columns, "description")
	deletedIndex := columnIndex(t, columns, "is_deleted")
	privacyIndex := columnIndex(t, columns, "privacy_level")

	byID := map[string][]any{}
	for _, row := range rows {
		byID[mustString(t, row[idIndex])] = row
	}
	deleted := byID["moment_seed_23"]
	if deleted == nil {
		t.Fatal("缺少 moment_seed_23")
	}
	if !mustBool(t, deleted[deletedIndex]) || mustString(t, deleted[descriptionIndex]) != "" {
		t.Errorf("moment_seed_23 应为软删且 description 清空")
	}
	private := byID["moment_seed_24"]
	if private == nil {
		t.Fatal("缺少 moment_seed_24")
	}
	if mustString(t, private[privacyIndex]) != "private" {
		t.Errorf("moment_seed_24 privacy_level = %q，期望 private", private[privacyIndex])
	}
	if got := len(rowsOfTable(t, plan, "moment_favorites")); got != 2 {
		t.Errorf("moment_favorites 行数 = %d，期望 2", got)
	}
	mentioned := map[string]bool{}
	for _, row := range rowsOfTable(t, plan, "moment_mentions") {
		mentioned[mustString(t, row[0])] = true
	}
	if len(mentioned) != 2 {
		t.Errorf("moment_mentions 涉及瞬间数 = %d，期望 2", len(mentioned))
	}
}

func TestMomentTagsNotBlocked(t *testing.T) {
	data := buildSeedData()
	blocked := map[string]bool{}
	for _, tag := range data.blockedTags {
		blocked[tag.Name] = true
	}
	for _, moment := range data.moments {
		// 软删瞬间专门用于管理端标签管理页展示已屏蔽标签（不出现在任何用户视图）。
		if moment.IsDeleted {
			continue
		}
		for _, tag := range moment.Tags {
			if blocked[tag] {
				t.Errorf("瞬间 %s 使用了被屏蔽标签 %q", moment.MomentID, tag)
			}
		}
	}
}

// TestReactionEmojisInWhitelist 反应 emoji 必须在 AppConfig 白名单内。
func TestReactionEmojisInWhitelist(t *testing.T) {
	limits := appconfig.NewStaticSource(config.Config{}).Get(context.Background()).Limits
	allowed := map[string]bool{}
	for _, emoji := range limits.ReactionEmojis {
		allowed[emoji] = true
	}
	for _, moment := range buildSeedData().moments {
		for _, reaction := range moment.Reactions {
			if !allowed[reaction.Emoji] {
				t.Errorf("瞬间 %s 使用了白名单外 emoji %q", moment.MomentID, reaction.Emoji)
			}
		}
	}
}

// TestMomentS2Cells 瞬间 s2_cell_id 必须是 geo 真实口径（附近瞬间查询依赖）。
func TestMomentS2Cells(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	rows := rowsOfTable(t, plan, "moments")
	columns := columnsOfTable(t, plan, "moments")
	latIndex := columnIndex(t, columns, "latitude")
	lngIndex := columnIndex(t, columns, "longitude")
	s2Index := columnIndex(t, columns, "s2_cell_id")
	for _, row := range rows {
		point := geo.Point{Latitude: mustFloat(t, row[latIndex]), Longitude: mustFloat(t, row[lngIndex])}
		want := geo.LeafCellID(point)
		s2, ok := row[s2Index].(int64)
		if !ok {
			t.Fatalf("瞬间 s2_cell_id 参数类型 = %T，期望 int64", row[s2Index])
		}
		if want == 0 || s2 != want {
			t.Errorf("瞬间 %s s2_cell_id = %d，期望 %d", mustString(t, row[0]), s2, want)
		}
	}
}

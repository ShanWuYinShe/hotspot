package seed

import (
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/geo"
)

// data_moments.go — 瞬间相关表行构建（moments / moment_images / moment_tags /
// moment_mentions / moment_likes / moment_favorites / comments / comment_likes /
// reactions）。
//
// 计数列口径（与写路径一致，避免前端出现「显示 3 赞但详情只有 1 个」）：
//   - likes_count = 该瞬间在 reactions 表中的行数（迁移/模块装配后的自愈口径，
//     见 moments/module.go ensureReactionCountBackfill）；
//   - comments_count = 该瞬间 comments 行数（含二级回复，comments 写路径每新增
//     一条就 +1）；
//   - replies_count = 直接回复该评论的行数；
//   - shares_count = 指向该瞬间的 shares 行数（分享创建时 +1，撤销不回减）。

func buildMomentsTable(baseTime time.Time, moments []momentSpec, shares []shareSpec) tableData {
	momentShareCounts := make(map[string]int, len(moments))
	for _, share := range shares {
		if share.ShareType != "moment" {
			continue
		}
		if momentID, ok := share.Payload["shared_moment_id"].(string); ok {
			momentShareCounts[momentID]++
		}
	}
	table := tableData{
		Table: "moments",
		Columns: []column{
			col("moment_id"), col("user_id"), col("description"), col("created_at"),
			col("likes_count"), col("comments_count"), col("shares_count"), col("location_name"),
			col("latitude"), col("longitude"), col("privacy_level"), col("is_deleted"), col("s2_cell_id"),
		},
	}
	for _, moment := range moments {
		description := moment.Description
		if moment.IsDeleted {
			// 迁移 037 语义：软删瞬间 description 置空，客户端展示「内容已删除」占位。
			description = ""
		}
		s2CellID := geo.LeafCellID(geo.Point{Latitude: moment.Latitude, Longitude: moment.Longitude})
		if s2CellID == 0 {
			panic(fmt.Sprintf("瞬间 %s 坐标非法", moment.MomentID))
		}
		table.addRow(
			moment.MomentID, moment.AuthorID, description, baseTime.Add(-moment.CreatedAgo),
			len(moment.Reactions), len(moment.Comments), momentShareCounts[moment.MomentID],
			moment.LocationName, moment.Latitude, moment.Longitude, moment.PrivacyLevel,
			moment.IsDeleted, s2CellID,
		)
	}
	return table
}

func buildMomentImagesTable(baseTime time.Time, moments []momentSpec, media *mediaBundle) tableData {
	table := tableData{
		Table:   "moment_images",
		Columns: []column{col("moment_id"), col("file_id"), col("position"), col("created_at")},
	}
	for _, moment := range moments {
		fileIDs := media.momentImageFileIDs[moment.MomentID]
		if len(fileIDs) != moment.ImageCount {
			panic(fmt.Sprintf("瞬间 %s 图片位 %d 与媒体计划 %d 不一致", moment.MomentID, moment.ImageCount, len(fileIDs)))
		}
		createdAt := baseTime.Add(-moment.CreatedAgo)
		for position, fileID := range fileIDs {
			table.addRow(moment.MomentID, fileID, position, createdAt)
		}
	}
	return table
}

func buildMomentTagsTable(baseTime time.Time, moments []momentSpec) tableData {
	table := tableData{
		Table:   "moment_tags",
		Columns: []column{col("moment_id"), col("tag_name"), col("created_at")},
	}
	for _, moment := range moments {
		createdAt := baseTime.Add(-moment.CreatedAgo)
		for _, tag := range moment.Tags {
			table.addRow(moment.MomentID, tag, createdAt)
		}
	}
	return table
}

func buildMomentMentionsTable(baseTime time.Time, moments []momentSpec) tableData {
	table := tableData{
		Table:   "moment_mentions",
		Columns: []column{col("moment_id"), col("mentioned_user_id"), col("created_at")},
	}
	for _, moment := range moments {
		createdAt := baseTime.Add(-moment.CreatedAgo)
		for _, userID := range moment.Mentions {
			table.addRow(moment.MomentID, userID, createdAt)
		}
	}
	return table
}

// buildMomentInteractionRows 构建 moment_likes 与 reactions 两张表。
//
// reactions 是表情反应（迁移 031，聚合读取口径）；moment_likes 是历史点赞表
// （当前代码已无读取方，保留写入是为了数据完整性与后续兼容）。两者都按
// momentSpec.Reactions 生成，保证 likes_count 与 reactions 行数严格一致。
func buildMomentInteractionRows(baseTime time.Time, moments []momentSpec) (momentLikes tableData, reactions tableData) {
	momentLikes = tableData{
		Table:   "moment_likes",
		Columns: []column{col("moment_id"), col("user_id"), col("liked_at")},
	}
	reactions = tableData{
		Table:   "reactions",
		Columns: []column{col("target_type"), col("target_id"), col("user_id"), col("emoji"), col("reacted_at")},
	}
	for _, moment := range moments {
		createdAt := baseTime.Add(-moment.CreatedAgo)
		for index, reaction := range moment.Reactions {
			reactedAt := createdAt.Add(time.Duration(index+1) * 5 * time.Minute)
			momentLikes.addRow(moment.MomentID, reaction.UserID, reactedAt)
			reactions.addRow("moment", moment.MomentID, reaction.UserID, reaction.Emoji, reactedAt)
		}
	}
	// 评论/消息上的表情反应（覆盖 reaction 的三种 target_type；不影响任何计数列）。
	reactions.addRow("comment", "comment_seed_01_01", "user_seed_marco", emojiThumb, baseTime.Add(-90*time.Minute))
	reactions.addRow("message", "chat_seed_sophia_01", "user_seed_lumos", emojiHeart, baseTime.Add(-47*time.Hour))
	reactions.addRow("message", "gmsg_seed_10", "user_seed_sophia", emojiThumb, baseTime.Add(-40*time.Minute))
	return momentLikes, reactions
}

func buildMomentFavoritesTable(baseTime time.Time) tableData {
	table := tableData{
		Table:   "moment_favorites",
		Columns: []column{col("user_id"), col("moment_id"), col("favorited_at")},
	}
	// 规格 §3.2：lumos 收藏 moment_seed_01 / moment_seed_05。
	table.addRow("user_seed_lumos", "moment_seed_01", baseTime.Add(-3*time.Hour))
	table.addRow("user_seed_lumos", "moment_seed_05", baseTime.Add(-20*time.Hour))
	return table
}

func buildCommentsTables(baseTime time.Time, moments []momentSpec) (topLevel tableData, replies tableData) {
	topLevel = tableData{
		Table: "comments",
		Columns: []column{
			col("comment_id"), col("moment_id"), col("user_id"), col("parent_id"),
			col("content"), col("created_at"), col("likes_count"), col("replies_count"),
		},
	}
	replies = tableData{
		Table: "comments",
		Columns: []column{
			col("comment_id"), col("moment_id"), col("user_id"), col("parent_id"),
			col("content"), col("created_at"), col("likes_count"), col("replies_count"),
		},
	}
	for _, moment := range moments {
		replyCounts := make(map[string]int)
		for _, comment := range moment.Comments {
			if comment.ParentID != "" {
				replyCounts[comment.ParentID]++
			}
		}
		for _, comment := range moment.Comments {
			var parent any
			if comment.ParentID != "" {
				parent = comment.ParentID
			}
			row := []any{
				comment.CommentID, moment.MomentID, comment.UserID, parent,
				comment.Content, baseTime.Add(-comment.CreatedAgo), len(comment.Likes), replyCounts[comment.CommentID],
			}
			if comment.ParentID == "" {
				topLevel.Rows = append(topLevel.Rows, row)
			} else {
				replies.Rows = append(replies.Rows, row)
			}
		}
	}
	return topLevel, replies
}

func buildCommentLikesTable(baseTime time.Time, moments []momentSpec) tableData {
	table := tableData{
		Table:   "comment_likes",
		Columns: []column{col("comment_id"), col("user_id"), col("liked_at")},
	}
	for _, moment := range moments {
		for _, comment := range moment.Comments {
			createdAt := baseTime.Add(-comment.CreatedAgo)
			for index, userID := range comment.Likes {
				table.addRow(comment.CommentID, userID, createdAt.Add(time.Duration(index+1)*time.Minute))
			}
		}
	}
	return table
}

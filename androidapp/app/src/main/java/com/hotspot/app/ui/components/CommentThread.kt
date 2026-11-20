package com.hotspot.app.ui.components

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.FavoriteBorder
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.hotspot.app.data.model.CommentItem
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.ui.theme.Tokens
import com.hotspot.app.util.CommentThread
import com.hotspot.app.util.TimeFormat

/**
 * 评论楼中楼线程（方案 B，docs/ui-components.md「评论楼中楼（方案 B）」/「详情页契约」）：
 * - 主评论头像 36 + 12 间距 = 内容列 x=48；昵称 12 / 正文 12 / 时间 11；
 * - 回复块无灰底/无描边/无内距，与主评论内容列同轴：回复头像 20 + 8 间距（回复头/正文/动作统一在头像右）；
 *   回复昵称 11 / 「回复」「@父昵称」10（@ Medium）/ 时间 10 / 正文 11；
 * - 作者微徽章 28×16 / r4 / 10sp（与昵称间距 6dp）；
 * - 动作行：心形 14 + 4dp + 计数 11 Medium，与「回复」11 Medium 间距 16，整行高 44 保证热区；
 * - 导引线：主评论头像底中心 → 首条回复头像左缘左侧 6dp，1.25dp / opacity 0.35 / textSecondary 圆头，
 *   随主评论实测行高自适应（onSizeChanged 只在行高变化时写状态，不做每帧测量）。
 */

// 评论楼中楼几何（docs/ui-components.md「详情页契约」/ Figma 748:48）：
// 主评论头像 36、回复头像 20——头像刻度 22/32/40/48 与 36/20 是近邻档，
// 按「缺档不新增近邻档位」纪律登记为组件布局几何（与鸿蒙 MomentDetailPage 同口径）。
private val MainAvatar = 36.dp
private val MainAvatarGap = Spacing.md // 12 → 主评论内容列 x=48
private val ReplyAvatar = 20.dp
private val ReplyAvatarGap = Spacing.sm // 8（回复头/正文/动作在头像右）
private val ReplyBlockGap = Spacing.md3 // 14（主评论行 → 首条回复，与 web mt-3.5 同口径）
private val ReplySpacing = Spacing.md2 // 10（回复之间）
private val GuideLineWidth = 1.25.dp // 设计稿登记细描边（圆角/描边刻度外的装饰几何）
private const val GuideLineOpacity = 0.35f // 设计稿 0.35 透明度（静态装饰线，不进主题刻度）
private val GuideLineCorner = Tokens.Radius.sm.dp // 12（与 web ThreadLine 圆弧同量级）
private val ActionGap = Spacing.lg // 16（点赞组 ↔ 回复）

// 字号档来自生成 token（12/11/10）；行高按 design-system §4.2 的 12→17 / 11→15 / 10→15 映射
private val MainNameStyle = TextStyle(
    fontSize = Tokens.FontSize.sm.sp,
    lineHeight = 17.sp,
    fontWeight = FontWeight.SemiBold,
)
private val MainBodyStyle = TextStyle(fontSize = Tokens.FontSize.sm.sp, lineHeight = 17.sp)
private val MainTimeStyle = TextStyle(fontSize = Tokens.FontSize.xs.sp, lineHeight = 15.sp)
private val ReplyNameStyle = TextStyle(
    fontSize = Tokens.FontSize.xs.sp,
    lineHeight = 15.sp,
    fontWeight = FontWeight.SemiBold,
)
private val ReplyBodyStyle = TextStyle(fontSize = Tokens.FontSize.xs.sp, lineHeight = 15.sp)
private val ReplyMetaStyle = TextStyle(fontSize = Tokens.FontSize.xxs.sp, lineHeight = 15.sp)
private val ActionTextStyle = TextStyle(
    fontSize = Tokens.FontSize.xs.sp,
    lineHeight = 15.sp,
    fontWeight = FontWeight.Medium,
)

/** 是否显示「作者」微徽章（评论作者 == 瞬间作者，用户 id 非空才判定）。 */
private fun isMomentAuthor(comment: CommentItem, momentAuthorId: String): Boolean =
    momentAuthorId.isNotBlank() && comment.user.userId == momentAuthorId

@Composable
fun CommentThreadItem(
    thread: CommentThread,
    momentAuthorId: String,
    actionsEnabled: Boolean,
    onToggleLike: (commentId: String, active: Boolean) -> Unit,
    onReply: (parentCommentId: String, nickname: String, userId: String) -> Unit,
    modifier: Modifier = Modifier,
) {
    var mainRowHeightPx by remember(thread.comment.commentId) { mutableIntStateOf(0) }
    Box(modifier.fillMaxWidth()) {
        Column(Modifier.fillMaxWidth()) {
            MainCommentRow(
                comment = thread.comment,
                isAuthor = isMomentAuthor(thread.comment, momentAuthorId),
                actionsEnabled = actionsEnabled,
                onToggleLike = onToggleLike,
                onReply = onReply,
                modifier = Modifier
                    .fillMaxWidth()
                    .onSizeChanged { size ->
                        // 只在行高真的变化时写状态；普通评论（无楼中楼）不测量，避免无谓重组
                        if (thread.replies.isNotEmpty() && size.height != mainRowHeightPx) {
                            mainRowHeightPx = size.height
                        }
                    },
            )
            if (thread.replies.isNotEmpty()) {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(start = MainAvatar + MainAvatarGap, top = ReplyBlockGap),
                    verticalArrangement = Arrangement.spacedBy(ReplySpacing),
                ) {
                    thread.replies.forEach { reply ->
                        ReplyRow(
                            reply = reply,
                            parentCommentId = thread.comment.commentId,
                            parentNickname = thread.comment.user.nickname,
                            momentAuthorId = momentAuthorId,
                            actionsEnabled = actionsEnabled,
                            onReply = onReply,
                        )
                    }
                }
            }
        }
        if (thread.replies.isNotEmpty() && mainRowHeightPx > 0) {
            ThreadGuideLine(
                mainRowHeightPx = mainRowHeightPx,
                color = HotSpotTheme.colors.textSecondary.copy(alpha = GuideLineOpacity),
                modifier = Modifier.matchParentSize(),
            )
        }
    }
}

/**
 * 楼中楼建筑导引线（纯静态装饰，无动效）：主评论头像底中心垂直下探，圆角右转，
 * 止于首条回复头像左缘左侧 6dp（终点垂直落在首条回复头像中心，随主评论行高自适应）。
 */
@Composable
private fun ThreadGuideLine(
    mainRowHeightPx: Int,
    color: Color,
    modifier: Modifier = Modifier,
) {
    Canvas(modifier) {
        val startX = (MainAvatar * 0.5f).toPx()
        val startY = MainAvatar.toPx()
        val corner = GuideLineCorner.toPx()
        // 内容列 48 - 6 = 42：导引线横段止于回复头像左缘左侧 6dp
        val endX = (MainAvatar + MainAvatarGap - Spacing.sm2).toPx()
        // 终点 y = 主评论实测高 + 回复块间距 + 首条回复头像半高
        val endY = mainRowHeightPx + (ReplyBlockGap + ReplyAvatar * 0.5f).toPx()
        if (endY - corner <= startY) return@Canvas
        val path = Path().apply {
            moveTo(startX, startY)
            lineTo(startX, endY - corner)
            quadraticTo(startX, endY, startX + corner, endY)
            lineTo(endX, endY)
        }
        drawPath(
            path = path,
            color = color,
            style = Stroke(width = GuideLineWidth.toPx(), cap = StrokeCap.Round, join = StrokeJoin.Round),
        )
    }
}

@Composable
private fun MainCommentRow(
    comment: CommentItem,
    isAuthor: Boolean,
    actionsEnabled: Boolean,
    onToggleLike: (commentId: String, active: Boolean) -> Unit,
    onReply: (parentCommentId: String, nickname: String, userId: String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    Row(modifier, verticalAlignment = Alignment.Top) {
        Avatar(
            name = comment.user.nickname.ifBlank { "匿名" },
            imageUrl = comment.user.avatarUrl,
            size = MainAvatar,
        )
        Spacer(Modifier.width(MainAvatarGap))
        Column(Modifier.weight(1f)) {
            CommentHeadRow(
                comment = comment,
                isReply = false,
                replyToNickname = "",
                isAuthor = isAuthor,
            )
            Spacer(Modifier.height(Spacing.sm2))
            if (comment.isDeleted) {
                DeletedPlaceholder(isReply = false)
            } else {
                Text(
                    text = comment.content,
                    style = MainBodyStyle,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.xs))
                CommentActions(
                    comment = comment,
                    showLike = true,
                    enabled = actionsEnabled,
                    onToggleLike = { onToggleLike(comment.commentId, !comment.liked) },
                    onReply = {
                        onReply(comment.commentId, comment.user.nickname, comment.user.userId)
                    },
                )
            }
        }
    }
}

@Composable
private fun ReplyRow(
    reply: CommentItem,
    parentCommentId: String,
    parentNickname: String,
    momentAuthorId: String,
    actionsEnabled: Boolean,
    onReply: (parentCommentId: String, nickname: String, userId: String) -> Unit,
) {
    val colors = HotSpotTheme.colors
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.Top) {
        Avatar(
            name = reply.user.nickname.ifBlank { "匿名" },
            imageUrl = reply.user.avatarUrl,
            size = ReplyAvatar,
        )
        Spacer(Modifier.width(ReplyAvatarGap))
        Column(Modifier.weight(1f)) {
            CommentHeadRow(
                comment = reply,
                isReply = true,
                replyToNickname = if (reply.isDeleted) "" else parentNickname,
                isAuthor = isMomentAuthor(reply, momentAuthorId),
            )
            Spacer(Modifier.height(Spacing.sm2))
            if (reply.isDeleted) {
                DeletedPlaceholder(isReply = true)
            } else {
                Text(
                    text = reply.content,
                    style = ReplyBodyStyle,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.xs))
                // 回复块的动作行只有「回复」（点赞回复首版不做；回复目标仍挂在所属顶层评论上）
                CommentActions(
                    comment = reply,
                    showLike = false,
                    enabled = actionsEnabled,
                    onToggleLike = {},
                    onReply = {
                        onReply(parentCommentId, reply.user.nickname, reply.user.userId)
                    },
                )
            }
        }
    }
}

/** 评论头行：昵称 + 作者微徽章 +（楼中楼）「回复 @父昵称」+ 时间。 */
@Composable
private fun CommentHeadRow(
    comment: CommentItem,
    isReply: Boolean,
    replyToNickname: String,
    isAuthor: Boolean,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Row(
            modifier = Modifier.weight(1f),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = comment.user.nickname.ifBlank { "匿名" },
                style = if (isReply) ReplyNameStyle else MainNameStyle,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f, fill = false),
            )
            if (isAuthor) {
                Spacer(Modifier.width(Spacing.sm2))
                CommentAuthorBadge()
            }
            if (isReply && replyToNickname.isNotBlank()) {
                Spacer(Modifier.width(Spacing.sm))
                Text(text = "回复", style = ReplyMetaStyle, color = colors.textSecondary, maxLines = 1)
                Spacer(Modifier.width(Spacing.xs))
                Text(
                    text = "@" + replyToNickname,
                    style = ReplyMetaStyle.copy(fontWeight = FontWeight.Medium),
                    color = colors.textSecondary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.widthIn(max = 120.dp),
                )
            }
        }
        Spacer(Modifier.width(Spacing.sm))
        Text(
            text = TimeFormat.relative(comment.createdAt),
            style = if (isReply) ReplyMetaStyle else MainTimeStyle,
            color = colors.textSecondary,
            maxLines = 1,
        )
    }
}

/** 软删除占位（spec/comments/delete 语义：content 清空，展示「该评论已删除」）。 */
@Composable
private fun DeletedPlaceholder(isReply: Boolean) {
    val colors = HotSpotTheme.colors
    Text(
        text = "该评论已删除",
        style = if (isReply) ReplyBodyStyle else MainBodyStyle,
        color = colors.textSecondary,
        fontStyle = FontStyle.Italic,
    )
}

/**
 * 评论动作行：点赞（心形 14 + 4dp + 计数 11 Medium）+「回复」11 Medium，组间距 16，
 * 整行垂直居中、高 44dp 保证触控热区。
 */
@Composable
private fun CommentActions(
    comment: CommentItem,
    showLike: Boolean,
    enabled: Boolean,
    onToggleLike: () -> Unit,
    onReply: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(Dimens.minTouchTarget),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (showLike) {
            val tint = if (comment.liked) colors.accent else colors.textSecondary
            Row(
                modifier = Modifier
                    .fillMaxHeight()
                    .widthIn(min = Dimens.minTouchTarget)
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                        enabled = enabled,
                        onClick = onToggleLike,
                    )
                    .hotSpotFocusRing(cornerRadius = Tokens.Radius.sm.dp)
                    .semantics {
                        role = Role.Button
                        contentDescription = if (comment.liked) "取消点赞评论" else "点赞评论"
                    },
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    imageVector = if (comment.liked) Icons.Filled.Favorite else Icons.Filled.FavoriteBorder,
                    contentDescription = null,
                    modifier = Modifier.size(Dimens.icon2xs),
                    tint = tint,
                )
                Spacer(Modifier.width(Spacing.xs))
                Text(text = comment.likeCount.toString(), style = ActionTextStyle, color = tint)
            }
            Spacer(Modifier.width(ActionGap))
        }
        Box(
            modifier = Modifier
                .fillMaxHeight()
                .widthIn(min = Dimens.minTouchTarget)
                .clickable(
                    interactionSource = remember { MutableInteractionSource() },
                    indication = null,
                    enabled = enabled,
                    onClick = onReply,
                )
                .hotSpotFocusRing(cornerRadius = Tokens.Radius.sm.dp)
                .semantics {
                    role = Role.Button
                    contentDescription = "回复评论"
                },
            contentAlignment = Alignment.CenterStart,
        ) {
            Text(text = "回复", style = ActionTextStyle, color = colors.textSecondary)
        }
    }
}

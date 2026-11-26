package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.MailOutline
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.hotspot.app.R
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MediaUrls
import com.hotspot.app.util.TimeFormat

/**
 * 整卡图片卡（design-system §17 Image-first overlay card）：
 * 图片铺满整卡 + 底部遮罩承载描述/作者/操作行 + 无图走「文字优先海报」。
 * feed 档 4:5，网格档 3:4（同一视图只用一档）。
 */
@Composable
fun MomentCard(
    moment: MomentItem,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    aspectRatio: Float = 4f / 5f,
    onAuthorClick: (() -> Unit)? = null,
    likedByMe: Boolean = false,
    onToggleLike: (() -> Unit)? = null,
) {
    val colors = HotSpotTheme.colors
    val imageUrl = MediaUrls.absolute(moment.images.firstOrNull()?.url)
    val hasImage = imageUrl != null
    Box(
        modifier = modifier
            .fillMaxWidth()
            .aspectRatio(aspectRatio)
            .clip(AppShapes.card)
            .background(if (hasImage) colors.subtle else colors.island)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                onClick = onClick,
            )
            .hotSpotFocusRing(cornerRadius = 20.dp),
    ) {
        if (hasImage) {
            AsyncImage(
                model = ImageRequest.Builder(LocalContext.current).data(imageUrl).crossfade(true).build(),
                contentDescription = moment.description.ifBlank { "瞬间图片" },
                modifier = Modifier.fillMaxSize(),
                contentScale = ContentScale.Crop,
            )
            // 底部遮罩：承载标题/作者/操作行（压图白字对比度硬约束）
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(
                        Brush.verticalGradient(
                            0f to Color.Transparent,
                            0.52f to Color.Transparent,
                            1f to colors.overlayStrong,
                        ),
                    ),
            )
            // 顶部作者行局部遮罩
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(72.dp)
                    .background(
                        Brush.verticalGradient(
                            colors = listOf(colors.overlayStrong, Color.Transparent),
                        ),
                    ),
            )
        } else {
            // 文字优先海报：保持卡位不变，读作文本卡而不是缺图骨架
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(Spacing.lg2),
                verticalArrangement = Arrangement.Center,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_quote),
                    contentDescription = null,
                    modifier = Modifier.size(Dimens.iconSm),
                    tint = colors.accent,
                )
                Text(
                    text = moment.description.ifBlank { "内容已删除" },
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Medium,
                    color = colors.textPrimary,
                    maxLines = 5,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }

        // 顶部作者行（压图：头像 + 白字昵称/时间）
        Row(
            modifier = Modifier
                .align(Alignment.TopStart)
                .fillMaxWidth()
                .padding(Spacing.md)
                .then(if (onAuthorClick != null) Modifier.clickable { onAuthorClick() } else Modifier),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Avatar(
                name = moment.user.nickname,
                imageUrl = moment.user.avatarUrl,
                size = 28.dp,
            )
            Spacer(Modifier.width(Spacing.sm))
            Column(Modifier.weight(1f)) {
                Text(
                    text = moment.user.nickname.ifBlank { "未知用户" },
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.contentInverse,
                    maxLines = 1,
                )
                Text(
                    text = TimeFormat.relative(moment.createdAt),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.contentInverseMuted,
                    maxLines = 1,
                )
            }
            val locationName = moment.location?.locationName
            if (!locationName.isNullOrBlank()) {
                MediaBadge(text = locationName)
            }
        }

        // 底部信息 + 操作行
        Column(
            modifier = Modifier
                .align(Alignment.BottomStart)
                .fillMaxWidth()
                .padding(Spacing.md),
        ) {
            if (hasImage) {
                Text(
                    text = moment.description.ifBlank { "内容已删除" },
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.contentInverse,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                Spacer(Modifier.height(Spacing.sm))
            }
            Row(verticalAlignment = Alignment.CenterVertically) {
                ActionCount(
                    icon = Icons.Filled.Favorite,
                    count = moment.likeCount,
                    tint = if (likedByMe) colors.accent else if (hasImage) colors.contentInverse else colors.textSecondary,
                    active = likedByMe,
                    onClick = onToggleLike,
                )
                Spacer(Modifier.width(Spacing.lg))
                ActionCount(
                    icon = Icons.Filled.MailOutline,
                    count = moment.commentsCount,
                    tint = if (hasImage) colors.contentInverse else colors.textSecondary,
                )
                Spacer(Modifier.width(Spacing.lg))
                ActionCount(
                    icon = Icons.Filled.Share,
                    count = moment.sharesCount,
                    tint = if (hasImage) colors.contentInverse else colors.textSecondary,
                )
            }
        }
    }
}

@Composable
private fun ActionCount(
    icon: ImageVector,
    count: Int,
    tint: Color,
    active: Boolean = false,
    onClick: (() -> Unit)? = null,
) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = if (onClick != null) {
            Modifier
                .heightIn(min = Dimens.minTouchTarget)
                .clip(AppShapes.pill)
                .clickable(onClick = onClick)
        } else {
            Modifier
        },
    ) {
        Icon(
            imageVector = icon,
            contentDescription = when {
                onClick == null -> null
                active -> "取消点赞"
                else -> "点赞瞬间"
            },
            modifier = Modifier.size(18.dp),
            tint = tint,
        )
        Spacer(Modifier.width(Spacing.xs))
        Text(
            text = count.toString(),
            style = MaterialTheme.typography.labelSmall,
            color = tint,
            maxLines = 1,
        )
    }
}

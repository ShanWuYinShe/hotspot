package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.MailOutline
import androidx.compose.material.icons.filled.Place
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.hotspot.app.data.model.GroupItem
import com.hotspot.app.data.model.MomentItem
import com.hotspot.app.data.model.UserItem
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MediaUrls
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

// 附近模块（设计稿 87:4409 枢纽 / 87:4192 人 / 87:3798 群组 / 87:3995 瞬间）共用卡片与原语。
// 距离由后端下发，客户端只格式化展示；关注/加入的交互态由页面持有，卡片只做展示 + 回调。

private val DateChipFormatter = DateTimeFormatter.ofPattern("M月d日").withZone(ZoneId.systemDefault())

/** 距离标签：<1000m 显示 "Nm"，否则 "N.Nkm"；无距离返回 null（不展示假数据）。 */
fun nearbyDistanceLabel(meters: Double?): String? {
    val value = meters ?: return null
    return if (value < 1000) value.toInt().toString() + "m" else String.format("%.1fkm", value / 1000)
}

/** 瞬间日期胶囊：ISO → "10月29日"；解析失败回落相对时间。 */
fun nearbyMomentDate(createdAt: String?): String {
    if (createdAt.isNullOrBlank()) return ""
    return try {
        DateChipFormatter.format(Instant.parse(createdAt))
    } catch (_: Exception) {
        try {
            DateChipFormatter.format(java.time.OffsetDateTime.parse(createdAt).toInstant())
        } catch (_: Exception) {
            ""
        }
    }
}

/** 定位胶囊（设计稿：标题右侧「济源市南天门 · 5km 内」白色胶囊）。 */
@Composable
fun NearbyLocationChip(text: String, modifier: Modifier = Modifier) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = modifier
            .clip(AppShapes.pill)
            .background(colors.island)
            .border(1.dp, colors.borderSubtle, AppShapes.pill)
            .padding(horizontal = Spacing.md2, vertical = Spacing.xs),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            imageVector = Icons.Filled.Place,
            contentDescription = null,
            modifier = Modifier.size(12.dp),
            tint = colors.textSecondary,
        )
        Spacer(Modifier.width(Spacing.xs))
        Text(
            text = text,
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

/** 实时附近统计卡（设计稿 87:4409：icon + 实时附近 + 实时雷达 + 位置/范围 + 人/瞬间/在线）。 */
@Composable
fun NearbyStatsCard(
    subtitle: String,
    peopleCount: String,
    momentsCount: String,
    onlineCount: String,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    SystemCard(modifier = modifier, contentPadding = PaddingValues(Spacing.lg)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                modifier = Modifier
                    .size(40.dp)
                    .clip(CircleShape)
                    .background(colors.accentSoftBg),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    imageVector = Icons.Filled.Place,
                    contentDescription = null,
                    modifier = Modifier.size(Dimens.iconSm),
                    tint = colors.accent,
                )
            }
            Spacer(Modifier.width(Spacing.md))
            Column(Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = "实时附近",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.SemiBold,
                        color = colors.textPrimary,
                    )
                    Spacer(Modifier.width(Spacing.sm))
                    Badge(text = "实时雷达", tone = BadgeTone.ONLINE, size = BadgeSize.SM)
                }
                Spacer(Modifier.height(2.dp))
                Text(
                    text = subtitle,
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
        Spacer(Modifier.height(Spacing.md))
        SoftDivider()
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceEvenly) {
            NearbyStatItem(value = peopleCount, label = "人", tint = colors.textPrimary, modifier = Modifier.weight(1f))
            NearbyStatItem(value = momentsCount, label = "瞬间", tint = colors.textPrimary, modifier = Modifier.weight(1f))
            NearbyStatItem(value = onlineCount, label = "在线", tint = colors.online, modifier = Modifier.weight(1f))
        }
    }
}

@Composable
private fun NearbyStatItem(value: String, label: String, tint: Color, modifier: Modifier = Modifier) {
    val colors = HotSpotTheme.colors
    Column(modifier = modifier, horizontalAlignment = Alignment.CenterHorizontally) {
        Text(
            text = value,
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.SemiBold,
            color = tint,
        )
        Spacer(Modifier.height(2.dp))
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall,
            color = colors.textSecondary,
        )
    }
}

/** 分区头（设计稿：标题 + 计数 + 右侧「查看全部 ›」）。 */
@Composable
fun NearbySectionHead(
    title: String,
    count: Int?,
    onSeeAll: (() -> Unit)? = null,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.SemiBold,
            color = colors.textPrimary,
        )
        if (count != null) {
            Spacer(Modifier.width(Spacing.xs))
            Text(
                text = count.toString(),
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
            )
        }
        Spacer(Modifier.weight(1f))
        if (onSeeAll != null) {
            Row(
                modifier = Modifier
                    .clip(AppShapes.compact)
                    .clickable(onClick = onSeeAll)
                    .padding(horizontal = Spacing.xs, vertical = Spacing.xs),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = "查看全部",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.accent,
                )
                Text(
                    text = " ›",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.accent,
                )
            }
        }
    }
}

/** 筛选胶囊行（设计稿：全部/摄影/户外/旅行/音乐/美食/阅读）。 */
@Composable
fun NearbyFilterChips(
    options: List<String>,
    selectedIndex: Int,
    onSelected: (Int) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    LazyRow(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(Spacing.sm),
        contentPadding = PaddingValues(horizontal = 2.dp),
    ) {
        itemsIndexed(options) { index, label ->
            val selected = index == selectedIndex
            Box(
                modifier = Modifier
                    .clip(AppShapes.pill)
                    .background(if (selected) colors.accent else colors.island)
                    .border(1.dp, if (selected) Color.Transparent else colors.borderSubtle, AppShapes.pill)
                    .clickable { onSelected(index) }
                    .padding(horizontal = Spacing.md, vertical = Spacing.xs)
                    .heightIn(min = 28.dp),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = label,
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
                    color = if (selected) colors.textOnAccent else colors.textSecondary,
                )
            }
        }
    }
}

/** 关注胶囊（设计稿：关注 = accent 软底胶囊；已关注 = subtle 底）。 */
@Composable
fun NearbyFollowButton(
    following: Boolean,
    busy: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = modifier
            .clip(AppShapes.pill)
            .background(if (following) colors.subtle else colors.accentSoftBg)
            .clickable(enabled = !busy, onClick = onClick)
            .padding(horizontal = Spacing.md, vertical = Spacing.xs)
            .heightIn(min = 28.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (!following) {
            Icon(
                imageVector = Icons.Filled.Add,
                contentDescription = null,
                modifier = Modifier.size(Dimens.iconXs),
                tint = colors.accent,
            )
            Spacer(Modifier.width(2.dp))
        }
        Text(
            text = if (following) "已关注" else "关注",
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.Medium,
            color = if (following) colors.textSecondary else colors.textAccentOnSoft,
        )
    }
}

/** 附近的人 · 网格卡（设计稿 87:4192：标签/距离 → 头像 → 昵称 → 副标题 → 关注）。 */
@Composable
fun NearbyUserCard(
    user: UserItem,
    following: Boolean,
    followBusy: Boolean,
    onToggleFollow: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    val label = user.tags.firstOrNull()?.let { "#$it" } ?: "#热点邻居"
    Box(
        modifier = modifier
            .shadow(1.dp, AppShapes.card, ambientColor = colors.shadowDark, spotColor = colors.shadowDark)
            .clip(AppShapes.card)
            .background(colors.island)
            .padding(Spacing.md),
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.fillMaxWidth()) {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Box(
                    modifier = Modifier
                        .clip(AppShapes.pill)
                        .background(colors.subtle)
                        .padding(horizontal = Spacing.sm2, vertical = 2.dp),
                ) {
                    Text(
                        text = label,
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                Spacer(Modifier.weight(1f))
                val distance = nearbyDistanceLabel(user.location?.distance)
                if (distance != null) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Box(Modifier.size(5.dp).clip(CircleShape).background(colors.accent))
                        Spacer(Modifier.width(Spacing.xs))
                        Text(
                            text = distance,
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.textSecondary,
                        )
                    }
                }
            }
            Spacer(Modifier.height(Spacing.md2))
            Avatar(
                name = user.user.nickname,
                imageUrl = user.user.avatarUrl,
                size = 56.dp,
            )
            Spacer(Modifier.height(Spacing.sm))
            Text(
                text = user.user.nickname.ifBlank { "匿名用户" },
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.SemiBold,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Spacer(Modifier.height(2.dp))
            Text(
                text = user.bio?.takeIf { it.isNotBlank() } ?: "热点邻居",
                style = MaterialTheme.typography.labelSmall,
                color = colors.textSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Spacer(Modifier.height(Spacing.md))
            NearbyFollowButton(following = following, busy = followBusy, onClick = onToggleFollow)
        }
    }
}

/** 附近的人 · 行（枢纽预览用：头像 + 昵称/距离 + 关注）。 */
@Composable
fun NearbyUserRow(
    user: UserItem,
    following: Boolean,
    followBusy: Boolean,
    onToggleFollow: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(vertical = Spacing.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Avatar(
            name = user.user.nickname,
            imageUrl = user.user.avatarUrl,
            size = Dimens.avatarMd,
        )
        Spacer(Modifier.width(Spacing.md))
        Column(Modifier.weight(1f)) {
            Text(
                text = user.user.nickname.ifBlank { "匿名用户" },
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium,
                color = colors.textPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            val distance = nearbyDistanceLabel(user.location?.distance)
            if (distance != null) {
                Text(
                    text = distance,
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.textSecondary,
                )
            }
        }
        NearbyFollowButton(following = following, busy = followBusy, onClick = onToggleFollow)
    }
}

/** 附近群组 · 网格卡（设计稿 87:3798：封面/首字 + 分类胶囊 + 群名 + 成员/距离 + 加入）。 */
@Composable
fun NearbyGroupCard(
    group: GroupItem,
    joined: Boolean,
    pending: Boolean,
    busy: Boolean,
    onClick: () -> Unit,
    onJoin: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    val avatarUrl = MediaUrls.absolute(group.group.avatarUrl)
    Box(
        modifier = modifier
            .shadow(1.dp, AppShapes.card, ambientColor = colors.shadowDark, spotColor = colors.shadowDark)
            .clip(AppShapes.card)
            .background(colors.island)
            .clickable(onClick = onClick),
    ) {
        Column {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(96.dp)
                    .background(colors.subtle),
                contentAlignment = Alignment.Center,
            ) {
                if (avatarUrl != null) {
                    AsyncImage(
                        model = ImageRequest.Builder(LocalContext.current).data(avatarUrl).crossfade(true).build(),
                        contentDescription = group.group.name,
                        modifier = Modifier.fillMaxSize(),
                        contentScale = ContentScale.Crop,
                    )
                } else {
                    Text(
                        text = group.group.name.trim().take(1).ifEmpty { "群" },
                        style = MaterialTheme.typography.headlineMedium,
                        fontWeight = FontWeight.SemiBold,
                        color = colors.accent,
                    )
                }
                // 分类胶囊（深色，对齐设计稿左上的「摄影/户外/其他」徽章）
                Box(
                    modifier = Modifier
                        .align(Alignment.TopStart)
                        .padding(Spacing.sm)
                        .clip(AppShapes.pill)
                        .background(colors.textPrimary)
                        .padding(horizontal = Spacing.sm2, vertical = 2.dp),
                ) {
                    Text(
                        text = group.tags.firstOrNull() ?: "其他",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.island,
                    )
                }
                // 加入状态点（设计稿右上圆点：已加入 = online 实心）
                Box(
                    modifier = Modifier
                        .align(Alignment.TopEnd)
                        .padding(Spacing.sm)
                        .size(10.dp)
                        .clip(CircleShape)
                        .background(if (joined) colors.online else colors.textPrimary.copy(alpha = 0.35f)),
                )
            }
            Column(Modifier.padding(Spacing.md)) {
                Text(
                    text = group.group.name.ifBlank { "未命名群组" },
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Spacer(Modifier.height(Spacing.xs))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = group.group.membersCount.toString() + " 位成员",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                    )
                    Spacer(Modifier.weight(1f))
                    val distance = nearbyDistanceLabel(group.location?.distance)
                    if (distance != null) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Icon(
                                imageVector = Icons.Filled.Place,
                                contentDescription = null,
                                modifier = Modifier.size(12.dp),
                                tint = colors.textSecondary,
                            )
                            Spacer(Modifier.width(2.dp))
                            Text(
                                text = distance,
                                style = MaterialTheme.typography.labelSmall,
                                color = colors.textSecondary,
                            )
                        }
                    }
                }
                Spacer(Modifier.height(Spacing.sm))
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(AppShapes.pill)
                        .background(
                            when {
                                joined -> colors.subtle
                                else -> colors.accentSoftBg
                            },
                        )
                        .clickable(enabled = !joined && !pending && !busy, onClick = onJoin)
                        .padding(vertical = Spacing.xs),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = when {
                            joined -> "已加入"
                            pending -> "已申请"
                            else -> "加入"
                        },
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.Medium,
                        color = when {
                            joined -> colors.textSecondary
                            else -> colors.textAccentOnSoft
                        },
                    )
                }
            }
        }
    }
}

/** 附近瞬间 · 网格卡（设计稿 87:3995：日期胶囊 + 图/文 + 作者 + 互动计数）。 */
@Composable
fun NearbyMomentCard(
    moment: MomentItem,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    val imageUrl = MediaUrls.absolute(moment.images.firstOrNull()?.url)
    Box(
        modifier = modifier
            .shadow(1.dp, AppShapes.card, ambientColor = colors.shadowDark, spotColor = colors.shadowDark)
            .clip(AppShapes.card)
            .background(colors.island)
            .clickable(onClick = onClick),
    ) {
        Column {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .aspectRatio(4f / 5f)
                    .background(if (imageUrl != null) colors.subtle else colors.island),
            ) {
                if (imageUrl != null) {
                    AsyncImage(
                        model = ImageRequest.Builder(LocalContext.current).data(imageUrl).crossfade(true).build(),
                        contentDescription = moment.description.ifBlank { "瞬间图片" },
                        modifier = Modifier.fillMaxSize(),
                        contentScale = ContentScale.Crop,
                    )
                } else {
                    Column(
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(Spacing.md),
                    ) {
                        Text(
                            text = "❝❝",
                            style = MaterialTheme.typography.titleMedium,
                            color = colors.accent,
                        )
                        Spacer(Modifier.height(Spacing.sm2))
                        Text(
                            text = moment.description.ifBlank { "（无描述）" },
                            style = MaterialTheme.typography.bodyMedium,
                            color = colors.textPrimary,
                            maxLines = 6,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
                val date = nearbyMomentDate(moment.createdAt)
                if (date.isNotBlank()) {
                    Box(
                        modifier = Modifier
                            .align(Alignment.TopEnd)
                            .padding(Spacing.sm)
                            .clip(AppShapes.pill)
                            .background(colors.textPrimary.copy(alpha = 0.82f))
                            .padding(horizontal = Spacing.sm2, vertical = 2.dp),
                    ) {
                        Text(
                            text = date,
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.island,
                        )
                    }
                }
            }
            Column(Modifier.padding(Spacing.md)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Avatar(
                        name = moment.user.nickname,
                        imageUrl = moment.user.avatarUrl,
                        size = 20.dp,
                    )
                    Spacer(Modifier.width(Spacing.xs))
                    Text(
                        text = moment.user.nickname.ifBlank { "匿名用户" },
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textPrimary,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f),
                    )
                }
                Spacer(Modifier.height(Spacing.sm2))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(
                        imageVector = Icons.Filled.Favorite,
                        contentDescription = null,
                        modifier = Modifier.size(14.dp),
                        tint = colors.textSecondary,
                    )
                    Spacer(Modifier.width(2.dp))
                    Text(
                        text = moment.likeCount.toString(),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                    )
                    Spacer(Modifier.width(Spacing.md))
                    Icon(
                        imageVector = Icons.Filled.MailOutline,
                        contentDescription = null,
                        modifier = Modifier.size(14.dp),
                        tint = colors.textSecondary,
                    )
                    Spacer(Modifier.width(2.dp))
                    Text(
                        text = moment.commentsCount.toString(),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.textSecondary,
                    )
                }
            }
        }
    }
}

package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Email
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Place
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.hotspot.app.navigation.Routes
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing

private data class BottomNavItem(
    val route: String,
    val label: String,
    val icon: ImageVector,
    val primary: Boolean = false,
)

/** 悬浮岛内边距（对齐 Web MobilePortrait：px-3.5 = 14 / p-1.5 = 6 / 圆角 28 / 玻璃 + 发丝边 + 投影）。 */
private val DockSideInset = 14.dp
private val DockIslandPadding = 6.dp
private val DockBottomGap = 12.dp
private val DockIndicatorWidth = 14.dp
private val DockIndicatorHeight = 3.dp
private val DockFabSize = 48.dp

/**
 * 底部 5 tab：首页 / 附近 / 发布(+) / 消息 / 我的。
 * 形态对齐设计稿「底部导航」522:2 与 Web MobilePortrait 悬浮岛：
 * 页面底部左右 14 留白 + 底部 max(12, 系统手势条) 留白，岛体 28 圆角 / 玻璃底 / 发丝边 / 抬升投影；
 * 中心「发布」为 48 圆钮（无文字，对齐 Web 中心 FAB）；其余 tab 为 24 图标 + 11fp 标签 + 选中指示条。
 */
@Composable
fun BottomNav(
    currentRoute: String?,
    onNavigate: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    val items = listOf(
        BottomNavItem(Routes.HOME, "首页", Icons.Filled.Home),
        BottomNavItem(Routes.NEARBY, "附近", Icons.Filled.Place),
        BottomNavItem(Routes.PUBLISH, "发布", Icons.Filled.Add, primary = true),
        BottomNavItem(Routes.MESSAGES, "消息", Icons.Filled.Email),
        BottomNavItem(Routes.ME, "我的", Icons.Filled.Person),
    )
    // 底部安全区由系统提供：取系统导航栏 inset，至少保留 12dp 呼吸（对齐 Web pb-[max(12px,env(...))]）
    val navBarBottom = WindowInsets.navigationBars.asPaddingValues().calculateBottomPadding()
    val bottomGap = if (navBarBottom > DockBottomGap) navBarBottom else DockBottomGap
    Box(
        modifier = modifier
            .fillMaxWidth()
            .padding(
                start = DockSideInset,
                end = DockSideInset,
                bottom = bottomGap,
            ),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .shadow(
                    elevation = 12.dp,
                    shape = AppShapes.sheet,
                    ambientColor = colors.shadowDark,
                    spotColor = colors.shadowDark,
                )
                .clip(AppShapes.sheet)
                .background(colors.island)
                .border(1.dp, colors.border, AppShapes.sheet)
                .heightIn(min = Dimens.bottomNavHeight)
                .padding(horizontal = DockIslandPadding, vertical = DockIslandPadding),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceEvenly,
        ) {
            items.forEach { item ->
                BottomNavItemView(
                    item = item,
                    selected = Routes.isTabActive(item.route, currentRoute),
                    onClick = { onNavigate(item.route) },
                    modifier = Modifier.weight(1f),
                )
            }
        }
    }
}

@Composable
private fun BottomNavItemView(
    item: BottomNavItem,
    selected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = HotSpotTheme.colors
    val tint = if (selected) colors.accent else colors.navContent
    Column(
        modifier = modifier
            .heightIn(min = Dimens.minTouchTarget)
            .clip(RoundedCornerShape(12.dp))
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                onClick = onClick,
            )
            .padding(vertical = Spacing.xs),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        if (item.primary) {
            // 中心发布：48 圆钮（对齐设计稿 522:2 / Web size-12 FAB），无文字
            Box(
                modifier = Modifier
                    .size(DockFabSize)
                    .shadow(
                        elevation = 8.dp,
                        shape = CircleShape,
                        ambientColor = colors.accent,
                        spotColor = colors.accent,
                    )
                    .clip(CircleShape)
                    .background(colors.accent),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    imageVector = item.icon,
                    contentDescription = "发布",
                    modifier = Modifier.size(Dimens.iconSm),
                    tint = colors.textOnAccent,
                )
            }
        } else {
            Icon(
                imageVector = item.icon,
                contentDescription = null,
                modifier = Modifier
                    .size(Dimens.iconMd)
                    .scale(if (selected) 1.1f else 1f),
                tint = tint,
            )
            Spacer(Modifier.height(2.dp))
            Text(
                text = item.label,
                style = MaterialTheme.typography.labelSmall,
                fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
                color = tint,
                maxLines = 1,
            )
            // 选中指示条在「图标 + 文字」下方（设计稿 底部导航 522:2：label 下方 14x3 短横条），
            // 形状线索不依赖颜色（design-system §18）；未选中保留同尺寸占位避免文字跳动。
            Spacer(Modifier.height(Spacing.half))
            Box(
                modifier = Modifier
                    .width(DockIndicatorWidth)
                    .height(DockIndicatorHeight)
                    .background(if (selected) colors.accent else Color.Transparent, AppShapes.pill),
            )
        }
    }
}

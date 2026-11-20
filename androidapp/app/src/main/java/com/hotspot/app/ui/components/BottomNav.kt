package com.hotspot.app.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
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

/** 底部 5 tab：首页 / 附近 / 发布(+) / 消息 / 我的（34 图标 + 11fp 标签 + 选中指示条）。 */
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
    Row(
        modifier = modifier
            .fillMaxWidth()
            .background(colors.surface)
            .navigationBarsPadding()
            .heightIn(min = Dimens.bottomNavHeight)
            .padding(horizontal = Spacing.sm, vertical = Spacing.xs),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceEvenly,
    ) {
        items.forEach { item ->
            BottomNavItemView(
                item = item,
                selected = currentRoute == item.route,
                onClick = { onNavigate(item.route) },
                modifier = Modifier.weight(1f),
            )
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
            Box(
                modifier = Modifier
                    .size(Dimens.avatarXs)
                    .clip(CircleShape)
                    .background(colors.accent),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    imageVector = item.icon,
                    contentDescription = item.label,
                    modifier = Modifier.size(Dimens.iconSm),
                    tint = colors.textOnAccent,
                )
            }
        } else {
            Icon(
                imageVector = item.icon,
                contentDescription = null,
                modifier = Modifier.size(Dimens.iconMd),
                tint = tint,
            )
        }
        Spacer(Modifier.height(2.dp))
        Text(
            text = item.label,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
            color = tint,
            maxLines = 1,
        )
        // 选中指示条在「图标 + 文字」下方（设计稿 底部导航 522:2：首页 label 下方短横条），
        // 形状线索不依赖颜色（design-system §18）；未选中保留同尺寸占位避免文字跳动。
        Spacer(Modifier.height(Spacing.half))
        Box(
            modifier = Modifier
                .width(Spacing.lg2)
                .height(3.dp)
                .background(if (selected) colors.accent else Color.Transparent, AppShapes.pill),
        )
    }
}

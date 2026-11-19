package com.hotspot.app.ui.theme

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Shapes
import androidx.compose.ui.unit.dp

/**
 * 圆角档只允许 design-system §6 登记的 4 档 + full + sheet：
 * 12 / 16 / 20 / 24 / 9999(胶囊) / 28(sheet)。近邻档（8/18/22）不存在，禁止引入。
 */
private val r = Tokens.Radius

val HotSpotShapes = Shapes(
    extraSmall = RoundedCornerShape(r.sm.dp),
    small = RoundedCornerShape(r.md.dp),
    medium = RoundedCornerShape(r.lg.dp),
    large = RoundedCornerShape(r.xl.dp),
    extraLarge = RoundedCornerShape(r.sheet.dp),
)

/** 语义形状：与前端 SHAPE 常量一一对应（CARD 20 / HERO 24 / COMPACT 16 / INPUT 16 / BUTTON FULL …）。 */
object AppShapes {
    val compact = RoundedCornerShape(r.md.dp) // 16
    val card = RoundedCornerShape(r.lg.dp) // 20
    val hero = RoundedCornerShape(r.xl.dp) // 24
    val sheet = RoundedCornerShape(r.sheet.dp) // 28
    val pill = RoundedCornerShape(r.full.dp)
    val chip = RoundedCornerShape(r.full.dp)
    val iconTile = RoundedCornerShape(r.sm.dp) // 12
    val badge = RoundedCornerShape(r.full.dp)
}

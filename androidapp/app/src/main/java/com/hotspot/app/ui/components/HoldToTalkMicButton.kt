package com.hotspot.app.ui.components

import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import com.hotspot.app.R
import com.hotspot.app.ui.theme.Dimens

/**
 * 按住说话麦克风：按下开始录音，按住上滑超过阈值切换为取消态，松手时回调是否取消。
 * 录音中的行会保持组合（不换成录音栏），否则 pointerInput 会被取消导致收不到松手事件。
 */
@Composable
fun HoldToTalkMicButton(
    enabled: Boolean,
    tint: Color,
    onPressStart: () -> Unit,
    onSlideCancelChange: (Boolean) -> Unit,
    onRelease: (cancelled: Boolean) -> Unit,
    modifier: Modifier = Modifier,
) {
    val cancelThreshold = with(LocalDensity.current) { 72.dp.toPx() }
    Box(
        modifier = modifier
            .clip(CircleShape)
            .pointerInput(enabled) {
                if (!enabled) return@pointerInput
                awaitEachGesture {
                    val down = awaitFirstDown(requireUnconsumed = false)
                    var cancelled = false
                    onPressStart()
                    while (true) {
                        val event = awaitPointerEvent()
                        val change = event.changes.firstOrNull { it.id == down.id } ?: break
                        if (!change.pressed) {
                            change.consume()
                            break
                        }
                        val nowCancelled = change.position.y - down.position.y < -cancelThreshold
                        if (nowCancelled != cancelled) {
                            cancelled = nowCancelled
                            onSlideCancelChange(nowCancelled)
                        }
                        change.consume()
                    }
                    onSlideCancelChange(false)
                    onRelease(cancelled)
                }
            },
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_mic),
            contentDescription = "按住说话",
            modifier = Modifier.size(Dimens.iconMd),
            tint = tint,
        )
    }
}

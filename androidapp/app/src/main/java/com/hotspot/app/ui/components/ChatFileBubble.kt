package com.hotspot.app.ui.components

import android.content.Intent
import android.net.Uri
import android.widget.Toast
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.hotspot.app.R
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import com.hotspot.app.util.MediaUrls

/** 聊天文件气泡：文件图标 + 文件名 + 大小；点击用系统 ACTION_VIEW 打开签名 URL。 */
@Composable
fun ChatFileBubble(
    fileName: String?,
    fileSize: Long?,
    fileUrl: String?,
    isMine: Boolean,
) {
    val colors = HotSpotTheme.colors
    val context = LocalContext.current
    val absoluteUrl = MediaUrls.absolute(fileUrl)
    val tint = if (isMine) colors.textOnAccent else colors.textPrimary
    Row(
        modifier = Modifier
            .widthIn(max = 240.dp)
            .clickable(enabled = !absoluteUrl.isNullOrBlank()) {
                runCatching {
                    context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(absoluteUrl)))
                }.onFailure {
                    Toast.makeText(context, "没有可打开该文件的应用", Toast.LENGTH_SHORT).show()
                }
            },
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_action_file),
            contentDescription = "文件",
            modifier = Modifier.size(28.dp),
            tint = tint,
        )
        Spacer(Modifier.width(Spacing.sm))
        Column {
            Text(
                text = fileName?.takeIf { it.isNotBlank() } ?: "聊天文件",
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium,
                color = tint,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = formatFileSize(fileSize),
                style = MaterialTheme.typography.bodySmall,
                color = if (isMine) colors.textOnAccent.copy(alpha = 0.8f) else colors.textSecondary,
            )
        }
    }
}

private fun formatFileSize(bytes: Long?): String {
    if (bytes == null || bytes <= 0) return "未知大小"
    if (bytes < 1024) return "$bytes B"
    val kb = bytes / 1024.0
    if (kb < 1024) return String.format(java.util.Locale.US, "%.1f KB", kb)
    val mb = kb / 1024.0
    return String.format(java.util.Locale.US, "%.1f MB", mb)
}

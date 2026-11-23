package com.hotspot.app.util

/**
 * 聊天文件校验（spec/api_spec_media.md → chat_file 安全约束）：
 * 100MB 上限 + 扩展名/MIME 双黑名单（可执行、脚本/标记、服务端脚本）。
 * 服务端仍会做权威校验；客户端提前拦截给明确提示，避免大文件白传。
 */
object ChatFileRules {
    const val MAX_FILE_BYTES = 100L * 1024 * 1024

    private val BLOCKED_EXTENSIONS = setOf(
        "exe", "bat", "cmd", "sh", "app", "msi", "com", "scr",
        "html", "htm", "svg", "js", "xhtml",
        "php", "jsp", "asp", "aspx",
    )

    private val BLOCKED_MIME_PREFIXES = listOf(
        "application/x-msdownload",
        "application/x-msdos-program",
        "application/x-sh",
        "application/x-bat",
        "application/x-msi",
        "application/javascript",
        "text/javascript",
        "text/html",
        "application/xhtml+xml",
        "image/svg+xml",
    )

    /** 返回 null 表示可上传；否则为给用户的中文原因。 */
    fun blockedReason(fileName: String, fileSize: Long, mimeType: String?): String? {
        if (fileSize <= 0) return "文件为空，无法发送"
        if (fileSize > MAX_FILE_BYTES) return "文件不能超过 100MB"
        val extension = fileName.substringAfterLast('.', "").lowercase()
        if (extension in BLOCKED_EXTENSIONS) return "该文件类型不允许发送"
        val mime = mimeType?.substringBefore(';')?.trim()?.lowercase().orEmpty()
        if (mime.isNotEmpty() && BLOCKED_MIME_PREFIXES.any { mime == it }) {
            return "该文件类型不允许发送"
        }
        return null
    }
}

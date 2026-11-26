package com.hotspot.app.util

import android.content.Context

/** 聊天输入草稿：按会话 key 持久化，返回会话时恢复未发送文本。 */
object ChatDrafts {
    private const val PREFS = "chat_drafts"

    fun load(context: Context, key: String): String =
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getString(key, "").orEmpty()

    fun save(context: Context, key: String, text: String) {
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit().putString(key, text).apply()
    }

    fun clear(context: Context, key: String) {
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit().remove(key).apply()
    }
}

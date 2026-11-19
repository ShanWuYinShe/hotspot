package com.hotspot.app.data.auth

import android.content.Context
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import java.util.UUID

/** 登录会话（App 端凭证）：与 spec/api_spec_public.md AuthTokenPairApp 对应。 */
data class AuthSession(
    val userId: String,
    val accessToken: String,
    val refreshToken: String,
)

/**
 * App 端 Token 持久化（SharedPreferences）。
 *
 * TODO(安全加固，登记待办)：本机依赖清单未包含 androidx.security:security-crypto，
 *  当前用普通 SharedPreferences 持久化 access/refresh token。上线前必须迁移到
 *  EncryptedSharedPreferences + Android Keystore（或 DataStore + 系统密钥库），
 *  并把本文件替换为加密实现；接口（session/save/clear/deviceId）保持不变。
 */
class TokenStore(context: Context) {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    private val _session = MutableStateFlow(loadSession())
    val session: StateFlow<AuthSession?> = _session.asStateFlow()

    val accessToken: String? get() = _session.value?.accessToken

    fun save(userId: String, accessToken: String, refreshToken: String) {
        prefs.edit()
            .putString(KEY_USER_ID, userId)
            .putString(KEY_ACCESS_TOKEN, accessToken)
            .putString(KEY_REFRESH_TOKEN, refreshToken)
            .apply()
        _session.value = AuthSession(userId, accessToken, refreshToken)
    }

    fun clear() {
        prefs.edit()
            .remove(KEY_USER_ID)
            .remove(KEY_ACCESS_TOKEN)
            .remove(KEY_REFRESH_TOKEN)
            .apply()
        _session.value = null
    }

    /** 设备标识（UUID v4，长期稳定）：App 端 signin/register 必填非 null。 */
    fun deviceId(): String {
        val existing = prefs.getString(KEY_DEVICE_ID, null)
        if (existing != null && existing.isNotBlank()) return existing
        val created = UUID.randomUUID().toString()
        prefs.edit().putString(KEY_DEVICE_ID, created).apply()
        return created
    }

    private fun loadSession(): AuthSession? {
        val userId = prefs.getString(KEY_USER_ID, null)
        val accessToken = prefs.getString(KEY_ACCESS_TOKEN, null)
        val refreshToken = prefs.getString(KEY_REFRESH_TOKEN, null)
        if (userId.isNullOrBlank() || accessToken.isNullOrBlank() || refreshToken.isNullOrBlank()) return null
        return AuthSession(userId, accessToken, refreshToken)
    }

    private companion object {
        const val PREFS_NAME = "hotspot_auth"
        const val KEY_USER_ID = "user_id"
        const val KEY_ACCESS_TOKEN = "access_token"
        const val KEY_REFRESH_TOKEN = "refresh_token"
        const val KEY_DEVICE_ID = "device_id"
    }
}

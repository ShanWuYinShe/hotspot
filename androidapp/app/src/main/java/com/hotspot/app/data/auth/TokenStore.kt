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
 * App 端 Token 持久化：access/refresh token 使用 Android Keystore AES/GCM 加密后写入 SharedPreferences。
 *
 * - 首次读取旧版本明文 token 时自动加密迁移，并删除明文键；
 * - Keystore 解密失败（密钥丢失/跨设备恢复备份）按未登录处理，不抛异常导致启动崩溃；
 * - device_id 非敏感、保持明文；user_id 仅用于本地会话展示，保持明文。
 */
class TokenStore(context: Context) {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
    private val cipher = TokenCipher()

    private val _session = MutableStateFlow(loadSession())
    val session: StateFlow<AuthSession?> = _session.asStateFlow()

    val accessToken: String? get() = _session.value?.accessToken

    fun save(userId: String, accessToken: String, refreshToken: String) {
        storeEncrypted(userId, accessToken, refreshToken)
        _session.value = AuthSession(userId, accessToken, refreshToken)
    }

    fun clear() {
        prefs.edit()
            .remove(KEY_USER_ID)
            .remove(KEY_ACCESS_TOKEN_ENC)
            .remove(KEY_REFRESH_TOKEN_ENC)
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
        val encryptedAccess = prefs.getString(KEY_ACCESS_TOKEN_ENC, null)
        val encryptedRefresh = prefs.getString(KEY_REFRESH_TOKEN_ENC, null)
        if (!userId.isNullOrBlank() && encryptedAccess != null && encryptedRefresh != null) {
            val accessToken = cipher.decrypt(encryptedAccess)
            val refreshToken = cipher.decrypt(encryptedRefresh)
            if (!accessToken.isNullOrBlank() && !refreshToken.isNullOrBlank()) {
                return AuthSession(userId, accessToken, refreshToken)
            }
            // 密钥失效：清掉不可解密的密文，按未登录处理。
            prefs.edit()
                .remove(KEY_USER_ID)
                .remove(KEY_ACCESS_TOKEN_ENC)
                .remove(KEY_REFRESH_TOKEN_ENC)
                .apply()
            return null
        }

        // 旧版本迁移：明文 token → Keystore 加密后删除明文键。
        val legacyAccess = prefs.getString(KEY_ACCESS_TOKEN, null)
        val legacyRefresh = prefs.getString(KEY_REFRESH_TOKEN, null)
        if (userId.isNullOrBlank() || legacyAccess.isNullOrBlank() || legacyRefresh.isNullOrBlank()) {
            return null
        }
        return runCatching {
            storeEncrypted(userId, legacyAccess, legacyRefresh)
            AuthSession(userId, legacyAccess, legacyRefresh)
        }.getOrNull()
    }

    private fun storeEncrypted(userId: String, accessToken: String, refreshToken: String) {
        prefs.edit()
            .putString(KEY_USER_ID, userId)
            .putString(KEY_ACCESS_TOKEN_ENC, cipher.encrypt(accessToken))
            .putString(KEY_REFRESH_TOKEN_ENC, cipher.encrypt(refreshToken))
            .remove(KEY_ACCESS_TOKEN)
            .remove(KEY_REFRESH_TOKEN)
            .apply()
    }

    private companion object {
        const val PREFS_NAME = "hotspot_auth"
        const val KEY_USER_ID = "user_id"
        const val KEY_ACCESS_TOKEN = "access_token"
        const val KEY_REFRESH_TOKEN = "refresh_token"
        const val KEY_ACCESS_TOKEN_ENC = "access_token_enc"
        const val KEY_REFRESH_TOKEN_ENC = "refresh_token_enc"
        const val KEY_DEVICE_ID = "device_id"
    }
}

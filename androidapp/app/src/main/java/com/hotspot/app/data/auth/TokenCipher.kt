package com.hotspot.app.data.auth

import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import java.nio.ByteBuffer
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey

/**
 * Token 加密（Android Keystore AES/GCM/NoPadding）。
 *
 * 密钥不出 Keystore（不可导出）；每条密文格式为 "v1:" + base64(4 字节 IV 长度 + IV + 密文)。
 * 解密失败（密钥被清除、备份恢复到新设备等）返回 null，调用方按未登录处理。
 */
class TokenCipher {
    private val keyStore: KeyStore = KeyStore.getInstance(ANDROID_KEYSTORE).apply { load(null) }

    fun encrypt(plaintext: String): String {
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.ENCRYPT_MODE, secretKey())
        val iv = cipher.iv
        val encrypted = cipher.doFinal(plaintext.toByteArray(Charsets.UTF_8))
        val combined = ByteBuffer.allocate(Int.SIZE_BYTES + iv.size + encrypted.size)
            .putInt(iv.size)
            .put(iv)
            .put(encrypted)
            .array()
        return VERSION_PREFIX + Base64.encodeToString(combined, Base64.NO_WRAP)
    }

    fun decrypt(value: String): String? = runCatching {
        require(value.startsWith(VERSION_PREFIX)) { "未知密文版本" }
        val combined = Base64.decode(value.removePrefix(VERSION_PREFIX), Base64.NO_WRAP)
        val buffer = ByteBuffer.wrap(combined)
        val ivSize = buffer.int
        require(ivSize in 12..16 && buffer.remaining() > ivSize) { "密文格式错误" }
        val iv = ByteArray(ivSize).also(buffer::get)
        val encrypted = ByteArray(buffer.remaining()).also(buffer::get)
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.DECRYPT_MODE, secretKey(), javax.crypto.spec.GCMParameterSpec(128, iv))
        String(cipher.doFinal(encrypted), Charsets.UTF_8)
    }.getOrNull()

    private fun secretKey(): SecretKey {
        (keyStore.getKey(KEY_ALIAS, null) as? SecretKey)?.let { return it }
        synchronized(this) {
            (keyStore.getKey(KEY_ALIAS, null) as? SecretKey)?.let { return it }
            val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, ANDROID_KEYSTORE)
            generator.init(
                KeyGenParameterSpec.Builder(KEY_ALIAS, KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT)
                    .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                    .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                    .setKeySize(256)
                    .setUserAuthenticationRequired(false)
                    .build(),
            )
            return generator.generateKey()
        }
    }

    private companion object {
        const val ANDROID_KEYSTORE = "AndroidKeyStore"
        const val KEY_ALIAS = "hotspot_token_aes_key_v1"
        const val TRANSFORMATION = "AES/GCM/NoPadding"
        const val VERSION_PREFIX = "v1:"
    }
}

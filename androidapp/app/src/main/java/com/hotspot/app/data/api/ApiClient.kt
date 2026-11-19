package com.hotspot.app.data.api

import com.hotspot.app.BuildConfig
import com.hotspot.app.data.auth.TokenStore
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.KSerializer
import kotlinx.serialization.SerializationException
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.IOException
import java.net.SocketTimeoutException
import java.time.Duration
import java.util.UUID

/**
 * 统一响应信封（spec/api_spec_base.md「统一响应格式」）：code=0 成功；!=0 失败且 data 可能
 * 携带结构化错误（如 10200 的 forbidden_reason）。data 先按 JsonElement 接收，成功后再按端点类型解码。
 */
@Serializable
private data class ApiEnvelope(
    val code: Int = 0,
    val message: String? = null,
    val data: JsonElement? = null,
)

/** 统一错误模型：网络层 / HTTP 层 / 业务码层，UI 只消费 userMessage 与业务分支。 */
sealed class ApiError {
    abstract val userMessage: String

    data class Network(override val userMessage: String) : ApiError()

    data class Http(val status: Int, override val userMessage: String) : ApiError()

    data class Business(
        val code: Int,
        override val userMessage: String,
        val forbiddenReason: String? = null,
    ) : ApiError()
}

sealed interface ApiResult<out T> {
    data class Ok<T>(val data: T) : ApiResult<T>
    data class Err(val error: ApiError) : ApiResult<Nothing>
}

fun <T> ApiResult<T>.dataOrNull(): T? = (this as? ApiResult.Ok)?.data

fun <T> ApiResult<T>.errorOrNull(): ApiError? = (this as? ApiResult.Err)?.error

/**
 * HTTP 客户端：OkHttp + kotlinx-serialization；所有请求 POST + JSON envelope。
 * - 自动附带 Authorization: Bearer <access_token>（/public/v1 携带也无害：该节点忽略 Token）
 * - 抬头 X-Request-Id 便于跨端追踪；幂等写操作由调用方传 Idempotency-Key
 * - 业务码 10100 统一视为登录失效：清空本地会话，UI 导航回登录页
 */
class ApiClient(private val tokenStore: TokenStore) {
    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        coerceInputValues = true
    }

    private val client = OkHttpClient.Builder()
        .connectTimeout(Duration.ofSeconds(10))
        .readTimeout(Duration.ofSeconds(20))
        .callTimeout(Duration.ofSeconds(25))
        .retryOnConnectionFailure(true)
        .build()

    suspend fun <T> post(
        path: String,
        body: JsonObject = JsonObject(emptyMap()),
        deserializer: KSerializer<T>,
        idempotencyKey: String? = null,
    ): ApiResult<T> = execute(path, body.toString(), idempotencyKey) { data ->
        try {
            json.decodeFromJsonElement(deserializer, data)
        } catch (error: SerializationException) {
            throw IllegalStateException("响应字段解析失败：" + error.message, error)
        }
    }

    /** 无 data 的端点（signout / mark_read 等）：code=0 即成功。 */
    suspend fun postVoid(
        path: String,
        body: JsonObject = JsonObject(emptyMap()),
        idempotencyKey: String? = null,
    ): ApiResult<Unit> = execute(path, body.toString(), idempotencyKey) { Unit }

    private suspend fun <T> execute(
        path: String,
        bodyText: String,
        idempotencyKey: String?,
        parse: (JsonElement) -> T,
    ): ApiResult<T> = withContext(Dispatchers.IO) {
        val request = Request.Builder()
            .url(BuildConfig.API_BASE_URL + path)
            .post(bodyText.toRequestBody(JSON_MEDIA_TYPE))
            .header("Accept", "application/json")
            .header("X-Request-Id", UUID.randomUUID().toString())
            .apply {
                tokenStore.accessToken?.let { header("Authorization", "Bearer " + it) }
                idempotencyKey?.let { header("Idempotency-Key", it) }
            }
            .build()
        try {
            client.newCall(request).execute().use { response ->
                val text = response.body.string()
                if (!response.isSuccessful) {
                    return@withContext ApiResult.Err(
                        ApiError.Http(response.code, "请求失败（HTTP " + response.code + "）"),
                    )
                }
                val envelope = json.decodeFromString(ApiEnvelope.serializer(), text)
                if (envelope.code != 0) {
                    if (envelope.code == CODE_UNAUTHORIZED) tokenStore.clear()
                    return@withContext ApiResult.Err(
                        ApiError.Business(
                            code = envelope.code,
                            userMessage = businessErrorMessage(envelope.code, envelope.message),
                            forbiddenReason = readForbiddenReason(envelope.data),
                        ),
                    )
                }
                val data = envelope.data
                if (data == null) {
                    return@withContext ApiResult.Err(ApiError.Network("服务端返回数据为空"))
                }
                try {
                    ApiResult.Ok(parse(data))
                } catch (error: IllegalStateException) {
                    ApiResult.Err(ApiError.Network(error.message ?: "响应解析失败"))
                }
            }
        } catch (error: SocketTimeoutException) {
            ApiResult.Err(ApiError.Network("网络超时，请稍后重试"))
        } catch (error: IOException) {
            ApiResult.Err(ApiError.Network("网络连接失败，请检查网络后重试"))
        } catch (error: SerializationException) {
            ApiResult.Err(ApiError.Network("响应解析失败"))
        }
    }

    private fun readForbiddenReason(data: JsonElement?): String? = try {
        data?.jsonObject?.get("forbidden_reason")?.jsonPrimitive?.content
    } catch (_: Exception) {
        null
    }

    companion object {
        const val CODE_UNAUTHORIZED = 10100
        private val JSON_MEDIA_TYPE = "application/json; charset=utf-8".toMediaType()

        /** 构造空 JSON 对象（空请求体端点的契约要求：必须是 JSON 对象）。 */
        fun emptyBody(): JsonObject = buildJsonObject { }
    }
}

/**
 * 业务错误码 → 用户可读文案（spec/api_spec_errors.md HTTP 10000-19999）。
 * 抽成 top-level internal 便于纯 JVM 单测（不依赖 Android Context）。
 */
internal fun businessErrorMessage(code: Int, serverMessage: String?): String = when (code) {
    ApiClient.CODE_UNAUTHORIZED -> "登录已失效，请重新登录"
    10000 -> serverMessage ?: "参数错误"
    10200 -> serverMessage ?: "权限不足"
    10201 -> "账号已被封禁"
    10300 -> serverMessage ?: "内容不存在或已删除"
    10400 -> serverMessage ?: "操作无法完成"
    10401 -> serverMessage ?: "验证失败"
    10500 -> "请求过于频繁，请稍后再试"
    10600 -> "服务异常，请稍后再试"
    else -> serverMessage ?: ("请求失败（" + code + "）")
}

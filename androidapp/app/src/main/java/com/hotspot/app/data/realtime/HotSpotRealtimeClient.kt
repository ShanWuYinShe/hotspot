package com.hotspot.app.data.realtime

import android.util.Log
import com.hotspot.app.BuildConfig
import com.hotspot.app.data.api.ApiClient
import com.hotspot.app.data.api.ApiError
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.auth.AuthSession
import com.hotspot.app.data.auth.TokenStore
import com.hotspot.app.data.repo.AuthRepository
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeoutOrNull
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import java.net.URI
import java.time.Duration
import java.util.UUID
import kotlin.math.min
import kotlin.random.Random

/** WebSocket 连接状态（UI 可观测；供调试页/后续在线指示复用）。 */
enum class RealtimeConnectionState {
    Disconnected,
    Connecting,
    Connected,
}

/**
 * HotSpot WebSocket 客户端（spec/api_spec_websocket.md）。
 *
 * 连接 /ws/v1 后 5 秒内发送 Auth（app: access_token + device_id + resume_token）；
 * 读取 ConnectionEstablished.config 的动态心跳参数；断线指数退避重连并尝试 resume_token 增量恢复，
 * resume 不可用时按本地订阅集合全量 set 重放；close 4001 先刷新 token，失败清会话；
 * 1000/1008 不重连；SessionRevoked / AccountBanned 清会话并停止重连。
 */
class HotSpotRealtimeClient(
    private val tokenStore: TokenStore,
    private val authRepository: AuthRepository,
    private val scope: CoroutineScope,
    private val baseUrl: String = BuildConfig.API_BASE_URL,
) {
    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        coerceInputValues = true
    }

    private val httpClient = OkHttpClient.Builder()
        .connectTimeout(Duration.ofSeconds(15))
        .readTimeout(Duration.ofSeconds(0))
        .build()

    private val lock = Any()

    // webSocket/authenticated/resumeToken/pongDeferred 会被 OkHttp 回调线程与协程交叉读写，
    // 加 @Volatile 保证可见性（重连时 Auth 必须读到最新 resume_token，Pong 不能漏判超时）。
    @Volatile
    private var webSocket: WebSocket? = null
    private var connecting = false
    private var started = false
    private var closeRequested = false

    @Volatile
    private var authenticated = false
    private var reconnectAttempt = 0

    @Volatile
    private var resumeToken: String? = null
    private var pingIntervalMs = 30_000L
    private var pingTimeoutMs = 10_000L

    @Volatile
    private var pongDeferred: CompletableDeferred<Unit>? = null
    private var reconnectJob: Job? = null
    private var heartbeatJob: Job? = null
    private var authTimeoutJob: Job? = null

    /** 4002（连接级限流）前服务端 Error.retry_after 下发的建议等待时长（毫秒）。 */
    private var serverRetryAfterMs: Long? = null

    private val _connectionState = MutableStateFlow(RealtimeConnectionState.Disconnected)
    val connectionState: StateFlow<RealtimeConnectionState> = _connectionState.asStateFlow()

    private val _events = MutableSharedFlow<RealtimeEvent>(
        extraBufferCapacity = 128,
        onBufferOverflow = kotlinx.coroutines.channels.BufferOverflow.DROP_OLDEST,
    )
    val events: SharedFlow<RealtimeEvent> = _events.asSharedFlow()

    private val subscriptions = MutableStateFlow(RealtimeProtocol.SubscriptionSet())

    /** 登录态变化入口：null=登出，非 null=新会话（先断旧连接再连）。 */
    fun onSessionChanged(session: AuthSession?) {
        if (session == null) {
            stop()
        } else {
            stop()
            start()
        }
    }

    fun start() {
        synchronized(lock) {
            if (started) return
            if (tokenStore.accessToken.isNullOrBlank()) return
            started = true
            closeRequested = false
        }
        openSocket()
    }

    fun stop() {
        synchronized(lock) {
            started = false
            closeRequested = true
            reconnectJob?.cancel()
            reconnectJob = null
            cancelConnectionJobsLocked()
            webSocket?.close(1000, "client stop")
            webSocket = null
            connecting = false
            authenticated = false
            _connectionState.value = RealtimeConnectionState.Disconnected
        }
    }

    /** Activity onStart 调用：前台恢复时若连接不在线则立即补连（不等退避）。 */
    fun onForeground() {
        synchronized(lock) {
            if (!started || webSocket != null || tokenStore.accessToken.isNullOrBlank()) return
            reconnectJob?.cancel()
            reconnectJob = null
        }
        openSocket()
    }

    fun subscribeUsers(userIds: Collection<String>) {
        val ids = userIds.filter { it.isNotBlank() }.toSet()
        if (ids.isEmpty()) return
        val before = subscriptions.value
        val after = before.withUsers(ids)
        if (after == before) return
        subscriptions.value = after
        if (authenticated) sendSubscription("subscribe", userIds = ids, groupIds = emptySet())
    }

    fun unsubscribeUsers(userIds: Collection<String>) {
        val ids = userIds.filter { it.isNotBlank() }.toSet()
        if (ids.isEmpty()) return
        subscriptions.value = subscriptions.value.withoutUsers(ids)
        if (authenticated) sendSubscription("unsubscribe", userIds = ids, groupIds = emptySet())
    }

    fun subscribeGroups(groupIds: Collection<String>) {
        val ids = groupIds.filter { it.isNotBlank() }.toSet()
        if (ids.isEmpty()) return
        val before = subscriptions.value
        val after = before.withGroups(ids)
        if (after == before) return
        subscriptions.value = after
        if (authenticated) sendSubscription("subscribe", userIds = emptySet(), groupIds = ids)
    }

    fun unsubscribeGroups(groupIds: Collection<String>) {
        val ids = groupIds.filter { it.isNotBlank() }.toSet()
        if (ids.isEmpty()) return
        subscriptions.value = subscriptions.value.withoutGroups(ids)
        if (authenticated) sendSubscription("unsubscribe", userIds = emptySet(), groupIds = ids)
    }

    fun sendPrivateTyping(targetUserId: String, isTyping: Boolean) {
        if (!authenticated) return
        sendRaw(
            buildJsonObject {
                put("type", "SendPrivateTyping")
                put("payload", buildJsonObject {
                    put("target_user_id", targetUserId)
                    put("is_typing", isTyping)
                })
            },
        )
    }

    fun sendGroupTyping(groupId: String, isTyping: Boolean) {
        if (!authenticated) return
        sendRaw(
            buildJsonObject {
                put("type", "SendGroupTyping")
                put("payload", buildJsonObject {
                    put("group_id", groupId)
                    put("is_typing", isTyping)
                })
            },
        )
    }

    private fun openSocket() {
        synchronized(lock) {
            if (!started || closeRequested) return
            if (tokenStore.accessToken.isNullOrBlank()) return
            if (webSocket != null || connecting) return
            connecting = true
            _connectionState.value = RealtimeConnectionState.Connecting
        }
        val request = Request.Builder().url(websocketUrl()).build()
        val socket = httpClient.newWebSocket(request, Listener())
        synchronized(lock) {
            connecting = false
            if (!started || closeRequested) {
                // 建连期间被 stop()/会话切换：丢弃这条连接，避免泄漏一个仍会发 Auth 的 socket。
                socket.cancel()
                return
            }
            webSocket = socket
        }
    }

    private fun websocketUrl(): String {
        val uri = URI(baseUrl)
        val scheme = if (uri.scheme.equals("https", ignoreCase = true)) "wss" else "ws"
        val hostAndPort = if (uri.port > 0) uri.host + ":" + uri.port else uri.host
        return scheme + "://" + hostAndPort + "/ws/v1"
    }

    private fun sendAuth(socket: WebSocket) {
        val payload = buildJsonObject {
            put("client_type", "app")
            put("access_token", tokenStore.accessToken ?: "")
            put("device_id", tokenStore.deviceId())
            put("resume_token", resumeToken)
        }
        sendRaw(
            buildJsonObject {
                put("type", "Auth")
                put("payload", payload)
            },
            socket,
        )
    }

    private fun sendSubscription(operation: String, userIds: Set<String>, groupIds: Set<String>) {
        val items = buildJsonArray {
            if (userIds.isNotEmpty()) {
                add(
                    buildJsonObject {
                        put("type", "user")
                        put("ids", buildJsonArray { userIds.forEach { add(JsonPrimitive(it)) } })
                    },
                )
            }
            if (groupIds.isNotEmpty()) {
                add(
                    buildJsonObject {
                        put("type", "group")
                        put("ids", buildJsonArray { groupIds.forEach { add(JsonPrimitive(it)) } })
                    },
                )
            }
        }
        sendRaw(
            buildJsonObject {
                put("type", "SetSubscriptions")
                put("payload", buildJsonObject {
                    put("client_request_id", UUID.randomUUID().toString())
                    put("operation", operation)
                    put("items", items)
                })
            },
        )
    }

    private fun sendRaw(payload: JsonObject, socket: WebSocket? = null) {
        val target = socket ?: webSocket ?: return
        runCatching { target.send(payload.toString()) }
    }

    private fun handleEstablished(payload: JsonObject) {
        val (interval, timeout) = RealtimeProtocol.parseHeartbeat(payload)
        val resume = RealtimeProtocol.parseResumeToken(payload)
        synchronized(lock) {
            if (!started || closeRequested) return
            Log.i(TAG, "ws established ping=${interval}ms timeout=${timeout}ms")
            authenticated = true
            authTimeoutJob?.cancel()
            authTimeoutJob = null
            reconnectAttempt = 0
            pingIntervalMs = interval
            pingTimeoutMs = timeout
            resumeToken = resume
            serverRetryAfterMs = null
            _connectionState.value = RealtimeConnectionState.Connected
        }
        val current = subscriptions.value
        sendSubscription("set", current.userIds, current.groupIds)
        startHeartbeat()
    }

    private fun startHeartbeat() {
        synchronized(lock) { heartbeatJob?.cancel() }
        val job = scope.launch {
            while (isActive) {
                delay(pingIntervalMs)
                if (!authenticated) break
                val deferred = CompletableDeferred<Unit>()
                pongDeferred = deferred
                sendRaw(
                    buildJsonObject {
                        put("type", "Ping")
                        put("payload", buildJsonObject { })
                    },
                )
                val pong = withTimeoutOrNull(pingTimeoutMs) { deferred.await() }
                if (pong == null) {
                    val socket = webSocket
                    // 心跳超时视为连接已死：先终止底层连接，避免僵尸 socket 与新连接并存、
                    // 继续把旧连接的事件投递给 UI（表现为消息重复）。
                    socket?.cancel()
                    handleDisconnect(socket, 0)
                    break
                }
            }
        }
        synchronized(lock) { heartbeatJob = job }
    }

    private fun handleMessage(text: String) {
        val root = runCatching { json.parseToJsonElement(text).jsonObject }.getOrNull() ?: return
        val type = root["type"]?.jsonPrimitive?.content ?: return
        val payload = root["payload"]?.jsonObject ?: JsonObject(emptyMap())
        when (type) {
            "ConnectionEstablished" -> handleEstablished(payload)
            "Pong" -> {
                pongDeferred?.complete(Unit)
                pongDeferred = null
            }
            "Error" -> {
                val severity = payload["severity"]?.jsonPrimitive?.content
                val code = payload["code"]?.jsonPrimitive?.content?.toIntOrNull()
                val retryAfterSeconds = payload["retry_after"]?.jsonPrimitive?.content?.toLongOrNull()
                if (code == CODE_WS_RATE_LIMIT && retryAfterSeconds != null) {
                    // 4002 连接级限流：按 spec 在重连前等待 retry_after（加上限，防止异常值钉死客户端）。
                    synchronized(lock) {
                        serverRetryAfterMs = retryAfterSeconds.coerceIn(1L, 300L) * 1_000L
                    }
                }
                if (severity == "fatal") {
                    // spec「Close Code 规范」：access_token 过期同样先推 fatal Error 再以 4001 断开。
                    // 此处不能清会话或停连，否则 4001 的 refresh_token 分支永远不可达，用户会被误登出；
                    // 真正的登出交给 SessionRevoked / AccountBanned 与 4001 刷新失败（10100）分支。
                    val message = payload["message"]?.jsonPrimitive?.content
                    Log.w(TAG, "ws fatal error code=" + code + " message=" + message)
                }
            }
            "SessionRevoked" -> {
                _events.tryEmit(
                    RealtimeEvent.SessionRevoked(payload["reason"]?.jsonPrimitive?.content ?: ""),
                )
                stop()
                tokenStore.clear()
            }
            "AccountBanned" -> {
                _events.tryEmit(
                    RealtimeEvent.AccountBanned(payload["reason"]?.jsonPrimitive?.content ?: ""),
                )
                stop()
                tokenStore.clear()
            }
            else -> RealtimeProtocol.parse(type, payload)?.let {
                Log.d(TAG, "ws event " + type)
                _events.tryEmit(it)
            }
        }
    }

    private fun handleDisconnect(socket: WebSocket?, code: Int) {
        synchronized(lock) {
            if (socket != null && webSocket !== socket) return
            webSocket = null
            connecting = false
            authenticated = false
            cancelConnectionJobsLocked()
            _connectionState.value = RealtimeConnectionState.Disconnected
            Log.i(TAG, "ws disconnected code=" + code + " started=" + started)
            if (!started || closeRequested) return
            when (code) {
                1000, 1008 -> {
                    started = false
                    closeRequested = true
                }
                4001 -> scheduleTokenRefresh()
                4002 -> {
                    val retryAfter = serverRetryAfterMs
                    serverRetryAfterMs = null
                    scheduleReconnect(delayOverrideMs = retryAfter)
                }
                else -> scheduleReconnect()
            }
        }
    }

    private fun scheduleReconnect(delayOverrideMs: Long? = null) {
        if (reconnectJob != null) return
        val attempt = reconnectAttempt
        reconnectAttempt = min(reconnectAttempt + 1, 10)
        val jitter = Random.nextLong(0, 500)
        val delayMs = delayOverrideMs ?: RealtimeProtocol.reconnectDelayMs(attempt, jitter)
        Log.i(TAG, "ws reconnect in " + delayMs + "ms attempt=" + attempt)
        reconnectJob = scope.launch(Dispatchers.Default) {
            delay(delayMs)
            synchronized(lock) { reconnectJob = null }
            openSocket()
        }
    }

    private fun scheduleTokenRefresh() {
        if (reconnectJob != null) return
        reconnectJob = scope.launch(Dispatchers.IO) {
            val result = authRepository.refresh()
            synchronized(lock) { reconnectJob = null }
            when (result) {
                is ApiResult.Ok -> openSocket()
                is ApiResult.Err -> {
                    val unauthorized = (result.error as? ApiError.Business)?.code == ApiClient.CODE_UNAUTHORIZED
                    if (unauthorized || tokenStore.session.value == null) {
                        tokenStore.clear()
                        stop()
                    } else {
                        // 网络/服务端临时故障不代表会话失效：保留会话并退避重连；
                        // 若 access_token 仍过期，重连鉴权会再次得到 4001 并重试刷新。
                        Log.w(TAG, "ws token refresh failed: " + result.error.userMessage)
                        synchronized(lock) { scheduleReconnect() }
                    }
                }
            }
        }
    }

    private fun cancelConnectionJobsLocked() {
        heartbeatJob?.cancel()
        heartbeatJob = null
        authTimeoutJob?.cancel()
        authTimeoutJob = null
        pongDeferred = null
    }

    /** 回调所属连接是否仍是当前连接（旧连接/被替换连接的迟到消息直接丢弃）。 */
    private fun isCurrentSocket(socket: WebSocket): Boolean = synchronized(lock) { webSocket === socket }

    private inner class Listener : WebSocketListener() {
        override fun onOpen(webSocket: WebSocket, response: Response) {
            if (!isCurrentSocket(webSocket)) {
                webSocket.close(1000, "stale connection")
                return
            }
            sendAuth(webSocket)
            synchronized(lock) {
                authTimeoutJob?.cancel()
                authTimeoutJob = scope.launch {
                    delay(AUTH_TIMEOUT_MS)
                    if (!authenticated && isCurrentSocket(webSocket)) {
                        // 本地 5 秒未收到 ConnectionEstablished 属于连接异常（token 失效会被服务端
                        // 以 Error + 4001 明确拒绝），按通用退避重连，不触发不必要的 token 刷新。
                        webSocket.cancel()
                        handleDisconnect(webSocket, 0)
                    }
                }
            }
        }

        override fun onMessage(webSocket: WebSocket, text: String) {
            // 心跳超时/被替换的旧连接可能还有迟到消息，不能投递给 UI（否则新连接会重复收事件）。
            if (!isCurrentSocket(webSocket)) return
            handleMessage(text)
        }

        override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
            // 服务端 close code 决定是否重连（4001=刷新 token、1000/1008=不重连）；
            // 先按原 code 处理断开，再完成关闭握手，避免 onClosed 收到 1000 覆盖原语义。
            handleDisconnect(webSocket, code)
            webSocket.close(1000, null)
        }

        override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
            handleDisconnect(webSocket, code)
        }

        override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
            handleDisconnect(webSocket, 0)
        }
    }

    private companion object {
        const val TAG = "HotSpotWS"

        /** 鉴权必须在连接后 5 秒内完成（spec「连接与鉴权」）。 */
        const val AUTH_TIMEOUT_MS = 5_000L

        /** WS 应用层错误码：连接级限流（spec/api_spec_websocket.md「错误码定义」）。 */
        const val CODE_WS_RATE_LIMIT = 20300
    }
}

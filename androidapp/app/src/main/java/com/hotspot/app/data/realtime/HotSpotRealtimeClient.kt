package com.hotspot.app.data.realtime

import android.util.Log
import com.hotspot.app.BuildConfig
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
    private var webSocket: WebSocket? = null
    private var connecting = false
    private var started = false
    private var closeRequested = false
    private var authenticated = false
    private var reconnectAttempt = 0
    private var resumeToken: String? = null
    private var pingIntervalMs = 30_000L
    private var pingTimeoutMs = 10_000L
    private var pongDeferred: CompletableDeferred<Unit>? = null
    private var reconnectJob: Job? = null
    private var heartbeatJob: Job? = null
    private var authTimeoutJob: Job? = null

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
        val token: String
        synchronized(lock) {
            if (!started || closeRequested) return
            if (tokenStore.accessToken.isNullOrBlank()) return
            if (webSocket != null || connecting) return
            token = tokenStore.accessToken ?: return
            connecting = true
            _connectionState.value = RealtimeConnectionState.Connecting
        }
        val request = Request.Builder().url(websocketUrl()).build()
        val socket = httpClient.newWebSocket(request, Listener())
        synchronized(lock) {
            connecting = false
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
        Log.i(TAG, "ws established ping=${interval}ms timeout=${timeout}ms")
        synchronized(lock) {
            authenticated = true
            authTimeoutJob?.cancel()
            authTimeoutJob = null
            reconnectAttempt = 0
            pingIntervalMs = interval
            pingTimeoutMs = timeout
            resumeToken = payload["resume_token"]?.jsonPrimitive?.content
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
                if (severity == "fatal") {
                    stop()
                    tokenStore.clear()
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
                else -> scheduleReconnect()
            }
        }
    }

    private fun scheduleReconnect() {
        if (reconnectJob != null) return
        val attempt = reconnectAttempt
        reconnectAttempt = min(reconnectAttempt + 1, 10)
        val jitter = Random.nextLong(0, 500)
        val delayMs = RealtimeProtocol.reconnectDelayMs(attempt, jitter)
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
                    tokenStore.clear()
                    stop()
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

    private inner class Listener : WebSocketListener() {
        override fun onOpen(webSocket: WebSocket, response: Response) {
            sendAuth(webSocket)
            synchronized(lock) {
                authTimeoutJob?.cancel()
                authTimeoutJob = scope.launch {
                    delay(AUTH_TIMEOUT_MS)
                    if (!authenticated) {
                        webSocket.close(4001, "auth timeout")
                        handleDisconnect(webSocket, 4001)
                    }
                }
            }
        }

        override fun onMessage(webSocket: WebSocket, text: String) {
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
    }
}

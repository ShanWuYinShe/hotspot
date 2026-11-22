package com.hotspot.app

import android.content.Context
import com.hotspot.app.data.api.ApiClient
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.auth.TokenStore
import com.hotspot.app.data.model.AppLimits
import com.hotspot.app.data.model.ClientConfig
import com.hotspot.app.data.model.DefaultLocation
import com.hotspot.app.data.repo.AuthRepository
import com.hotspot.app.data.repo.ChatRepository
import com.hotspot.app.data.repo.CommentRepository
import com.hotspot.app.data.repo.FeedRepository
import com.hotspot.app.data.repo.GroupRepository
import com.hotspot.app.data.repo.MediaRepository
import com.hotspot.app.data.repo.MomentsRepository
import com.hotspot.app.data.repo.NearbyRepository
import com.hotspot.app.data.repo.NotifyRepository
import com.hotspot.app.data.repo.ProfileRepository
import com.hotspot.app.data.repo.ReportsRepository
import com.hotspot.app.data.realtime.HotSpotRealtimeClient
import com.hotspot.app.data.repo.SearchRepository
import com.hotspot.app.data.repo.SystemRepository
import com.hotspot.app.ui.theme.ThemeController
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

/**
 * 应用级依赖容器（无 DI 框架）：Application 启动时初始化一次，Compose 层直接取用。
 * 保持轻量：网络客户端 / 会话存储 / 主题偏好 / 各业务仓库。
 */
object AppContainer {
    lateinit var tokenStore: TokenStore
        private set
    lateinit var themeController: ThemeController
        private set
    lateinit var apiClient: ApiClient
        private set
    lateinit var authRepository: AuthRepository
        private set
    lateinit var feedRepository: FeedRepository
        private set
    lateinit var nearbyRepository: NearbyRepository
        private set
    lateinit var searchRepository: SearchRepository
        private set
    lateinit var chatRepository: ChatRepository
        private set
    lateinit var profileRepository: ProfileRepository
        private set
    lateinit var notifyRepository: NotifyRepository
        private set
    lateinit var commentRepository: CommentRepository
        private set
    lateinit var groupRepository: GroupRepository
        private set
    lateinit var systemRepository: SystemRepository
        private set
    lateinit var reportsRepository: ReportsRepository
        private set
    lateinit var momentsRepository: MomentsRepository
        private set
    lateinit var mediaRepository: MediaRepository
        private set

    /** WebSocket 实时增量通道（登录态驱动；聊天页/消息列表消费 events）。 */
    lateinit var realtimeClient: HotSpotRealtimeClient
        private set

    /** 应用级协程作用域：WebSocket 重连/心跳等常驻任务。 */
    private val appScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    @Volatile
    private var initialized = false

    fun init(context: Context) {
        if (initialized) return
        synchronized(this) {
            if (initialized) return
            val appContext = context.applicationContext
            tokenStore = TokenStore(appContext)
            themeController = ThemeController(appContext)
            apiClient = ApiClient(tokenStore)
            authRepository = AuthRepository(apiClient, tokenStore)
            feedRepository = FeedRepository(apiClient)
            nearbyRepository = NearbyRepository(apiClient)
            searchRepository = SearchRepository(apiClient)
            chatRepository = ChatRepository(apiClient)
            profileRepository = ProfileRepository(apiClient)
            notifyRepository = NotifyRepository(apiClient)
            commentRepository = CommentRepository(apiClient)
            groupRepository = GroupRepository(apiClient)
            systemRepository = SystemRepository(apiClient)
            reportsRepository = ReportsRepository(apiClient)
            momentsRepository = MomentsRepository(apiClient)
            mediaRepository = MediaRepository(apiClient)
            realtimeClient = HotSpotRealtimeClient(tokenStore, authRepository, appScope)
            // 登录态驱动连接生命周期：登出/会话失效自动断开，登录后自动连接并恢复订阅。
            appScope.launch {
                tokenStore.session.collect { session -> realtimeClient.onSessionChanged(session) }
            }
            initialized = true
        }
    }

    // 客户端公开配置缓存（client_config）：默认浏览位置 + 业务限额（发布器）。
    private var cachedClientConfig: ClientConfig? = null
    private var clientConfigRequested = false

    /** 首次调用拉取并缓存 client_config；失败置回未请求，下次调用重试。 */
    suspend fun clientConfig(): ClientConfig? {
        if (!clientConfigRequested) {
            clientConfigRequested = true
            when (val result = systemRepository.clientConfig()) {
                is ApiResult.Ok -> cachedClientConfig = result.data
                is ApiResult.Err -> clientConfigRequested = false
            }
        }
        return cachedClientConfig
    }

    /** 浏览锚点（reference_point 回退链第 3 级）：client_config.defaults.default_location。 */
    suspend fun defaultLocation(): DefaultLocation? = clientConfig()?.defaults?.defaultLocation

    /** 发布器限额；配置不可用时退回 spec 默认值（9 图 / 9 标签 / 280 字）。 */
    suspend fun limits(): AppLimits = clientConfig()?.limits ?: AppLimits()
}

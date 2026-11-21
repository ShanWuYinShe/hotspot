package com.hotspot.app

import android.content.Context
import com.hotspot.app.data.api.ApiClient
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.data.auth.TokenStore
import com.hotspot.app.data.model.DefaultLocation
import com.hotspot.app.data.repo.AuthRepository
import com.hotspot.app.data.repo.ChatRepository
import com.hotspot.app.data.repo.CommentRepository
import com.hotspot.app.data.repo.FeedRepository
import com.hotspot.app.data.repo.GroupRepository
import com.hotspot.app.data.repo.NearbyRepository
import com.hotspot.app.data.repo.NotifyRepository
import com.hotspot.app.data.repo.ProfileRepository
import com.hotspot.app.data.repo.ReportsRepository
import com.hotspot.app.data.repo.SearchRepository
import com.hotspot.app.data.repo.SystemRepository
import com.hotspot.app.ui.theme.ThemeController

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
            initialized = true
        }
    }

    // 浏览锚点缓存：client_config.defaults.default_location（reference_point 回退链第 3 级）。
    private var cachedDefaultLocation: DefaultLocation? = null
    private var defaultLocationRequested = false

    /** 首次调用拉取并缓存默认浏览位置；失败置回未请求，下次调用重试。 */
    suspend fun defaultLocation(): DefaultLocation? {
        if (!defaultLocationRequested) {
            defaultLocationRequested = true
            when (val result = systemRepository.clientConfig()) {
                is ApiResult.Ok -> cachedDefaultLocation = result.data.defaults.defaultLocation
                is ApiResult.Err -> defaultLocationRequested = false
            }
        }
        return cachedDefaultLocation
    }
}

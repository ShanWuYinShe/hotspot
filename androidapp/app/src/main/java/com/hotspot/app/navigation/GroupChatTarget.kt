package com.hotspot.app.navigation

/**
 * 群聊页入口参数（与 ChatTarget 同构）：路由只带 groupId，头部展示所需的群资料由入口页写入。
 * 群详情 → 打开群聊时 select；返回时 clear，避免下次进入残留上一个群。
 */
object GroupChatTarget {
    var groupId: String = ""
        private set
    var name: String = ""
        private set
    var avatarUrl: String? = null
        private set
    var membersCount: Int = 0
        private set
    var onlineCount: Int = 0
        private set

    fun select(
        groupId: String,
        name: String,
        avatarUrl: String?,
        membersCount: Int,
        onlineCount: Int,
    ) {
        this.groupId = groupId
        this.name = name
        this.avatarUrl = avatarUrl
        this.membersCount = membersCount
        this.onlineCount = onlineCount
    }

    fun clear() {
        groupId = ""
        name = ""
        avatarUrl = null
        membersCount = 0
        onlineCount = 0
    }
}

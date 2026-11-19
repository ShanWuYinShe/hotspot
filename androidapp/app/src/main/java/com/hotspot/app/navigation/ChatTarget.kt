package com.hotspot.app.navigation

/** 当前打开的私聊对象（navigation-compose 字符串路由只带 userId，昵称用于页头展示）。 */
object ChatTarget {
    var userId: String = ""
        private set
    var nickname: String = ""
        private set

    fun select(userId: String, nickname: String) {
        this.userId = userId
        this.nickname = nickname
    }

    fun clear() {
        userId = ""
        nickname = ""
    }
}

package com.hotspot.app

import android.app.Application

class HotSpotApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        AppContainer.init(this)
    }
}

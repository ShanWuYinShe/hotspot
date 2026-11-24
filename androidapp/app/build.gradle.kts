import java.net.URI

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.plugin.serialization")
    id("org.jetbrains.kotlin.plugin.compose")
}

// API 网关地址：debug 默认指向模拟器回环（10.0.2.2 → 宿主机 nginx 网关 8888），
// 真机联调用 `adb reverse tcp:8888 tcp:8888` 后在开发者设置里改成 http://127.0.0.1:8888。
// release 通过 Gradle 属性 / 环境变量覆盖，默认线上站点。
val debugBaseUrl = providers.gradleProperty("HOTSPOT_DEBUG_BASE_URL")
    .orElse(providers.environmentVariable("HOTSPOT_DEBUG_BASE_URL"))
    .orElse("http://10.0.2.2:8888")
    .get()
val releaseBaseUrl = providers.gradleProperty("HOTSPOT_BASE_URL")
    .orElse(providers.environmentVariable("HOTSPOT_BASE_URL"))
    .orElse("https://chaoset.com")
    .get()

fun validateBaseUrl(value: String, allowCleartext: Boolean): String {
    val uri = URI(value.trim())
    val scheme = uri.scheme?.lowercase()
    require(uri.host != null && scheme in setOf("http", "https")) { "Base URL 非法：$value" }
    require(allowCleartext || scheme == "https") { "release Base URL 必须为 HTTPS：$value" }
    return value.trim().trimEnd('/')
}
val debugUrl = validateBaseUrl(debugBaseUrl, allowCleartext = true)
val releaseUrl = validateBaseUrl(releaseBaseUrl, allowCleartext = false)

android {
    namespace = "com.hotspot.app"
    compileSdk = 37
    buildToolsVersion = "37.0.0"

    defaultConfig {
        applicationId = "com.hotspot.app"
        minSdk = 26
        targetSdk = 37
        // 版本号：默认与仓库发布 tag 对齐；发新版时通过环境变量覆盖即可，无需改代码
        // （HOTSPOT_VERSION_NAME=v1.15.1 HOTSPOT_VERSION_CODE=7 ./gradlew assembleRelease）。
        versionCode = providers.environmentVariable("HOTSPOT_VERSION_CODE").orNull?.toIntOrNull() ?: 7
        versionName = providers.environmentVariable("HOTSPOT_VERSION_NAME").orNull?.removePrefix("v") ?: "1.15.1"

        buildConfigField("String", "API_BASE_URL", "\"$debugUrl\"")
    }

    buildFeatures {
        buildConfig = true
        compose = true
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    // 发布签名：材料不进仓库。提供 4 个环境变量时启用（见 docs/release-runbook.md「Android APK 分发」）；
    // 未提供时 release 产物保持未签名（CI 只跑 debug，不受影响）。
    val releaseKeystoreFile = providers.environmentVariable("HOTSPOT_KEYSTORE_FILE").orNull
    val releaseKeystorePassword = providers.environmentVariable("HOTSPOT_KEYSTORE_PASSWORD").orNull
    val releaseKeyAlias = providers.environmentVariable("HOTSPOT_KEY_ALIAS").orNull
    val releaseKeyPassword = providers.environmentVariable("HOTSPOT_KEY_PASSWORD").orNull
    val hasReleaseSigning = listOf(
        releaseKeystoreFile,
        releaseKeystorePassword,
        releaseKeyAlias,
        releaseKeyPassword,
    ).all { !it.isNullOrBlank() }

    signingConfigs {
        if (hasReleaseSigning) {
            create("release") {
                storeFile = file(releaseKeystoreFile!!)
                storePassword = releaseKeystorePassword
                keyAlias = releaseKeyAlias
                keyPassword = releaseKeyPassword
            }
        }
    }

    buildTypes {
        debug {
            // 模拟器访问宿主 nginx 网关；真机 adb reverse 后用 127.0.0.1:8888 覆盖。
            buildConfigField("String", "API_BASE_URL", "\"$debugUrl\"")
        }
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
            buildConfigField("String", "API_BASE_URL", "\"$releaseUrl\"")
            if (hasReleaseSigning) {
                signingConfig = signingConfigs.getByName("release")
            }
        }
    }

    // 纯 JVM 单测会触达 android.util.Log：返回默认值让日志成为 no-op，而不是抛 not mocked。
    testOptions {
        unitTests.isReturnDefaultValues = true
    }

    packaging {
        resources {
            excludes += "/META-INF/{AL2.0,LGPL2.1}"
        }
    }
}

// AGP 9 不再使用 kotlinOptions{}（已废弃），统一走顶层 kotlin{compilerOptions{}}。
kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

dependencies {
    val composeBom = platform("androidx.compose:compose-bom:2025.12.01")
    implementation(composeBom)
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.compose.foundation:foundation")
    implementation("androidx.compose.material3:material3")
    // material3 1.4 起不再传递引入 icons-core，需显式声明
    implementation("androidx.compose.material:material-icons-core")
    implementation("androidx.activity:activity-compose:1.13.0")
    implementation("androidx.navigation:navigation-compose:2.9.0")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.11.0")
    implementation("androidx.lifecycle:lifecycle-viewmodel-ktx:2.11.0")
    implementation("androidx.lifecycle:lifecycle-runtime-compose:2.11.0")
    implementation("io.coil-kt:coil-compose:2.7.0")
    implementation("com.squareup.okhttp3:okhttp:5.4.0")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.11.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.11.0")

    debugImplementation("androidx.compose.ui:ui-tooling")

    testImplementation("junit:junit:4.13.2")
}

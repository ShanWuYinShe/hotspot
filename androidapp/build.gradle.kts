// 根构建脚本：插件版本与本机已验证可构建组合保持一致
// （AGP 9.2.1 + Kotlin 2.4.0 + Gradle 9.7.0，见 androidapp/README.md）。
plugins {
    id("com.android.application") version "9.2.1" apply false
    id("org.jetbrains.kotlin.plugin.serialization") version "2.4.0" apply false
    id("org.jetbrains.kotlin.plugin.compose") version "2.4.0" apply false
}

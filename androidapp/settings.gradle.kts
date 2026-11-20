pluginManagement {
    repositories {
        google()
        // Maven Central 的 Google Cloud 官方镜像（内容与 Central 同源、全球 CDN）。
        // 背景：GitHub Actions 出网访问 repo.maven.apache.org 偶发 404/限流，导致
        // 构建脚本 classpath 解析不到 kotlin-stdlib 等公共制品而失败；镜像作为第一顺位
        // 兜底（本地构建同样可用，国内网络更稳）。保留 mavenCentral() 作为后备。
        maven("https://maven-central.storage-download.googleapis.com/maven2")
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        maven("https://maven-central.storage-download.googleapis.com/maven2")
        mavenCentral()
    }
}

rootProject.name = "HotSpot"
include(":app")

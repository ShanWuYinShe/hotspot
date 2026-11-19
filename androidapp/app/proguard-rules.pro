# HotSpot Android R8 规则（release 开启 minify + resource shrink）
# kotlinx.serialization：保留生成的 serializer
-keepattributes *Annotation*, InnerClasses
-dontnote kotlinx.serialization.**
-keepclassmembers class **$$serializer { *; }
-keepclasseswithmembers class * {
    kotlinx.serialization.KSerializer serializer(...);
}
-keepclassmembers class com.hotspot.app.data.model.** {
    *** Companion;
}
-keepclasseswithmembers class com.hotspot.app.data.model.** {
    kotlinx.serialization.KSerializer serializer(...);
}
# OkHttp 平台适配（可选依赖缺失时告警但不影响运行）
-dontwarn okhttp3.**
-dontwarn okio.**
-dontwarn org.conscrypt.**

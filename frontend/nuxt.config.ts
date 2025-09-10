// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  devtools: { enabled: true },
  compatibilityDate: "2025-05-28",
  future: {
    compatibilityVersion: 4, // 启用 Nuxt 4 特性
  },

  modules: [
    "@pinia/nuxt",
    "pinia-plugin-persistedstate/nuxt",
    "@nuxt/ui",
    "@nuxtjs/color-mode",
    "@nuxtjs/i18n"
  ],

  // 组件自动导入（含 app/components 与插画目录、虚拟滚动组件）
  components: [
    { path: "~/components", pathPrefix: false },
    { path: "~/components/illustrations", pathPrefix: false },
  ],

  // Pinia 持久化配置
  piniaPluginPersistedstate: {
    storage: "localStorage",
    debug: process.env.NODE_ENV !== "production",
  },

  // SSR 配置
  ssr: true,

  // Nuxt UI 配置
  ui: {
    fonts: false
  },

  colorMode: {
    classSuffix: "",
    preference: "system",
    fallback: "light",
    storage: "localStorage"
  },

  i18n: {
    defaultLocale: 'zh-CN',
    strategy: 'no_prefix',
    lazy: true,
    locales: [
      { code: 'zh-CN', name: '中文', file: 'zh-CN.json' },
      { code: 'en-US', name: 'English', file: 'en-US.json' }
    ]
  },

  // 样式配置
  css: [
    "~/assets/styles/main.css",
    "~/assets/styles/mobile.css"
  ],

  // TypeScript 配置
  typescript: {
    strict: true,
    typeCheck: false,
  },

  // 实验性特性
  experimental: {
    typedPages: false,
    appManifest: true,
  },


  runtimeConfig: {
    public: {
      apiBaseUrl:
        process.env.NUXT_PUBLIC_API_BASE_URL || "http://localhost:8080/api/v1",
      wsBaseUrl:
        process.env.NUXT_PUBLIC_WS_BASE_URL || "ws://localhost:8080/ws/",
      amapKey: process.env.NUXT_PUBLIC_AMAP_KEY || "",
      amapServiceSecurityCode:
        process.env.NUXT_PUBLIC_AMAP_SERVICE_SECURITY_CODE || "",
      amapServiceHost: process.env.NUXT_PUBLIC_AMAP_SERVICE_HOST || "",
      wechatAppId: process.env.NUXT_PUBLIC_WECHAT_APPID || "",
      shareDefaultImage: process.env.NUXT_PUBLIC_SHARE_DEFAULT_IMAGE || "",
      siteUrl: process.env.NUXT_PUBLIC_SITE_URL || "https://hotspot.example.com"
    },
  },
});

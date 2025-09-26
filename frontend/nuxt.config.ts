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

  // 自动导入配置 - 使用 Nuxt 默认配置，不包含非内置目录
  // Nuxt 默认自动导入：composables/, utils/, stores/ (via @pinia/nuxt)
  // services/ 和 config/ 需要手动导入以保持代码依赖关系清晰

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
    "~/assets/styles/transitions.css"
  ],

  // TypeScript 配置
  typescript: {
    strict: true,
    typeCheck: false,
  },

  // Nuxt v4 实验性特性
  experimental: {
    // 启用类型化页面路由
    typedPages: true,
    // 启用应用清单
    appManifest: true,
    // 启用视图过渡 API
    viewTransition: true,
    // 启用 payload 提取
    payloadExtraction: false,
    // 启用 watcher 忽略
    watcher: 'chokidar',
    // 启用 sharedPrerenderData
    sharedPrerenderData: true,
  },

  // Nuxt v4 构建优化
  build: {
    // 启用 analyze 模式
    analyze: process.env.ANALYZE === 'true'
  },

  // Nuxt v4 Vite 配置
  vite: {
    // 启用 Vue DevTools
    vue: {
      script: {
        defineModel: true,
        propsDestructure: true
      }
    },
    // 优化依赖
    optimizeDeps: {
      include: ['@headlessui/vue', '@heroicons/vue/24/outline', '@heroicons/vue/24/solid']
    }
  },

  // Nuxt v4 Nitro 配置
  nitro: {
    // 启用实验性功能
    experimental: {
      wasm: true
    },
    // 预渲染配置
    prerender: {
      crawlLinks: true,
      routes: ['/']
    }
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

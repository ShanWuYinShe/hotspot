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
    "@nuxt/image",
    "@nuxtjs/color-mode",
    "@nuxtjs/i18n",
    "@vueuse/nuxt"
  ],

  // 组件自动导入（含 app/components 与插画目录、虚拟滚动组件）
  components: [
    { path: "~/components", pathPrefix: false },
  ],

  // 自动导入配置 - 使用 Nuxt 默认配置，不包含非内置目录
  // Nuxt 默认自动导入：composables/, utils/, stores/ (via @pinia/nuxt)
  // services/ 和 config/ 需要手动导入以保持代码依赖关系清晰

  // Pinia 持久化配置
  // 使用 localStorage 存储（仅客户端）
  piniaPluginPersistedstate: {
    storage: 'localStorage',
    debug: import.meta.env.NODE_ENV !== 'production',
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
    "~/assets/styles/base.css",
  ],

  postcss: {
    plugins: {
      "@tailwindcss/postcss": {},
      autoprefixer: {},
    },
  },

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
    // 视图过渡在本项目中会导致部分页面跳转超时并抛出 DOMException（影响实际跳转）。
    // 先禁用，确保导航可靠；如未来需要再按页面/组件精细开启。
    viewTransition: false,
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
    analyze: import.meta.env.ANALYZE === 'true'
  },

  // Nuxt v4 Vite 配置
  vite: {
    // 构建配置（消除 Tailwind CSS sourcemap 警告）
    build: {
      // 生产环境禁用 sourcemap，减小构建体积并消除警告
      sourcemap: import.meta.env.NODE_ENV !== 'production',
    },
    // 同源环境：统一由 Nginx 做反代（见 docs/hotspot.test.nginx.conf，可直接用于真实环境）
    server: {
      allowedHosts: true,
    },
  },

  // Nuxt v4 Nitro 配置
  nitro: {
    experimental: {
      wasm: true,
    }
  },

  runtimeConfig: {
    public: {
      // 高德 JS API：key 必须显式配置；安全密钥通过 Nginx /_AMapService 代理注入（前端不再持有 securityJsCode）
      amapKey: import.meta.env.NUXT_PUBLIC_AMAP_KEY || "d6e827b4530d8a87f745f3f1f237cb1d",
      // 可选：手动指定代理域名；不填时默认使用当前站点 origin + /_AMapService
      amapServiceHost: import.meta.env.NUXT_PUBLIC_AMAP_SERVICE_HOST || "/_AMapService",
      wechatAppId: import.meta.env.NUXT_PUBLIC_WECHAT_APPID || "",
      shareDefaultImage: import.meta.env.NUXT_PUBLIC_SHARE_DEFAULT_IMAGE || "",
      siteUrl: import.meta.env.NUXT_PUBLIC_SITE_URL || ""
    },
  },
});
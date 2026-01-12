// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  devtools: { enabled: true },
  compatibilityDate: "2025-05-28",
  future: {
    compatibilityVersion: 4, // 启用 Nuxt 4 特性
  },

  app: {
    head: {
      /**
       * Favicon / App icons
       *
       * Note:
       * - Browsers decide the *mask shape* (square/rounded) in the tab UI; you cannot force a true circular favicon via CSS.
       * - To make it appear circular, use a circular logo on a square canvas (or transparent background) as the icon asset.
       *
       * Place these files under `frontend/public/`:
       * - /logo_64.png
       * - /logo_32.png
       * - /logo_16.png
       * - /logo_180.png
       */
      link: [
        { rel: "icon", type: "image/x-icon", href: "/logo_64.png" },
        {
          rel: "icon",
          type: "image/png",
          sizes: "32x32",
          href: "/logo_32.png",
        },
        {
          rel: "icon",
          type: "image/png",
          sizes: "16x16",
          href: "/logo_16.png",
        },
        {
          rel: "apple-touch-icon",
          sizes: "180x180",
          href: "/logo_180.png",
        },
      ],
    },
  },

  modules: [
    "@pinia/nuxt",
    "pinia-plugin-persistedstate/nuxt",
    "@nuxt/ui",
    "@nuxt/image",
    "@nuxtjs/color-mode",
    "@nuxtjs/i18n",
    "@vueuse/nuxt",
  ],

  icon: {
    serverBundle: {
      collections: ["lucide"],
    },
    clientBundle: {
      scan: true,
    },
  },

  // 组件自动导入（含 app/components 与插画目录、虚拟滚动组件）
  components: [{ path: "~/components", pathPrefix: false }],

  // 自动导入配置 - 使用 Nuxt 默认配置，不包含非内置目录
  // Nuxt 默认自动导入：composables/, utils/, stores/ (via @pinia/nuxt)
  // services/ 和 config/ 需要手动导入以保持代码依赖关系清晰

  // Pinia 持久化配置
  // 使用 localStorage 存储（仅客户端）
  piniaPluginPersistedstate: {
    storage: "localStorage",
    debug: import.meta.env.NODE_ENV !== "production",
  },

  // SSR 配置
  ssr: true,

  // Nuxt UI 配置
  ui: {
    fonts: false,
  },

  colorMode: {
    classSuffix: "",
    preference: "system",
    fallback: "light",
    storage: "localStorage",
  },

  i18n: {
    defaultLocale: "zh-CN",
    strategy: "no_prefix",
    lazy: true,
    locales: [
      { code: "zh-CN", name: "中文", file: "zh-CN.json" },
      { code: "en-US", name: "English", file: "en-US.json" },
    ],
  },

  // 样式配置
  css: ["~/assets/styles/base.css"],

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
    watcher: "chokidar",
    // 启用 sharedPrerenderData
    sharedPrerenderData: true,
  },

  // Nuxt v4 构建优化
  build: {
    // 启用 analyze 模式
    analyze: import.meta.env.ANALYZE === "true",
  },

  // Nuxt v4 Vite 配置
  vite: {
    // 构建配置（消除 Tailwind CSS sourcemap 警告）
    build: {
      // 生产环境禁用 sourcemap，减小构建体积并消除警告
      sourcemap: import.meta.env.NODE_ENV !== "production",
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
    },
  },

  runtimeConfig: {
    public: {
      // 高德 JS API：dev模式用securitycode，prod模式用servicehost
      amapKey:
        import.meta.env.NUXT_PUBLIC_AMAP_KEY ||
        "d6e827b4530d8a87f745f3f1f237cb1d",
      amapSecurityCode: import.meta.env.NUXT_PUBLIC_AMAP_SECURITY_CODE || "b18c8efd4340355a7546e65bba339cab",
      // 可选：手动指定代理域名；不填时默认使用当前站点 origin + /_AMapService
      amapServiceHost:
        import.meta.env.NUXT_PUBLIC_AMAP_SERVICE_HOST || "/_AMapService",
      wechatAppId: import.meta.env.NUXT_PUBLIC_WECHAT_APPID || "",
      shareDefaultImage: import.meta.env.NUXT_PUBLIC_SHARE_DEFAULT_IMAGE || "",
      siteUrl: import.meta.env.NUXT_PUBLIC_SITE_URL || "",
    },
  },
});

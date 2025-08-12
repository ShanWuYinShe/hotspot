// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  devtools: { enabled: true },
  compatibilityDate: '2025-05-28',
  future: {
    compatibilityVersion: 4,  // 启用 Nuxt 4 特性
  },

  modules: [
    '@pinia/nuxt',
    'pinia-plugin-persistedstate/nuxt',
    '@nuxt/ui'
  ],

  // Pinia 持久化配置  
  piniaPluginPersistedstate: {
    storage: 'localStorage',
    debug: true
  },

  // SSR 配置
  ssr: true,

  // Nuxt UI 配置
  ui: {
    fonts: false
  },

  // 样式配置
  css: [
    '~/assets/styles/main.css'
  ],

  // TypeScript 配置
  typescript: {
    strict: true,
    typeCheck: true
  },

  // 实验性特性
  experimental: {
    typedPages: true,
    appManifest: true
  },
  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL || 'http://localhost:8080/api/v1',
      wsBaseUrl: process.env.NUXT_PUBLIC_WS_BASE_URL || 'ws://localhost:8080/ws/',
      amapKey: process.env.NUXT_PUBLIC_AMAP_KEY || '',
      amapServiceSecurityCode: process.env.NUXT_PUBLIC_AMAP_SERVICE_SECURITY_CODE || '',
      amapServiceHost: process.env.NUXT_PUBLIC_AMAP_SERVICE_HOST || '',
      wechatAppId: process.env.NUXT_PUBLIC_WECHAT_APPID || '',
      shareDefaultImage: process.env.NUXT_PUBLIC_SHARE_DEFAULT_IMAGE || ''
    }
  }
})

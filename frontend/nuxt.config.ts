// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  devtools: { enabled: true },
  compatibilityDate: '2025-05-28',

  modules: [
    '@pinia/nuxt',
    'pinia-plugin-persistedstate/nuxt',
    '@nuxt/icon'
  ],

  // Pinia 持久化配置
  piniaPluginPersistedstate: {
    storage: 'localStorage',
    debug: true
  },
  // SSR 配置
  ssr: true,
  // 禁用特定组件的 SSR
  build: {
    transpile: ['naive-ui', 'vueuc', '@css-render/vue3-ssr', '@juggle/resize-observer']
  },

  // Naive UI 配置
  css: [
    '~/assets/styles/index.scss'
  ],
  runtimeConfig: {
    public: {
      uiLibrary: 'naive-ui',
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

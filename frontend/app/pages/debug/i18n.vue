<template>
  <div class="p-8">
    <h1>i18n Debug</h1>
    
    <div class="mt-4">
      <h2>Current Locale: {{ currentLocale }}</h2>
      <h2>Available Locales: {{ availableLocales }}</h2>
      <h2>Nuxt App $i18n exists: {{ hasNuxtI18n }}</h2>
    </div>

    <div class="mt-4">
      <h3>Messages for current locale:</h3>
      <pre>{{ JSON.stringify(currentMessages, null, 2) }}</pre>
    </div>

    <div class="mt-4">
      <UButton @click="switchToZhCN">切换到中文</UButton>
      <UButton @click="switchToEnUS" class="ml-2">Switch to English</UButton>
      <UButton @click="forceReload" class="ml-2">强制重新加载</UButton>
    </div>

    <div class="mt-4">
      <h3>Test translations:</h3>
      <p>app.title: {{ $t('app.title') }}</p>
      <p>nav.home: {{ $t('nav.home') }}</p>
      <p>nav.settings: {{ $t('nav.settings') }}</p>
    </div>

    <div class="mt-4">
      <h3>Debug Log:</h3>
      <pre>{{ debugLog }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
const { locale, availableLocales, messages } = useI18n()
const nuxtApp = useNuxtApp()

const debugLog = ref<string[]>([])
const currentLocale = ref(locale.value)
const currentMessages = ref(messages.value)
const hasNuxtI18n = ref(!!nuxtApp.$i18n)

function addLog(message: string) {
  debugLog.value.push(`[${new Date().toLocaleTimeString()}] ${message}`)
  console.log(message)
}

// 初始调试信息
addLog(`Initial locale: ${locale.value}`)
addLog(`Available locales: ${JSON.stringify(availableLocales.value)}`)
addLog(`Has $i18n: ${!!nuxtApp.$i18n}`)
addLog(`Messages loaded: ${Object.keys(messages.value).length > 0}`)

const switchToZhCN = async () => {
  addLog('Attempting to switch to zh-CN...')
  const { switchLocale } = useLocaleSwitcher()
  const success = await switchLocale('zh-CN')
  
  if (success) {
    addLog('SUCCESS: Language switched to zh-CN')
    currentLocale.value = locale.value
    currentMessages.value = messages.value
    addLog(`Locale now: ${locale.value}`)
    addLog(`Messages loaded: ${Object.keys(messages.value).length}`)
  } else {
    addLog('ERROR: Failed to switch language')
  }
}

const switchToEnUS = async () => {
  addLog('Attempting to switch to en-US...')
  const { switchLocale } = useLocaleSwitcher()
  const success = await switchLocale('en-US')
  
  if (success) {
    addLog('SUCCESS: Language switched to en-US')
    currentLocale.value = locale.value
    currentMessages.value = messages.value
    addLog(`Locale now: ${locale.value}`)
    addLog(`Messages loaded: ${Object.keys(messages.value).length}`)
  } else {
    addLog('ERROR: Failed to switch language')
  }
}

const forceReload = () => {
  window.location.reload()
}

// 监听 locale 变化
watch(locale, (newLocale) => {
  addLog(`Locale changed to: ${newLocale}`)
  currentLocale.value = newLocale
})

watch(messages, (newMessages) => {
  addLog(`Messages updated, keys: ${Object.keys(newMessages).length}`)
  currentMessages.value = newMessages
})

onMounted(async () => {
  addLog('Component mounted')
  // 尝试强制设置为 zh-CN
  await switchToZhCN()
})
</script>

<template>
  <div class="p-8">
    <h1 class="text-2xl font-bold mb-6">Locale 切换方法对比</h1>
    
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <!-- 方法1：直接设置 locale.value -->
      <UCard>
        <template #header>
          <h2 class="text-lg font-semibold">方法1: locale.value 直接赋值</h2>
        </template>
        
        <div class="space-y-4">
          <UButton @click="switchByDirectAssignment" color="red" variant="outline">
            切换到中文 (locale.value)
          </UButton>
          
          <div class="text-sm">
            <p><strong>当前 locale:</strong> {{ locale }}</p>
            <p><strong>翻译结果:</strong> {{ $t('nav.home') }}</p>
            <p><strong>Messages 数量:</strong> {{ Object.keys(messages).length }}</p>
          </div>
        </div>
      </UCard>

      <!-- 方法2：使用 $i18n.setLocale -->
      <UCard>
        <template #header>
          <h2 class="text-lg font-semibold">方法2: $i18n.setLocale</h2>
        </template>
        
        <div class="space-y-4">
          <UButton @click="switchBySetLocale" color="green">
            切换到中文 ($i18n.setLocale)
          </UButton>
          
          <div class="text-sm">
            <p><strong>当前 locale:</strong> {{ locale }}</p>
            <p><strong>翻译结果:</strong> {{ $t('nav.home') }}</p>
            <p><strong>Messages 数量:</strong> {{ Object.keys(messages).length }}</p>
          </div>
        </div>
      </UCard>
    </div>

    <div class="mt-8">
      <h3 class="text-lg font-semibold mb-4">实时日志</h3>
      <UCard>
        <pre class="text-sm">{{ logs.join('\n') }}</pre>
      </UCard>
    </div>

    <div class="mt-6">
      <UButton @click="resetToEnglish" variant="outline">重置为英文</UButton>
      <UButton @click="clearLogs" variant="ghost" class="ml-2">清空日志</UButton>
    </div>
  </div>
</template>

<script setup lang="ts">
const { locale, messages } = useI18n()
const nuxtApp = useNuxtApp()
const logs = ref<string[]>([])

function addLog(message: string) {
  const timestamp = new Date().toLocaleTimeString()
  logs.value.push(`[${timestamp}] ${message}`)
}

// 方法1：直接设置 locale.value
const switchByDirectAssignment = () => {
  addLog('=== 使用 locale.value 直接赋值 ===')
  addLog(`切换前 - locale: ${locale.value}, messages: ${Object.keys(messages.value).length}`)
  
  locale.value = 'zh-CN'
  
  addLog(`切换后 - locale: ${locale.value}, messages: ${Object.keys(messages.value).length}`)
  addLog(`翻译结果: ${$t('nav.home')}`)
  addLog('注意：locale 改变了，但 messages 没有加载，翻译失败')
}

// 方法2：使用 $i18n.setLocale
const switchBySetLocale = async () => {
  addLog('=== 使用 $i18n.setLocale ===')
  addLog(`切换前 - locale: ${locale.value}, messages: ${Object.keys(messages.value).length}`)
  
  try {
    if (nuxtApp.$i18n?.setLocale) {
      await nuxtApp.$i18n.setLocale('zh-CN')
      addLog(`切换后 - locale: ${locale.value}, messages: ${Object.keys(messages.value).length}`)
      addLog(`翻译结果: ${$t('nav.home')}`)
      addLog('成功：locale 改变，messages 已加载，翻译成功')
    } else {
      addLog('错误：$i18n.setLocale 不可用')
    }
  } catch (error) {
    addLog(`错误：${error}`)
  }
}

const resetToEnglish = async () => {
  addLog('=== 重置为英文 ===')
  if (nuxtApp.$i18n?.setLocale) {
    await nuxtApp.$i18n.setLocale('en-US')
    addLog('已重置为英文')
  }
}

const clearLogs = () => {
  logs.value = []
}

// 初始日志
onMounted(() => {
  addLog('页面加载完成')
  addLog(`初始状态 - locale: ${locale.value}, messages: ${Object.keys(messages.value).length}`)
})
</script>

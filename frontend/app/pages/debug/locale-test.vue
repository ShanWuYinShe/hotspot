<template>
  <div class="max-w-2xl mx-auto p-6 space-y-6">
    <div class="text-center">
      <h1 class="text-2xl font-bold mb-4">语言切换测试</h1>
      <p class="text-neutral-600 dark:text-neutral-400">
        测试 Nuxt i18n 语言切换功能
      </p>
    </div>

    <!-- 当前语言信息 -->
    <UCard>
      <template #header>
        <h2 class="text-lg font-semibold">当前语言信息</h2>
      </template>
      
      <div class="space-y-3">
        <div class="flex justify-between">
          <span class="font-medium">当前语言:</span>
          <UBadge>{{ currentLocale }}</UBadge>
        </div>
        
        <div class="flex justify-between">
          <span class="font-medium">语言显示名:</span>
          <span>{{ locales.find(l => l.code === currentLocale)?.name || currentLocale }}</span>
        </div>
        
        <div class="flex justify-between">
          <span class="font-medium">可用语言数量:</span>
          <span>{{ locales.length }}</span>
        </div>
      </div>
    </UCard>

    <!-- 语言切换控件 -->
    <UCard>
      <template #header>
        <h2 class="text-lg font-semibold">语言切换</h2>
      </template>
      
      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium mb-2">选择语言:</label>
          <USelectMenu 
            v-model="selectedLocale" 
            :options="localeOptions"
            placeholder="选择语言"
            class="w-full"
          />
        </div>
        
        <div class="flex gap-2">
          <UButton 
            @click="switchToZhCN" 
            :color="currentLocale === 'zh-CN' ? 'primary' : 'gray'"
          >
            切换到中文
          </UButton>
          <UButton 
            @click="switchToEnUS" 
            :color="currentLocale === 'en-US' ? 'primary' : 'gray'"
          >
            Switch to English
          </UButton>
        </div>
      </div>
    </UCard>

    <!-- 翻译测试 -->
    <UCard>
      <template #header>
        <h2 class="text-lg font-semibold">翻译测试</h2>
      </template>
      
      <div class="space-y-3">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <span class="text-sm text-neutral-500">nav.home:</span>
            <p class="font-medium">{{ $t('nav.home') }}</p>
          </div>
          <div>
            <span class="text-sm text-neutral-500">nav.messages:</span>
            <p class="font-medium">{{ $t('nav.messages') }}</p>
          </div>
          <div>
            <span class="text-sm text-neutral-500">nav.profile:</span>
            <p class="font-medium">{{ $t('nav.profile') }}</p>
          </div>
          <div>
            <span class="text-sm text-neutral-500">nav.settings:</span>
            <p class="font-medium">{{ $t('nav.settings') }}</p>
          </div>
        </div>
      </div>
    </UCard>

    <!-- 调试信息 -->
    <UCard>
      <template #header>
        <h2 class="text-lg font-semibold">调试信息</h2>
      </template>
      
      <div class="space-y-3">
        <div>
          <h4 class="font-medium mb-2">useLocaleSwitcher 信息:</h4>
          <pre class="text-xs bg-neutral-100 dark:bg-neutral-800 p-3 rounded overflow-auto">{{ JSON.stringify({ currentLocale, locales }, null, 2) }}</pre>
        </div>
        
        <div>
          <h4 class="font-medium mb-2">useI18n 原始信息:</h4>
          <pre class="text-xs bg-neutral-100 dark:bg-neutral-800 p-3 rounded overflow-auto">{{ JSON.stringify(i18nDebugInfo, null, 2) }}</pre>
        </div>
      </div>
    </UCard>

    <!-- 操作日志 -->
    <UCard v-if="logs.length > 0">
      <template #header>
        <div class="flex justify-between items-center">
          <h2 class="text-lg font-semibold">操作日志</h2>
          <UButton size="sm" variant="ghost" @click="clearLogs">
            清空日志
          </UButton>
        </div>
      </template>
      
      <div class="space-y-2 max-h-40 overflow-y-auto">
        <div 
          v-for="(log, index) in logs" 
          :key="index"
          class="text-sm p-2 bg-neutral-50 dark:bg-neutral-800 rounded"
        >
          <span class="text-neutral-500">{{ log.time }}</span>
          <span class="ml-2" :class="log.success ? 'text-green-600' : 'text-red-600'">
            {{ log.message }}
          </span>
        </div>
      </div>
    </UCard>
  </div>
</template>

<script setup lang="ts">
// 页面元数据
definePageMeta({
  title: '语言切换测试'
})

// 使用 i18n 和 locale switcher
const { locale, locales: availableLocales } = useI18n()
const { 
  currentLocale, 
  locales, 
  switchLocale
} = useLocaleSwitcher()

// 操作日志
const logs = ref<Array<{ time: string, message: string, success: boolean }>>([])

// 添加日志
const addLog = (message: string, success: boolean = true) => {
  logs.value.unshift({
    time: new Date().toLocaleTimeString(),
    message,
    success
  })
  // 限制日志数量
  if (logs.value.length > 10) {
    logs.value = logs.value.slice(0, 10)
  }
}

// 清空日志
const clearLogs = () => {
  logs.value = []
}

// 选中的语言
const selectedLocale = computed({
  get: () => currentLocale.value,
  set: async (value) => {
    const success = await switchLocale(value)
    addLog(`切换语言到 ${value}: ${success ? '成功' : '失败'}`, success)
  }
})

// 语言选项
const localeOptions = computed(() => {
  return locales.value.map((locale: any) => ({
    label: locale.name || locale.code,
    value: locale.code
  }))
})

// 切换到中文
const switchToZhCN = async () => {
  const success = await switchLocale('zh-CN')
  addLog(`切换到中文: ${success ? '成功' : '失败'}`, success)
}

// 切换到英文
const switchToEnUS = async () => {
  const success = await switchLocale('en-US')
  addLog(`切换到英文: ${success ? '成功' : '失败'}`, success)
}

// i18n 调试信息
const i18nDebugInfo = computed(() => ({
  locale: locale.value,
  availableLocales: availableLocales.value,
  localesType: typeof availableLocales.value,
  localesLength: availableLocales.value?.length
}))

// 页面加载时添加初始日志
onMounted(() => {
  addLog(`页面加载，当前语言: ${currentLocale.value}`)
})
</script>

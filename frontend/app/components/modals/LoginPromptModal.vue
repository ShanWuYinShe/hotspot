<template>
  <UModal
    title="需要登录"
    :description="message"
    :ui="{ width: 'sm:max-w-md' }"
  >
    <template #body="{ close }">
      <section class="flex justify-center mb-4">
        <UAvatar
          icon="i-heroicons-lock-closed"
          size="xl"
          :ui="{ icon: 'text-secondary-content' }"
        />
      </section>

      <p class="text-sm text-secondary-content text-center mb-4">
        登录后即可解锁所有功能
      </p>

      <UCard :ui="{ body: 'p-4' }">
        <ul class="space-y-2 text-sm text-secondary-content">
          <li class="flex items-center gap-2">
            <UIcon
              name="i-heroicons-check-circle"
              class="w-4 h-4 text-success"
            />
            发布和分享生活瞬间
          </li>
          <li class="flex items-center gap-2">
            <UIcon
              name="i-heroicons-check-circle"
              class="w-4 h-4 text-success"
            />
            与好友互动和聊天
          </li>
          <li class="flex items-center gap-2">
            <UIcon
              name="i-heroicons-check-circle"
              class="w-4 h-4 text-success"
            />
            加入兴趣群组
          </li>
          <li class="flex items-center gap-2">
            <UIcon
              name="i-heroicons-check-circle"
              class="w-4 h-4 text-success"
            />
            发现附近的人和地点
          </li>
        </ul>
      </UCard>
    </template>

    <template #footer="{ close }">
      <nav class="flex flex-col gap-3 w-full">
        <UButton
          block
          size="lg"
          @click="() => goToLogin(close)"
        >
          立即登录
        </UButton>

        <p class="text-center text-sm text-secondary-content">
          还没有账号？
          <UButton
            variant="link"
            :padded="false"
            @click="() => goToRegister(close)"
          >
            立即注册
          </UButton>
        </p>

        <UButton
          block
          color="neutral"
          variant="ghost"
          @click="close"
        >
          继续浏览
        </UButton>
      </nav>
    </template>
  </UModal>
</template>

<script setup lang="ts">
const props = defineProps<{
  actionName?: string
  redirectPath?: string
}>()

const router = useRouter()

const message = computed(() => {
  if (props.actionName) return `登录后即可${props.actionName}`
  return '登录后即可使用完整功能'
})

const goToLogin = (close: () => void) => {
  close()
  const query: Record<string, string> = { reason: 'action_required' }
  if (props.redirectPath) query.redirect = props.redirectPath
  if (props.actionName) query.action = props.actionName
  router.push({ path: '/auth/login', query })
}

const goToRegister = (close: () => void) => {
  close()
  const query: Record<string, string> = {}
  if (props.redirectPath) query.redirect = props.redirectPath
  router.push({ path: '/auth/register', query })
}
</script>

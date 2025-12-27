<template>
  <UModal
    title="需要登录"
    :description="message"
  >
    <template #body="{ close }">
      <section class="flex justify-center mb-4">
        <UAvatar
          icon="i-lucide-lock"
          size="xl"
        />
      </section>

      <UAlert
        class="mb-4"
        color="neutral"
        variant="soft"
        icon="i-lucide-lock"
        title="登录后即可解锁所有功能"
      />

      <UCard variant="soft">
        <ul class="space-y-2">
          <li class="flex items-center gap-2">
            <UBadge color="success" variant="soft" size="xs" square icon="i-lucide-circle-check" />
            发布和分享生活瞬间
          </li>
          <li class="flex items-center gap-2">
            <UBadge color="success" variant="soft" size="xs" square icon="i-lucide-circle-check" />
            与好友互动和聊天
          </li>
          <li class="flex items-center gap-2">
            <UBadge color="success" variant="soft" size="xs" square icon="i-lucide-circle-check" />
            加入兴趣群组
          </li>
          <li class="flex items-center gap-2">
            <UBadge color="success" variant="soft" size="xs" square icon="i-lucide-circle-check" />
            发现附近的人和地点
          </li>
        </ul>
      </UCard>
    </template>

    <template #footer="{ close }">
      <nav class="flex flex-col gap-3 w-full">
        <UButton
          :block="true"
          @click="() => goToLogin(close)"
        >
          立即登录
        </UButton>

        <p class="text-center text-xs">
          还没有账号？
          <UButton
            variant="link"
            @click="() => goToRegister(close)"
          >
            立即注册
          </UButton>
        </p>

        <UButton
          :block="true"
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
  router.push({ path: '/auth', query: { ...query, mode: 'login' } })
}

const goToRegister = (close: () => void) => {
  close()
  const query: Record<string, string> = {}
  if (props.redirectPath) query.redirect = props.redirectPath
  router.push({ path: '/auth', query: { ...query, mode: 'register' } })
}
</script>

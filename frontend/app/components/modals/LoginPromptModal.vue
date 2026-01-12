<template>
    <UModal title="需要登录" :description="message">
    <template #body="{ close }">
      <div class="p-6">
          <div class="flex justify-center mb-6">
            <UAvatar
              icon="i-lucide-lock"
              size="3xl"
                    color="neutral"
                    variant="soft"
            />
          </div>

      <UAlert
        class="mb-6"
        color="neutral"
        variant="subtle"
        icon="i-lucide-shield"
        title="登录后即可解锁所有功能"
      />

      <UCard variant="soft" class="mb-2">
        <ul class="space-y-3">
          <li class="flex items-center gap-3 text-sm">
                        <UIcon
                            name="i-lucide-check-circle-2"
                            class="shrink-0"
                            size="18"
                            color="success"
                        />
            <span>发布和分享生活瞬间</span>
          </li>
          <li class="flex items-center gap-3 text-sm">
                        <UIcon
                            name="i-lucide-check-circle-2"
                            class="shrink-0"
                            size="18"
                            color="success"
                        />
            <span>与好友互动和聊天</span>
          </li>
          <li class="flex items-center gap-3 text-sm">
                        <UIcon
                            name="i-lucide-check-circle-2"
                            class="shrink-0"
                            size="18"
                            color="success"
                        />
            <span>加入兴趣群组</span>
          </li>
          <li class="flex items-center gap-3 text-sm">
                        <UIcon
                            name="i-lucide-check-circle-2"
                            class="shrink-0"
                            size="18"
                            color="success"
                        />
            <span>发现附近的人和地点</span>
          </li>
        </ul>
      </UCard>
      </div>
    </template>

    <template #footer="{ close }">
      <div class="flex flex-col gap-3 w-full">
        <UButton
          block
          size="lg"
          @click="() => goToLogin(close)"
        >
          立即登录
        </UButton>

                <div
                    class="flex items-center justify-center gap-2 text-sm text-neutral-500 dark:text-neutral-400"
                >
          <span>还没有账号？</span>
          <UButton
            variant="link"
            color="primary"
            @click="() => goToRegister(close)"
          >
            立即注册
          </UButton>
        </div>

        <USeparator label="或" />

        <UButton
          block
          color="neutral"
          variant="ghost"
          @click="close"
        >
          继续浏览
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
const props = defineProps<{
    actionName?: string;
    redirectPath?: string;
}>();

const router = useRouter();

const message = computed(() => {
    if (props.actionName) return `登录后即可${props.actionName}`;
    return "登录后即可使用完整功能";
});

const goToLogin = (close: () => void) => {
    close();
    const query: Record<string, string> = { reason: "action_required" };
    if (props.redirectPath) query.redirect = props.redirectPath;
    if (props.actionName) query.action = props.actionName;
    router.push({ path: "/auth", query: { ...query, mode: "login" } });
};

const goToRegister = (close: () => void) => {
    close();
    const query: Record<string, string> = {};
    if (props.redirectPath) query.redirect = props.redirectPath;
    router.push({ path: "/auth", query: { ...query, mode: "register" } });
};
</script>

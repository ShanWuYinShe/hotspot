<template>
  <UModal
    title="输入群组密码"
    :description="groupName ? `加入群组「${groupName}」需要输入密码` : '该群组需要密码才能加入'"
    :close="{ onClick: () => emit('close', null) }"
    @update:open="(open) => { if (!open) emit('close', null) }"
  >
    <template #body>
      <UFormField label="群组密码" name="password">
        <UInput
          v-model="password"
          type="password"
          placeholder="请输入群组密码"
          icon="i-lucide-lock"
          :autofocus="true"
          @keyup.enter="confirm"
        />
      </UFormField>
    </template>

    <template #footer>
      <div class="flex justify-end gap-3">
        <UButton 
          variant="outline" 
          color="neutral"
          @click="emit('close', null)"
        >
          取消
        </UButton>
        <UButton 
          :disabled="!password"
          @click="confirm"
        >
          确认加入
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
defineProps<{ groupName?: string }>()

const emit = defineEmits<{ close: [value: string | null] }>()

const password = ref('')

const confirm = () => {
  if (!password.value) return
  emit('close', password.value)
}
</script>

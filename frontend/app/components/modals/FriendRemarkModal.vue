<template>
  <UModal title="设置备注" :close="{ onClick: () => $emit('close') }">
    <template #body>
      <div class="space-y-4">
        <UFormField label="好友昵称">
          <p class="text-sm text-neutral-600 dark:text-neutral-400">
            {{ friend?.nickname }}
          </p>
        </UFormField>

        <UFormField label="备注名称">
          <UInput
            v-model="remarkText"
            placeholder="输入备注名称（留空则显示昵称）"
            :maxlength="20"
          />
          <div class="text-right text-xs text-neutral-500 mt-1">
            {{ remarkText.length }}/20
          </div>
        </UFormField>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton variant="outline" @click="$emit('cancel')">
          取消
        </UButton>
        <UButton @click="handleSave" :loading="saving">
          保存
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import type { UserResponse } from "~/types/api";

interface Props {
  friend?: UserResponse | null
  onSave?: (remarkText: string) => Promise<void>
}

const props = defineProps<Props>()
const emit = defineEmits<{
  cancel: []
  success: []
  close: []
}>()

// useOverlay 会自动管理 Modal 状态，无需手动绑定 isOpen

const remarkText = ref('')
const saving = ref(false)

// 监听 friend 变化，更新备注文本
watch(() => props.friend, (friend) => {
  if (friend) {
    remarkText.value = (friend as UserResponse & { remark?: string }).remark || ''
  }
}, { immediate: true })

const handleSave = async () => {
  if (!props.friend || saving.value) return

  saving.value = true
  try {
    if (props.onSave) {
      await props.onSave(remarkText.value)
    }
    emit('success')
  } catch (error) {
    // 错误处理由父组件负责
    throw error
  } finally {
    saving.value = false
  }
}
</script>


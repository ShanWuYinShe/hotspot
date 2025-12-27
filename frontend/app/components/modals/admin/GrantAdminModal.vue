<template>
  <UModal
    :title="title"
    :dismissible="false"
  >
    <template #body>
      <UForm :state="form" @submit="submit" class="space-y-4">
        <UFormField label="用户 ID" required>
          <UInput v-model.number="form.user_id" type="number" :disabled="isEditing" />
        </UFormField>

        <UFormField v-if="canSetSuper" label="超级管理员">
          <UCheckbox v-model="form.is_super" label="授予超级管理员（拥有全部权限）" />
        </UFormField>

        <UFormField label="角色">
          <USelectMenu
            v-model="form.role_ids"
            multiple
            :options="roleOptions"
            value-key="value"
            option-attribute="label"
            placeholder="选择角色（可多选）"
          />
        </UFormField>
      </UForm>
    </template>

    <template #footer>
      <UButton
        variant="outline"
        color="neutral"
        label="取消"
        @click="emit('close', null)"
      />
      <UButton
        :label="isEditing ? '保存' : '授权'"
        @click="submit()"
      />
    </template>
  </UModal>
</template>

<script setup lang="ts">
interface SelectOption {
  label: string
  value: string
}

export interface GrantAdminPayload {
  user_id: number
  is_super: boolean
  role_ids: string[]
}

interface Props {
  title: string
  isEditing: boolean
  canSetSuper: boolean
  roleOptions: SelectOption[]
  initial?: GrantAdminPayload
}

const props = withDefaults(defineProps<Props>(), {
  initial: undefined
})

const emit = defineEmits<{
  close: [value: GrantAdminPayload | null]
}>()

const form = reactive<GrantAdminPayload>({
  user_id: props.initial?.user_id ?? 0,
  is_super: props.initial?.is_super ?? false,
  role_ids: [...(props.initial?.role_ids ?? [])]
})

const submit = () => {
  emit('close', {
    user_id: Number(form.user_id) || 0,
    is_super: !!form.is_super,
    role_ids: [...(form.role_ids || [])]
  })
}
</script>


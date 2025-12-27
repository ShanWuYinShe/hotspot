<template>
  <UModal
    :title="title"
    :dismissible="false"
  >
    <template #body>
      <UForm :state="form" @submit="submit" class="space-y-4">
        <UFormField label="角色名称" required>
          <UInput v-model="form.name" />
        </UFormField>

        <UFormField label="描述">
          <UTextarea v-model="form.description" />
        </UFormField>

        <UFormField label="权限">
          <USelectMenu
            v-model="form.permission_keys"
            multiple
            :options="permissionOptions"
            value-key="value"
            option-attribute="label"
            placeholder="选择权限（可多选）"
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
        :label="isEditing ? '保存' : '创建'"
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

export interface RolePayload {
  id?: string
  name: string
  description?: string
  permission_keys: string[]
}

interface Props {
  title: string
  isEditing: boolean
  permissionOptions: SelectOption[]
  initial?: RolePayload
}

const props = withDefaults(defineProps<Props>(), {
  initial: undefined
})

const emit = defineEmits<{
  close: [value: RolePayload | null]
}>()

const form = reactive<RolePayload>({
  id: props.initial?.id,
  name: props.initial?.name ?? '',
  description: props.initial?.description ?? '',
  permission_keys: [...(props.initial?.permission_keys ?? [])]
})

const submit = () => {
  const payload: RolePayload = {
    id: form.id,
    name: String(form.name || '').trim(),
    description: String(form.description || '').trim() || undefined,
    permission_keys: [...(form.permission_keys || [])]
  }
  emit('close', payload)
}
</script>


<template>
  <!-- 独立模态框模式 -->
  <UModal v-if="!embedded" v-model:open="open" title="创建群组" description="填写群组信息并创建">
    <UTooltip text="创建群组">
      <slot>
        <UButton>
          <UIcon name="i-heroicons-plus" />
          创建群组
        </UButton>
      </slot>
    </UTooltip>

    <template #body>
      <CreateGroupForm @success="handleSuccess" @cancel="close" />
    </template>
  </UModal>
  
  <!-- 嵌入模式 - 直接渲染表单 -->
  <CreateGroupForm v-else @success="handleSuccess" @cancel="$emit('cancel')" />
</template>

<script setup lang="ts">
// Props
interface Props {
  embedded?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  embedded: false
})

// Emits
const emit = defineEmits<{
  success: [group: any];
  cancel: [];
}>()

// 弹窗开关（支持外部控制）
const open = defineModel<boolean>('open', { default: false })

// 关闭弹窗
function close() {
  open.value = false
}

// 处理创建成功
function handleSuccess(group: any) {
  emit('success', group)
  if (!props.embedded) {
    close()
  }
}
</script>
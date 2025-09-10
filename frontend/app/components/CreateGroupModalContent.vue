<template>
  <UModal title="创建群组" description="填写群组信息并创建">
    <template #body>
      <CreateGroupForm @success="handleSuccess" @cancel="handleCancel" />
    </template>
  </UModal>
</template>

<script setup lang="ts">
// Props
interface Props {
  open?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  open: false
});

// Emits
const emit = defineEmits<{
  'update:open': [value: boolean];
  'success': [group: any];
  'cancel': [];
  'close': [];
}>();

// useOverlay 会自动管理 Modal 状态，无需手动绑定 isOpen

// 处理函数
function handleSuccess(group: any) {
  emit('success', group);
  emit('close');
}

function handleCancel() {
  emit('cancel');
  emit('close');
}
</script>


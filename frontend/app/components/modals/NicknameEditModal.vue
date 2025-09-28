<template>
  <UModal title="编辑昵称" :close="{ onClick: () => $emit('close') }">
    <template #body>
      <UForm :state="nicknameForm" :validate="validateNickname" @submit="onSaveNickname" class="space-y-4">
        <UFormField label="昵称" name="nickname" required>
          <UInput v-model="nicknameForm.nickname" placeholder="请输入新的昵称" :maxlength="20" />
        </UFormField>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" @click="handleCancel">取消</UButton>
          <UButton type="submit" :loading="savingNickname">保存</UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>

<script setup lang="ts">
// Props
interface Props {
  open?: boolean;
  currentNickname?: string;
}

const props = withDefaults(defineProps<Props>(), {
  open: false,
  currentNickname: ''
});

// Emits
const emit = defineEmits<{
  'update:open': [value: boolean];
  'success': [nickname: string];
  'cancel': [];
  'close': [];
}>();

// useOverlay 会自动管理 Modal 状态，无需手动绑定 isOpen

const savingNickname = ref(false);

// 表单状态
const nicknameForm = reactive({
  nickname: ''
});

// 监听 props 变化，更新表单
watch(() => props.currentNickname, (newValue) => {
  nicknameForm.nickname = newValue || '';
}, { immediate: true });

// 表单验证
const validateNickname = (state: any) => {
  const errors = [];
  if (!state.nickname || state.nickname.trim().length === 0) {
    errors.push({ path: 'nickname', message: '昵称不能为空' });
  } else if (state.nickname.trim().length > 20) {
    errors.push({ path: 'nickname', message: '昵称不能超过20个字符' });
  }
  return errors;
};

// 保存昵称
const onSaveNickname = async () => {
  if (savingNickname.value) return;
  
  savingNickname.value = true;
  try {
    emit('success', nicknameForm.nickname.trim());
    emit('close');
  } catch (error) {
    console.error('保存昵称失败:', error);
  } finally {
    savingNickname.value = false;
  }
};

// 取消编辑
const handleCancel = () => {
  emit('cancel');
  emit('close');
  // 重置表单
  nicknameForm.nickname = props.currentNickname || '';
};
</script>


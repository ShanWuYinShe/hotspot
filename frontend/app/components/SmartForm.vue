<template>
  <UForm 
    ref="formRef"
    :schema="schema" 
    :state="state" 
    class="space-y-4"
    @submit="handleSubmit"
    @error="handleError"
  >
    <slot :state="state" :errors="errors" :loading="loading" />
    
    <!-- 表单操作按钮 -->
    <div v-if="showActions" class="flex flex-col sm:flex-row gap-3 pt-4">
      <UButton 
        type="submit"
        :loading="loading"
        :disabled="!isValid || loading"
        :color="submitColor"
        :variant="submitVariant"
        :size="submitSize"
        class="transition-colors duration-200"
      >
        <template v-if="submitIcon" #leading>
          <UIcon :name="submitIcon" class="w-4 h-4" />
        </template>
        {{ submitText }}
      </UButton>
      
      <UButton 
        v-if="showCancel"
        type="button"
        variant="outline"
        color="neutral"
        :size="submitSize"
        :disabled="loading"
        @click="$emit('cancel')"
        class="transition-colors duration-200"
      >
        <template #leading>
          <UIcon name="i-heroicons-x-mark" class="w-4 h-4" />
        </template>
        {{ cancelText }}
      </UButton>
      
      <UButton 
        v-if="showReset"
        type="button"
        variant="ghost"
        color="neutral"
        :size="submitSize"
        :disabled="loading"
        @click="resetForm"
        class="transition-colors duration-200"
      >
        <template #leading>
          <UIcon name="i-heroicons-arrow-path" class="w-4 h-4" />
        </template>
        {{ resetText }}
      </UButton>
    </div>
    
    <!-- 表单错误提示 -->
    <UAlert 
      v-if="formError"
      color="error"
      variant="soft"
      :title="formErrorTitle"
      :description="formError"
      :close-button="{ icon: 'i-heroicons-x-mark-20-solid', color: 'error', variant: 'link' }"
      @close="clearFormError"
    />
    
    <!-- 表单成功提示 -->
    <UAlert 
      v-if="formSuccess"
      color="success"
      variant="soft"
      :title="formSuccessTitle"
      :description="formSuccess"
      :close-button="{ icon: 'i-heroicons-x-mark-20-solid', color: 'green', variant: 'link' }"
      @close="clearFormSuccess"
    />
  </UForm>
</template>

<script setup lang="ts">
// 使用Nuxt UI v3的类型定义
type FormError = { path: string; message: string }
type FormSubmitEvent = Event & { data: any }

interface Props {
  schema: any;
  initialState?: Record<string, any>;
  showActions?: boolean;
  showCancel?: boolean;
  showReset?: boolean;
  submitText?: string;
  submitIcon?: string;
  submitColor?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral';
  submitVariant?: 'solid' | 'outline' | 'soft' | 'ghost' | 'link';
  submitSize?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  cancelText?: string;
  resetText?: string;
  formErrorTitle?: string;
  formSuccessTitle?: string;
  validateOnChange?: boolean;
  resetOnSubmit?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  showActions: true,
  showCancel: false,
  showReset: false,
  submitText: '提交',
  submitColor: 'primary',
  submitVariant: 'solid',
  submitSize: 'md',
  cancelText: '取消',
  resetText: '重置',
  formErrorTitle: '表单错误',
  formSuccessTitle: '操作成功',
  validateOnChange: true,
  resetOnSubmit: false
});

const emit = defineEmits<{
  submit: [data: any];
  cancel: [];
  error: [error: FormError[]];
  success: [data: any];
}>();

// 表单状态
const formRef = ref();
const state = ref({ ...props.initialState });
const errors = ref<FormError[]>([]);
const loading = ref(false);
const formError = ref('');
const formSuccess = ref('');

// 计算属性
const isValid = computed(() => {
  if (!props.validateOnChange) return true;
  return errors.value.length === 0 && Object.keys(state.value).length > 0;
});

// 表单提交处理
const handleSubmit = async (event: FormSubmitEvent) => {
  loading.value = true;
  formError.value = '';
  formSuccess.value = '';
  
  try {
    await emit('submit', event.data);
    
    if (props.resetOnSubmit) {
      resetForm();
    }
    
    // 可以通过外部设置成功消息
    // formSuccess.value = '操作成功完成';
  } catch (error: any) {
    formError.value = error?.message || '操作失败，请重试';
    console.error('Form submission error:', error);
  } finally {
    loading.value = false;
  }
};

// 表单错误处理
const handleError = (event: any) => {
  const errorArray = Array.isArray(event) ? event : [event];
  errors.value = errorArray;
  emit('error', errorArray);
};

// 重置表单
const resetForm = () => {
  state.value = { ...props.initialState };
  errors.value = [];
  formError.value = '';
  formSuccess.value = '';
  formRef.value?.clear();
};

// 清除错误消息
const clearFormError = () => {
  formError.value = '';
};

// 清除成功消息
const clearFormSuccess = () => {
  formSuccess.value = '';
};

// 设置表单错误（外部调用）
const setFormError = (error: string) => {
  formError.value = error;
};

// 设置表单成功（外部调用）
const setFormSuccess = (message: string) => {
  formSuccess.value = message;
};

// 设置加载状态（外部调用）
const setLoading = (isLoading: boolean) => {
  loading.value = isLoading;
};

// 验证表单
const validate = async () => {
  return await formRef.value?.validate();
};

// 暴露方法给父组件
defineExpose({
  resetForm,
  setFormError,
  setFormSuccess,
  setLoading,
  validate,
  state: readonly(state),
  loading: readonly(loading),
  isValid: readonly(isValid)
});

// 监听初始状态变化
watch(() => props.initialState, (newState) => {
  if (newState) {
    state.value = { ...newState };
  }
}, { deep: true });
</script>
</template>
</invoke>

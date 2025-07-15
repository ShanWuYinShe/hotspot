<template>
  <div v-if="modelValue" class="pixel-dialog-overlay" @click="closeDialog">
    <div class="pixel-dialog" @click.stop>
      <div class="pixel-dialog__header">
        <h3>⚔️ 创建公会</h3>
        <button @click="closeDialog" class="pixel-btn pixel-btn--close">
          <Icon name="mdi:close" size="16" />
        </button>
      </div>
      
      <div class="pixel-dialog__content">
        <form @submit.prevent="createGroup" class="pixel-form">
          <div class="pixel-form__group">
            <label class="pixel-form__label">🏰 公会名称</label>
            <input 
              ref="nameInputRef"
              v-model="formData.name" 
              type="text" 
              class="pixel-input" 
              placeholder="给你的公会起个霸气的名字" 
              maxlength="50"
              required
            />
          </div>
          
          <div class="pixel-form__group">
            <label class="pixel-form__label">📝 公会描述</label>
            <textarea 
              v-model="formData.description" 
              class="pixel-textarea" 
              placeholder="简单介绍一下公会的目标和玩法"
              maxlength="200"
              rows="3"
            ></textarea>
          </div>
          
          <div class="pixel-form__group">
            <label class="pixel-form__label">🌍 公会位置</label>
            <input 
              v-model="formData.location" 
              type="text" 
              class="pixel-input" 
              placeholder="选择公会的活动区域"
            />
          </div>
          
          <div class="pixel-form__group">
            <label class="pixel-form__label">👥 最大成员数</label>
            <select v-model="formData.maxMembers" class="pixel-select">
              <option value="10">10人 (小队)</option>
              <option value="25">25人 (中队)</option>
              <option value="50">50人 (大队)</option>
              <option value="100">100人 (军团)</option>
            </select>
          </div>
        </form>
      </div>
      
      <div class="pixel-dialog__footer">
        <button @click="closeDialog" class="pixel-btn pixel-btn--secondary">
          <Icon name="mdi:cancel" size="16" />
          取消
        </button>
        <button @click="createGroup" class="pixel-btn pixel-btn--primary" :disabled="!formData.name || isLoading">
          <Icon name="mdi:sword" size="16" />
          {{ isLoading ? '创建中...' : '创建公会' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, nextTick } from 'vue'
import { useMessage } from 'naive-ui'
import { useGroupService } from '~/services/group.service'
import { defaultImages } from '@/utils/assets'

interface Props {
  modelValue: boolean
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'group-created', group: any): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 服务实例
const { createGroup: createGroupApi } = useGroupService()
const message = useMessage()

// 名称输入框引用
const nameInputRef = ref()
const isLoading = ref(false)

// 监听对话框打开，自动聚焦名称输入框
watch(() => props.modelValue, (newValue) => {
  if (newValue) {
    nextTick(() => {
      nameInputRef.value?.focus()
    })
  }
})

// 群组表单数据接口
interface GroupFormData {
  name: string
  description: string
  location: string
  maxMembers: number
  avatar: string
}

const formData = reactive<GroupFormData>({
  name: '',
  description: '',
  location: '',
  maxMembers: 25,
  avatar: defaultImages.groupAvatar
})

const closeDialog = () => {
  emit('update:modelValue', false)
  // 重置表单
  formData.name = ''
  formData.description = ''
  formData.location = ''
  formData.maxMembers = 25
}

const createGroup = async () => {
  if (!formData.name.trim()) {
    message.warning('请输入群组名称')
    return
  }
  
  isLoading.value = true
  
  try {
    // 调用真实的 API 创建群组
    // 注意：后端API需要位置信息，这里使用默认位置或获取用户当前位置
    const result = await createGroupApi({
      name: formData.name.trim(),
      description: formData.description.trim() || undefined,
      latitude: 0, // TODO: 获取用户当前位置
      longitude: 0, // TODO: 获取用户当前位置
      location_name: formData.location.trim() || '未知位置'
    })
    
    message.success('群组创建成功！')
    emit('group-created', result)
    closeDialog()
  } catch (error: any) {
    message.error(error.message || '创建群组失败')
  } finally {
    isLoading.value = false
  }
}
</script>

<style lang="scss" scoped>
.pixel-dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: var(--z-modal);
  backdrop-filter: blur(2px);
}

.pixel-dialog {
  background: #2a2a2a;
  border: 2px solid #4a90e2;
  border-radius: 0;
  max-width: 500px;
  width: 90%;
  max-height: 80vh;
  overflow: hidden;
  box-shadow: 
    0 0 0 2px #000,
    0 0 20px rgba(74, 144, 226, 0.3),
    inset 0 0 0 1px rgba(255, 255, 255, 0.1);
  
  &__header {
    padding: 16px 20px;
    background: linear-gradient(135deg, #3a3a3a, #2a2a2a);
    border-bottom: 2px solid #4a90e2;
    display: flex;
    align-items: center;
    justify-content: space-between;
    
    h3 {
      color: #4a90e2;
      font-family: 'Courier New', monospace;
      font-size: 16px;
      font-weight: bold;
      margin: 0;
      text-shadow: 0 0 8px rgba(74, 144, 226, 0.5);
    }
  }
  
  &__content {
    padding: 20px;
    max-height: 400px;
    overflow-y: auto;
  }
  
  &__footer {
    padding: 16px 20px;
    background: linear-gradient(135deg, #2a2a2a, #1a1a1a);
    border-top: 2px solid #4a90e2;
    display: flex;
    gap: 12px;
    justify-content: flex-end;
  }
}

.pixel-form {
  &__group {
    margin-bottom: 20px;
    
    &:last-child {
      margin-bottom: 0;
    }
  }
  
  &__label {
    display: block;
    color: #e0e0e0;
    font-family: 'Courier New', monospace;
    font-size: 14px;
    font-weight: bold;
    margin-bottom: 8px;
    text-shadow: 0 0 4px rgba(224, 224, 224, 0.3);
  }
}

.pixel-input,
.pixel-textarea,
.pixel-select {
  width: 100%;
  padding: 12px;
  background: #1a1a1a;
  border: 2px solid #444;
  color: #e0e0e0;
  font-family: 'Courier New', monospace;
  font-size: 14px;
  outline: none;
  transition: all 0.3s ease;
  
  &:focus {
    border-color: #4a90e2;
    box-shadow: 
      0 0 0 1px #4a90e2,
      0 0 8px rgba(74, 144, 226, 0.3);
  }
  
  &::placeholder {
    color: #666;
  }
}

.pixel-textarea {
  resize: vertical;
  min-height: 80px;
}

.pixel-select {
  cursor: pointer;
  
  option {
    background: #1a1a1a;
    color: #e0e0e0;
  }
}

.pixel-btn--close {
  background: transparent;
  border: 1px solid #666;
  color: #999;
  padding: 4px;
  
  &:hover {
    background: #ff4757;
    border-color: #ff4757;
    color: white;
  }
}

// 滚动条样式
.pixel-dialog__content::-webkit-scrollbar {
  width: 8px;
}

.pixel-dialog__content::-webkit-scrollbar-track {
  background: #1a1a1a;
  border-radius: 0;
}

.pixel-dialog__content::-webkit-scrollbar-thumb {
  background: #4a90e2;
  border-radius: 0;
  
  &:hover {
    background: #357abd;
  }
}
</style>

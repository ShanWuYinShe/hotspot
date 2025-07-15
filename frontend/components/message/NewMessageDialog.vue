<template>
  <div v-if="modelValue" class="pixel-dialog-overlay" @click="closeDialog">
    <div class="pixel-dialog" @click.stop>
      <div class="pixel-dialog__header">
        <h3 class="text-responsive-lg text-gradient-fire">💌 发送消息</h3>
        <button @click="closeDialog" class="pixel-btn pixel-btn--close">
          <Icon name="mdi:close" size="16" />
        </button>
      </div>
      
      <div class="pixel-dialog__content">
        <form @submit.prevent="sendMessage" class="pixel-form">
          <div class="pixel-form__group">
            <label class="pixel-form__label text-responsive-base text-electric">👤 收信人</label>
            <input 
              ref="recipientInputRef"
              v-model="formData.recipient" 
              type="text" 
              class="pixel-input" 
              placeholder="输入用户名或ID"
              required
            />
            <div class="recipient-hints">
              <div v-for="hint in recipientHints" :key="hint.id" 
                   class="recipient-hint" 
                   @click="selectRecipient(hint)">
                <NAvatar :size="40" :src="hint.avatar" />
                <span class="hint-name text-responsive-sm text-bright">{{ hint.name }}</span>
                <span class="hint-id text-responsive-xs">ID: {{ hint.id }}</span>
              </div>
            </div>
          </div>
          
          <div class="pixel-form__group">
            <label class="pixel-form__label text-responsive-base text-electric">📝 消息内容</label>
            <textarea 
              v-model="formData.content" 
              class="pixel-textarea" 
              placeholder="输入你要发送的消息..."
              maxlength="500"
              rows="4"
              required
            ></textarea>
            <div class="char-count">{{ formData.content.length }}/500</div>
          </div>
          
          <div class="pixel-form__group">
            <label class="pixel-form__label text-responsive-base text-electric">🎭 消息类型</label>
            <select v-model="formData.type" class="pixel-select">
              <option value="normal">💬 普通消息</option>
              <option value="friend">👥 好友邀请</option>
              <option value="guild">🏰 公会邀请</option>
              <option value="trade">⚖️ 交易请求</option>
            </select>
          </div>
        </form>
      </div>
      
      <div class="pixel-dialog__footer">
        <button @click="closeDialog" class="pixel-btn pixel-btn--secondary">
          <Icon name="mdi:cancel" size="16" />
          取消
        </button>
        <button @click="sendMessage" class="pixel-btn pixel-btn--primary" 
                :disabled="!formData.recipient || !formData.content">
          <Icon name="mdi:send" size="16" />
          发送消息
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">

import { ref, reactive, computed, watch, nextTick } from 'vue'
import { NDialog, NInput, NButton, NAvatar, useMessage } from 'naive-ui'
import { Search, Add as Plus } from '@vicons/ionicons5'
import { useUserStore } from '@/stores/user'
import { useMessageService } from '~/services'
import { defaultImages } from '@/utils/assets'

// API实例
const message = useMessage()

interface Props {
  modelValue: boolean
  presetUserId?: string
  presetUserName?: string
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'message-sent', message: any): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 输入框引用
const recipientInputRef = ref<HTMLInputElement | null>(null)

// 表单数据接口
interface MessageFormData {
  recipient: string
  content: string
  type: 'normal' | 'friend' | 'guild' | 'trade'
}

// 用户接口
interface UserHint {
  id: number
  name: string
  avatar: string
}

const formData = reactive<MessageFormData>({
  recipient: '',
  content: '',
  type: 'normal'
})

// 用户搜索提示
const allUsers = ref<UserHint[]>([
  { id: 1001, name: '冒险者小明', avatar: defaultImages.userAvatar },
  { id: 1002, name: '法师莉莉', avatar: defaultImages.userAvatar },
  { id: 1003, name: '战士阿强', avatar: defaultImages.userAvatar },
  { id: 1004, name: '盗贼小红', avatar: defaultImages.userAvatar },
])

const recipientHints = computed(() => {
  if (!formData.recipient) return []
  return allUsers.value.filter(user => 
    user.name.toLowerCase().includes(formData.recipient.toLowerCase()) ||
    user.id.toString().includes(formData.recipient)
  ).slice(0, 3)
})

const selectRecipient = (user: any) => {
  formData.recipient = user.name
}

// 监听预设值变化
watch(() => props.modelValue, (newValue) => {
  if (newValue) {
    // 对话框打开时，如果有预设值就填入
    if (props.presetUserId && props.presetUserName) {
      formData.recipient = props.presetUserName
    }
    // 自动聚焦到收信人输入框
    nextTick(() => {
      recipientInputRef.value?.focus()
    })
  }
})

const closeDialog = () => {
  emit('update:modelValue', false)
  // 重置表单
  formData.recipient = ''
  formData.content = ''
  formData.type = 'normal'
}

const sendMessage = () => {
  if (!formData.recipient.trim() || !formData.content.trim()) return
  
  // 发送消息逻辑
  const newMessage = {
    id: Date.now(),
    recipient: formData.recipient,
    content: formData.content,
    type: formData.type,
    timestamp: new Date().toISOString()
  }
  
  emit('message-sent', newMessage)
  closeDialog()
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
    position: relative;
    
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
  min-height: 100px;
}

.pixel-select {
  cursor: pointer;
  
  option {
    background: #1a1a1a;
    color: #e0e0e0;
  }
}

.recipient-hints {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  background: #1a1a1a;
  border: 2px solid #444;
  border-top: none;
  max-height: 150px;
  overflow-y: auto;
  z-index: var(--z-elevated);
}

.recipient-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  transition: background 0.2s ease;
  
  &:hover {
    background: #333;
  }
  
  .hint-avatar {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: #444;
  }
  
  .hint-name {
    color: #e0e0e0;
    font-size: 14px;
  }
  
  .hint-id {
    color: #999;
    font-size: 12px;
    margin-left: auto;
  }
}

.char-count {
  text-align: right;
  color: #999;
  font-size: 12px;
  margin-top: 4px;
  font-family: 'Courier New', monospace;
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
.pixel-dialog__content::-webkit-scrollbar,
.recipient-hints::-webkit-scrollbar {
  width: 8px;
}

.pixel-dialog__content::-webkit-scrollbar-track,
.recipient-hints::-webkit-scrollbar-track {
  background: #1a1a1a;
  border-radius: 0;
}

.pixel-dialog__content::-webkit-scrollbar-thumb,
.recipient-hints::-webkit-scrollbar-thumb {
  background: #4a90e2;
  border-radius: 0;
  
  &:hover {
    background: #357abd;
  }
}
</style>

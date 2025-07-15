<template>
  <div v-if="modelValue" class="pixel-dialog-overlay" @click="closeDialog">
    <div class="pixel-dialog pixel-dialog--large" @click.stop>
      <div class="pixel-dialog__header">
        <h3 class="text-responsive-lg text-gradient-fire">👤 编辑个人资料</h3>
        <button @click="closeDialog" class="pixel-btn pixel-btn--close">
          <Icon name="mdi:close" size="16" />
        </button>
      </div>
      
      <div class="pixel-dialog__content">
        <form @submit.prevent="saveProfile" class="pixel-form">
          <!-- 头像部分 -->
          <div class="pixel-form__group">
            <label class="pixel-form__label text-responsive-base text-electric">🖼️ 头像</label>
            <div class="avatar-upload">
              <div class="avatar-preview">
                <NAvatar :size="100" :src="formData.avatar || defaultImages.userAvatar" />
              </div>
              <button type="button" @click="selectAvatar" class="pixel-btn pixel-btn--secondary">
                <Icon name="mdi:camera" size="16" />
                更换头像
              </button>
              <input ref="avatarInput" type="file" accept="image/*" @change="handleAvatarChange" style="display: none" />
            </div>
          </div>

          <!-- 基本信息 -->
          <div class="pixel-form__row">
            <div class="pixel-form__group">
              <label class="pixel-form__label text-responsive-base text-electric">👤 昵称</label>
              <input 
                ref="nicknameInputRef"
                v-model="formData.nickname" 
                type="text" 
                class="pixel-input" 
                placeholder="请输入昵称"
                maxlength="20"
                required
              />
            </div>
            
            <div class="pixel-form__group">
              <label class="pixel-form__label text-responsive-base text-electric">🆔 用户名</label>
              <input 
                v-model="formData.username" 
                type="text" 
                class="pixel-input" 
                placeholder="请输入用户名"
                maxlength="20"
                :disabled="isExistingUser"
              />
              <div v-if="isExistingUser" class="form-hint">用户名创建后不可修改</div>
            </div>
          </div>

          <!-- 联系信息 -->
          <div class="pixel-form__row">
            <div class="pixel-form__group">
              <label class="pixel-form__label text-responsive-base text-electric">📧 邮箱</label>
              <input 
                v-model="formData.email" 
                type="email" 
                class="pixel-input" 
                placeholder="请输入邮箱地址"
              />
            </div>
            
            <div class="pixel-form__group">
              <label class="pixel-form__label text-responsive-base text-electric">📱 手机号</label>
              <input 
                v-model="formData.phone" 
                type="tel" 
                class="pixel-input" 
                placeholder="请输入手机号码"
              />
            </div>
          </div>

          <!-- 个人简介 -->
          <div class="pixel-form__group">
            <label class="pixel-form__label text-responsive-base text-electric">📝 个人简介</label>
            <textarea 
              v-model="formData.bio" 
              class="pixel-textarea" 
              placeholder="介绍一下你自己吧..."
              maxlength="200"
              rows="4"
            ></textarea>
            <div class="char-count">{{ formData.bio.length }}/200</div>
          </div>

          <!-- 位置信息 -->
          <div class="pixel-form__group">
            <label class="pixel-form__label text-responsive-base text-electric">📍 所在地</label>
            <input 
              v-model="formData.location" 
              type="text" 
              class="pixel-input" 
              placeholder="请输入所在地"
            />
          </div>

          <!-- 隐私设置 -->
          <div class="pixel-form__group">
            <label class="pixel-form__label text-responsive-base text-electric">🔒 隐私设置</label>
            <div class="privacy-options">
              <label class="checkbox-option">
                <input type="checkbox" v-model="formData.showLocation" />
                <span class="checkbox-label">公开位置信息</span>
              </label>
              <label class="checkbox-option">
                <input type="checkbox" v-model="formData.showOnlineStatus" />
                <span class="checkbox-label">显示在线状态</span>
              </label>
              <label class="checkbox-option">
                <input type="checkbox" v-model="formData.allowFriendRequest" />
                <span class="checkbox-label">允许好友请求</span>
              </label>
            </div>
          </div>
        </form>
      </div>
      
      <div class="pixel-dialog__footer">
        <button @click="closeDialog" class="pixel-btn pixel-btn--secondary">
          <Icon name="mdi:cancel" size="16" />
          取消
        </button>
        <button @click="saveProfile" class="pixel-btn pixel-btn--primary" 
                :disabled="!formData.nickname.trim()" :loading="saving">
          <Icon name="mdi:content-save" size="16" />
          保存更改
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, nextTick } from 'vue'
import { NDialog, NForm, NInput, NButton, NAvatar } from 'naive-ui'
import { NFormItem, useMessage } from 'naive-ui'
import { Create as Edit } from '@vicons/ionicons5'
import ImageUploader from './ImageUploader.vue'
import { defaultImages } from '@/utils/assets'

interface Props {
  modelValue: boolean
  userData?: any
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'profile-updated', profile: any): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 表单引用
const nicknameInputRef = ref<HTMLInputElement | null>(null)
const avatarInput = ref<HTMLInputElement | null>(null)

// 保存状态
const saving = ref<boolean>(false)

// 表单数据接口
interface FormData {
  avatar: string
  nickname: string
  username: string
  email: string
  phone: string
  bio: string
  location: string
  showLocation: boolean
  showOnlineStatus: boolean
  allowFriendRequest: boolean
}

// 表单数据
const formData = reactive<FormData>({
  avatar: '',
  nickname: '',
  username: '',
  email: '',
  phone: '',
  bio: '',
  location: '',
  showLocation: true,
  showOnlineStatus: true,
  allowFriendRequest: true
})

// 是否为现有用户（用户名不可修改）
const isExistingUser = computed(() => {
  return props.userData?.username && props.userData.username.trim() !== ''
})

// 监听对话框打开，自动聚焦昵称输入框
watch(() => props.modelValue, (newValue) => {
  if (newValue) {
    // 填充现有数据
    if (props.userData) {
      Object.assign(formData, {
        avatar: props.userData.avatar_url || '',
        nickname: props.userData.nickname || '',
        username: props.userData.username || '',
        email: props.userData.email || '',
        phone: props.userData.phone || '',
        bio: props.userData.bio || '',
        location: props.userData.location || '',
        showLocation: props.userData.show_location ?? true,
        showOnlineStatus: props.userData.show_online_status ?? true,
        allowFriendRequest: props.userData.allow_friend_request ?? true
      })
    }
    
    // 自动聚焦
    nextTick(() => {
      nicknameInputRef.value?.focus()
    })
  }
})

const selectAvatar = () => {
  avatarInput.value?.click()
}

const handleAvatarChange = (event: Event) => {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (file) {
    // 这里应该上传文件到服务器
    const reader = new FileReader()
    reader.onload = (e) => {
      formData.avatar = e.target?.result as string
    }
    reader.readAsDataURL(file)
  }
}

const closeDialog = () => {
  emit('update:modelValue', false)
  // 重置表单
  Object.assign(formData, {
    avatar: '',
    nickname: '',
    username: '',
    email: '',
    phone: '',
    bio: '',
    location: '',
    showLocation: true,
    showOnlineStatus: true,
    allowFriendRequest: true
  })
}

const saveProfile = async () => {
  if (!formData.nickname.trim()) return
  
  saving.value = true
  try {
    // 保存个人资料逻辑
    const updatedProfile = {
      avatar_url: formData.avatar,
      nickname: formData.nickname,
      username: formData.username,
      email: formData.email,
      phone: formData.phone,
      bio: formData.bio,
      location: formData.location,
      show_location: formData.showLocation,
      show_online_status: formData.showOnlineStatus,
      allow_friend_request: formData.allowFriendRequest
    }
    
    emit('profile-updated', updatedProfile)
    closeDialog()
  } catch (error) {
    console.error('保存个人资料失败:', error)
  } finally {
    saving.value = false
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
  width: 90%;
  max-width: 700px;
  max-height: 80vh;
  overflow: hidden;
  box-shadow: 
    0 0 0 2px #000,
    0 0 20px rgba(74, 144, 226, 0.3),
    inset 0 0 0 1px rgba(255, 255, 255, 0.1);

  &--large {
    max-width: 700px;
  }
  
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
    max-height: 60vh;
    overflow-y: auto;
    
    &::-webkit-scrollbar {
      width: 8px;
    }
    
    &::-webkit-scrollbar-track {
      background: rgba(255, 255, 255, 0.1);
    }
    
    &::-webkit-scrollbar-thumb {
      background: #4a90e2;
      border-radius: 4px;
    }
  }
  
  &__footer {
    padding: 16px 20px;
    background: linear-gradient(135deg, #3a3a3a, #2a2a2a);
    border-top: 2px solid #4a90e2;
    display: flex;
    gap: 12px;
    justify-content: flex-end;
  }
}

.pixel-form {
  display: flex;
  flex-direction: column;
  gap: 20px;

  &__row {
    display: flex;
    gap: 16px;
    
    @media (max-width: 768px) {
      flex-direction: column;
    }
  }

  &__group {
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex: 1;
  }

  &__label {
    font-weight: bold;
    color: #4a90e2;
    font-size: 14px;
    text-shadow: 0 0 4px rgba(74, 144, 226, 0.3);
  }
}

.pixel-input, .pixel-textarea {
  background: rgba(255, 255, 255, 0.1);
  border: 2px solid #4a90e2;
  color: #ffffff;
  padding: 12px;
  font-family: 'Courier New', monospace;
  font-size: 14px;
  outline: none;
  transition: all 0.3s ease;

  &:focus {
    border-color: #00ffff;
    box-shadow: 0 0 0 2px rgba(0, 255, 255, 0.2);
  }

  &::placeholder {
    color: rgba(255, 255, 255, 0.6);
  }

  &:disabled {
    background: rgba(255, 255, 255, 0.05);
    color: rgba(255, 255, 255, 0.5);
    cursor: not-allowed;
  }
}

.pixel-textarea {
  resize: vertical;
  min-height: 100px;
}

.avatar-upload {
  display: flex;
  align-items: center;
  gap: 16px;
  
  .avatar-preview {
    width: 80px;
    height: 80px;
    border: 2px solid #4a90e2;
    border-radius: 50%;
    overflow: hidden;
    
    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }
}

.privacy-options {
  display: flex;
  flex-direction: column;
  gap: 12px;
  
  .checkbox-option {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    
    input[type="checkbox"] {
      width: 16px;
      height: 16px;
      accent-color: #4a90e2;
    }
    
    .checkbox-label {
      color: #ffffff;
      font-size: 14px;
    }
  }
}

.char-count {
  text-align: right;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.6);
}

.form-hint {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.6);
  margin-top: 4px;
}

.pixel-btn {
  background: linear-gradient(135deg, #4a90e2, #357abd);
  border: 2px solid #4a90e2;
  color: #ffffff;
  padding: 8px 16px;
  font-family: 'Courier New', monospace;
  font-size: 14px;
  font-weight: bold;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 8px;
  text-shadow: 0 0 4px rgba(74, 144, 226, 0.5);

  &:hover {
    background: linear-gradient(135deg, #357abd, #2a5f8f);
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(74, 144, 226, 0.3);
  }

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
    transform: none;
  }

  &--secondary {
    background: linear-gradient(135deg, #666666, #4a4a4a);
    border-color: #666666;
    text-shadow: 0 0 4px rgba(102, 102, 102, 0.5);

    &:hover {
      background: linear-gradient(135deg, #4a4a4a, #333333);
      box-shadow: 0 4px 12px rgba(102, 102, 102, 0.3);
    }
  }

  &--close {
    background: transparent;
    border: none;
    color: #ffffff;
    padding: 4px;
    font-size: 16px;

    &:hover {
      background: rgba(255, 255, 255, 0.1);
      transform: none;
    }
  }
}
</style>

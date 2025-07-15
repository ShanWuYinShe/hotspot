<!-- 微信分享组件 -->
<template>
  <div class="wechat-share">
    <NDialog
      v-model="visible"
      :title="title"
      width="90%"
      :style="{ maxWidth: '360px' }"
      @close="handleClose"
    >
      <div class="share-content">
        <!-- 分享预览 -->
        <div class="share-preview">
          <div class="preview-card">
            <div class="preview-image">
              <NImage 
                v-if="options.imgUrl" 
                :src="options.imgUrl" 
                fit="cover"
                class="preview-img"
              />
              <div v-else class="preview-placeholder">
                <NIcon><Image /></NIcon>
              </div>
            </div>
            <div class="preview-info">
              <h3 class="preview-title">{{ options.title }}</h3>
              <p class="preview-desc">{{ options.desc }}</p>
            </div>
          </div>
        </div>
        
        <!-- 微信分享选项 -->
        <div class="share-options">
          <el-row :gutter="20">
            <el-col :span="8">
              <div class="share-option" @click="handleShareToFriend">
                <div class="option-icon wechat">
                  <NIcon><ChatDotRound /></NIcon>
                </div>
                <div class="option-name">微信好友</div>
              </div>
            </el-col>
            <el-col :span="8">
              <div class="share-option" @click="handleShareToTimeline">
                <div class="option-icon moments">
                  <NIcon><Images /></NIcon>
                </div>
                <div class="option-name">朋友圈</div>
              </div>
            </el-col>
            <el-col :span="8">
              <div class="share-option" @click="handleCopyLink">
                <div class="option-icon copy">
                  <NIcon><Document /></NIcon>
                </div>
                <div class="option-name">复制链接</div>
              </div>
            </el-col>
          </el-row>
          
          <!-- 二维码分享（在浏览器环境下） -->
          <div v-if="!isWechat && qrCodeUrl" class="qr-share">
            <h4>或使用微信扫描二维码分享</h4>
            <div class="qr-container">
              <img :src="qrCodeUrl" alt="二维码" class="qr-image" />
            </div>
            <p class="qr-tip">打开微信扫一扫</p>
          </div>
        </div>
        
        <!-- 仅在非微信环境下显示的提示 -->
        <div v-if="!isWechat" class="browser-tip">
          <NAlert
            type="info"
            :closable="false"
          >
            <template #default>
              <p>当前不是在微信中访问，请点击右上角按钮，选择"在浏览器中打开"，然后复制链接分享给好友</p>
            </template>
          </NAlert>
        </div>
      </div>
      
      <template #action>
        <div class="dialog-footer">
          <NButton @click="handleClose">关闭</NButton>
        </div>
      </template>
    </NDialog>
  </div>
</template>

<script setup lang="ts">

import { NDialog, NImage, NIcon, NAlert, NButton, useMessage } from 'naive-ui'
import { computed, onMounted, ref, watch } from 'vue'
import { Image, Chatbubble as ChatDotRound, Images, Document } from '@vicons/ionicons5'
import QRCode from 'qrcode'
import { useWechatService } from '~/services'
import type { WxShareOptions } from '~/services/wechat.service'

// API实例
const message = useMessage()

// 服务实例
const wechatService = useWechatService()

interface Props {
  modelValue?: boolean
  title?: string
  shareOptions: Partial<WxShareOptions>
}

const props = withDefaults(defineProps<Props>(), {
  title: '分享',
  modelValue: false
})

const emit = defineEmits<{
  'update:modelValue': [visible: boolean]
  'close': []
  'share-success': [channel: 'friend' | 'timeline' | 'copy']
  'share-cancel': [channel: 'friend' | 'timeline']
  'share-fail': [error: any, channel: 'friend' | 'timeline' | 'copy']
}>()

// 状态
const visible = ref(props.modelValue)
const isWechat = ref(false)
const isJssdkReady = ref(false)
const qrCodeUrl = ref('')

// 分享选项
const options = computed<WxShareOptions>(() => ({
  title: props.shareOptions.title || document.title,
  desc: props.shareOptions.desc || '',
  link: props.shareOptions.link || window.location.href,
  imgUrl: props.shareOptions.imgUrl || '',
  success: () => {
    emit('share-success', currentChannel.value as 'friend' | 'timeline')
    message.success('分享成功')
    handleClose()
  },
  cancel: () => {
    emit('share-cancel', currentChannel.value as 'friend' | 'timeline')
  },
  fail: (error) => {
    emit('share-fail', error, currentChannel.value as 'friend' | 'timeline')
    message.error('分享失败')
  }
}))

// 当前选择的分享渠道
const currentChannel = ref<'friend' | 'timeline' | 'copy'>('friend')

// 监听对话框可见性
watch(() => props.modelValue, (newVal) => {
  visible.value = newVal
  if (newVal) {
    initShare()
  }
})

// 监听内部可见性变化
watch(visible, (newVal) => {
  emit('update:modelValue', newVal)
})

// 初始化
onMounted(() => {
  checkWechatEnvironment()
  if (visible.value) {
    initShare()
  }
})

// 检查是否在微信环境
const checkWechatEnvironment = () => {
  if (typeof window !== 'undefined') {
    const ua = window.navigator.userAgent.toLowerCase()
    isWechat.value = ua.includes('micromessenger')
  }
}

// 初始化分享
const initShare = async () => {
  if (isWechat.value) {
    // 在微信环境中，初始化JSSDK
    isJssdkReady.value = await wechatService.initWxJssdk()
  } else {
    // 在非微信环境中，生成二维码
    generateQRCode()
  }
}

// 生成二维码
const generateQRCode = async () => {
  try {
    const dataUrl = await QRCode.toDataURL(options.value.link, {
      width: 200,
      margin: 2,
      errorCorrectionLevel: 'M'
    })
    qrCodeUrl.value = dataUrl
  } catch (error) {
    console.error('生成二维码失败:', error)
  }
}

// 分享到微信好友
const handleShareToFriend = () => {
  currentChannel.value = 'friend'
  
  if (!isWechat.value) {
    message.info('请在微信中打开此页面')
    return
  }
  
  if (!isJssdkReady.value) {
    message.error('微信JSSDK未准备好，请稍后再试')
    return
  }
  
  wechatService.shareToWxFriend(options.value)
}

// 分享到朋友圈
const handleShareToTimeline = () => {
  currentChannel.value = 'timeline'
  
  if (!isWechat.value) {
    message.info('请在微信中打开此页面')
    return
  }
  
  if (!isJssdkReady.value) {
    message.error('微信JSSDK未准备好，请稍后再试')
    return
  }
  
  wechatService.shareToWxTimeline(options.value)
}

// 复制链接
const handleCopyLink = async () => {
  currentChannel.value = 'copy'
  
  try {
    await navigator.clipboard.writeText(options.value.link)
    message.success('链接已复制到剪贴板')
    emit('share-success', 'copy')
  } catch (err) {
    console.error('复制失败:', err)
    
    // 降级复制方法
    try {
      const textArea = document.createElement('textarea')
      textArea.value = options.value.link
      document.body.appendChild(textArea)
      textArea.select()
      document.execCommand('copy')
      document.body.removeChild(textArea)
      message.success('链接已复制到剪贴板')
      emit('share-success', 'copy')
    } catch (fallbackErr) {
      message.error('复制失败，请手动复制')
      emit('share-fail', fallbackErr, 'copy')
    }
  }
}

// 关闭对话框
const handleClose = () => {
  visible.value = false
  emit('close')
}

// 显示分享对话框
const show = (shareOptions?: Partial<WxShareOptions>) => {
  visible.value = true
  
  // 如果有新的分享选项，重新初始化
  if (shareOptions) {
    initShare()
  }
}

// 隐藏分享对话框
const hide = () => {
  visible.value = false
}

// 暴露方法
defineExpose({
  show,
  hide
})
</script>

<style scoped lang="scss">
.wechat-share {
  display: inline-block;
}

.share-content {
  .share-preview {
    margin-bottom: 1.5rem;
    
    .preview-card {
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 8px;
      overflow: hidden;
      
      .preview-image {
        height: 150px;
        background-color: #f5f5f5;
        position: relative;
        
        .preview-img {
          width: 100%;
          height: 100%;
          object-fit: cover;
        }
        
        .preview-placeholder {
          display: flex;
          align-items: center;
          justify-content: center;
          height: 100%;
          
          .el-icon {
            font-size: 48px;
            color: #ccc;
          }
        }
      }
      
      .preview-info {
        padding: 1rem;
        
        .preview-title {
          margin: 0 0 0.5rem 0;
          font-size: 1rem;
          line-height: 1.4;
          color: var(--el-text-color-primary);
          display: -webkit-box;
          -webkit-line-clamp: 2;
          line-clamp: 2;
          -webkit-box-orient: vertical;
          overflow: hidden;
        }
        
        .preview-desc {
          margin: 0;
          font-size: 0.875rem;
          color: var(--color-text-secondary);
          display: -webkit-box;
          -webkit-line-clamp: 2;
          line-clamp: 2;
          -webkit-box-orient: vertical;
          overflow: hidden;
          line-height: 1.4;
        }
      }
    }
  }
  
  .share-options {
    margin-bottom: 1.5rem;
    
    .share-option {
      display: flex;
      flex-direction: column;
      align-items: center;
      cursor: pointer;
      padding: 0.5rem;
      
      .option-icon {
        width: 48px;
        height: 48px;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        margin-bottom: 0.5rem;
        
        .el-icon {
          font-size: 24px;
          color: white;
        }
        
        &.wechat {
          background-color: #09bb07;
        }
        
        &.moments {
          background-color: #08c060;
        }
        
        &.copy {
          background-color: #1e88e5;
        }
      }
      
      .option-name {
        font-size: 0.875rem;
        color: var(--color-text-primary);
      }
      
      &:hover {
        background-color: #f5f5f5;
        border-radius: 8px;
      }
    }
    
    .qr-share {
      margin-top: 1.5rem;
      text-align: center;
      
      h4 {
        font-size: 0.875rem;
        font-weight: normal;
        color: var(--color-text-secondary);
        margin: 0 0 1rem 0;
      }
      
      .qr-container {
        display: flex;
        justify-content: center;
        margin-bottom: 0.5rem;
        
        .qr-image {
          width: 150px;
          height: 150px;
          border: 1px solid var(--color-border-secondary);
          padding: 0.5rem;
          border-radius: 4px;
        }
      }
      
      .qr-tip {
        font-size: 0.75rem;
        color: var(--color-text-secondary);
        margin: 0;
      }
    }
  }
  
  .browser-tip {
    margin-top: 1.5rem;
    
    :deep(.el-alert__content) {
      p {
        font-size: 0.75rem;
        margin: 0;
      }
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: center;
}
</style>

<!-- 二维码生成组件 -->
<template>
  <div class="qr-code-generator">
    <NDialog
      v-model="visible"
      :title="title"
      width="90%"
      :style="{ maxWidth: '400px' }"
      @close="handleClose"
    >
      <div class="qr-content">
        <!-- 二维码显示区域 -->
        <div class="qr-display">
          <div v-if="loading" class="qr-loading">
            <NSkeleton animated height="250px" width="250px" />
            <p>生成二维码中...</p>
          </div>
          
          <div v-else-if="error" class="qr-error">
            <NIcon class="error-icon"><Warning /></NIcon>
            <p>{{ error }}</p>
            <NButton size="small" @click="generateQRCode">重新生成</NButton>
          </div>
          
          <div v-else-if="qrCodeDataUrl" class="qr-image">
            <img :src="qrCodeDataUrl" :alt="title" class="qr-img" />
            <div class="qr-info">
              <h4>{{ qrTitle || title }}</h4>
              <p v-if="description">{{ description }}</p>
            </div>
          </div>
        </div>
        
        <!-- 操作按钮 -->
        <div class="qr-actions">
          <NButton
            @click="copyToClipboard"
            :icon="Document"
            :disabled="!content"
          >
            复制链接
          </NButton>
          <NButton
            @click="downloadQRCode"
            :icon="Download"
            :disabled="!qrCodeDataUrl"
          >
            下载二维码
          </NButton>
          <NButton
            @click="shareQRCode"
            :icon="Share"
            :disabled="!qrCodeDataUrl"
          >
            分享
          </NButton>
        </div>
        
        <!-- 二维码设置 -->
        <NCollapse v-if="showSettings" class="qr-settings">
          <NCollapseItem title="高级设置" name="settings">
            <NForm label-width="80px" size="small">
              <NFormItem label="尺寸">
                <NSlider
                  v-model:value="qrOptions.width"
                  :min="150"
                  :max="400"
                  :step="50"
                  @update:value="generateQRCode"
                />
              </NFormItem>
              <NFormItem label="颜色">
                <NColorPicker
                  v-model:value="qrOptions.color.dark"
                  @update:value="generateQRCode"
                />
              </NFormItem>
              <NFormItem label="背景">
                <NColorPicker
                  v-model:value="qrOptions.color.light"
                  @update:value="generateQRCode"
                />
              </NFormItem>
              <NFormItem label="容错级别">
                <NSelect
                  v-model:value="qrOptions.errorCorrectionLevel"
                  @update:value="generateQRCode"
                  placement="bottom-start"
                  :options="[
                    { label: '低 (L)', value: 'L' },
                    { label: '中 (M)', value: 'M' },
                    { label: '高 (Q)', value: 'Q' },
                    { label: '最高 (H)', value: 'H' }
                  ]"
                />
              </NFormItem>
            </NForm>
          </NCollapseItem>
        </NCollapse>
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
import { NDialog, NSkeleton, NIcon, NButton, NCollapse, NCollapseItem, NForm, NFormItem, NSlider, NColorPicker, NSelect, useMessage } from 'naive-ui'
import QRCode from 'qrcode'
import { Warning, Document, Download, Share } from '@vicons/ionicons5'

// API实例
const message = useMessage()

interface QRCodeOptions {
  width: number
  margin: number
  color: {
    dark: string
    light: string
  }
  errorCorrectionLevel: 'L' | 'M' | 'Q' | 'H'
}

interface Props {
  modelValue?: boolean
  title?: string
  qrTitle?: string
  description?: string
  content: string
  showSettings?: boolean
  defaultOptions?: Partial<QRCodeOptions>
}

const props = withDefaults(defineProps<Props>(), {
  title: '二维码',
  showSettings: false,
  defaultOptions: () => ({
    width: 250,
    margin: 2,
    color: {
      dark: '#000000',
      light: '#FFFFFF'
    },
    errorCorrectionLevel: 'M'
  })
})

const emit = defineEmits<{
  'update:modelValue': [visible: boolean]
  'close': []
  'generated': [dataUrl: string]
  'share': [content: string, dataUrl: string]
}>()

// 状态
const visible = ref(props.modelValue || false)
const loading = ref(false)
const error = ref('')
const qrCodeDataUrl = ref('')

// 二维码选项
const qrOptions = ref<QRCodeOptions>({
  width: 250,
  margin: 2,
  color: {
    dark: '#000000',
    light: '#FFFFFF'
  },
  errorCorrectionLevel: 'M',
  ...props.defaultOptions
})

// 监听外部状态变化
watch(() => props.modelValue, (newVal) => {
  visible.value = newVal || false
  if (newVal && props.content && !qrCodeDataUrl.value) {
    generateQRCode()
  }
})

// 监听内部状态变化
watch(visible, (newVal) => {
  emit('update:modelValue', newVal)
  if (newVal && props.content && !qrCodeDataUrl.value) {
    generateQRCode()
  }
})

// 监听内容变化
watch(() => props.content, (newVal) => {
  if (newVal && visible.value) {
    generateQRCode()
  }
})

// 生成二维码
const generateQRCode = async () => {
  if (!props.content) {
    error.value = '没有要生成的内容'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const dataUrl = await QRCode.toDataURL(props.content, {
      width: qrOptions.value.width,
      margin: qrOptions.value.margin,
      color: qrOptions.value.color,
      errorCorrectionLevel: qrOptions.value.errorCorrectionLevel
    })

    qrCodeDataUrl.value = dataUrl
    emit('generated', dataUrl)
  } catch (err) {
    console.error('生成二维码失败:', err)
    error.value = '生成二维码失败'
    qrCodeDataUrl.value = ''
  } finally {
    loading.value = false
  }
}

// 复制到剪贴板
const copyToClipboard = async () => {
  if (!props.content) return

  try {
    await navigator.clipboard.writeText(props.content)
    message.success('链接已复制到剪贴板')
  } catch (err) {
    console.error('复制失败:', err)
    message.error('复制失败')
    
    // 降级方案
    try {
      const textArea = document.createElement('textarea')
      textArea.value = props.content
      document.body.appendChild(textArea)
      textArea.select()
      document.execCommand('copy')
      document.body.removeChild(textArea)
      message.success('链接已复制到剪贴板')
    } catch (fallbackErr) {
      message.error('复制失败，请手动复制')
    }
  }
}

// 下载二维码
const downloadQRCode = () => {
  if (!qrCodeDataUrl.value) return

  try {
    const link = document.createElement('a')
    link.download = `qr-code-${Date.now()}.png`
    link.href = qrCodeDataUrl.value
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    
    message.success('二维码已下载')
  } catch (err) {
    console.error('下载失败:', err)
    message.error('下载失败')
  }
}

// 分享二维码
const shareQRCode = async () => {
  if (!qrCodeDataUrl.value || !props.content) return

  try {
    if (navigator.share) {
      // 使用原生分享 API（如果支持）
      await navigator.share({
        title: props.qrTitle || props.title,
        text: props.description || '扫描二维码查看',
        url: props.content
      })
    } else {
      // 检查是否在微信环境
      const isWechat = localStorage.getItem('isWechatBrowser') === 'true'
      
      if (isWechat) {
        // 在微信环境中，使用微信分享
        emit('share', props.content, qrCodeDataUrl.value)
      } else {
        // 其他环境，显示分享提示
        emit('share', props.content, qrCodeDataUrl.value)
        message.info('请选择分享方式')
      }
    }
  } catch (err) {
    if (err instanceof Error && err.name !== 'AbortError') {
      console.error('分享失败:', err)
      message.error('分享失败')
    }
  }
}

// 关闭对话框
const handleClose = () => {
  visible.value = false
  emit('close')
}

// 显示二维码
const show = (content?: string) => {
  if (content) {
    // 如果提供了新内容，重置二维码
    qrCodeDataUrl.value = ''
  }
  visible.value = true
}

// 隐藏二维码
const hide = () => {
  visible.value = false
}

// 重新生成
const regenerate = () => {
  qrCodeDataUrl.value = ''
  generateQRCode()
}

// 暴露方法
defineExpose({
  show,
  hide,
  regenerate,
  generateQRCode
})
</script>

<style scoped lang="scss">
.qr-code-generator {
  display: inline-block;
}

.qr-content {
  .qr-display {
    text-align: center;
    margin-bottom: 1.5rem;
    
    .qr-loading,
    .qr-error {
      padding: 2rem 1rem;
      
      p {
        margin: 1rem 0 0 0;
        color: var(--el-text-color-secondary);
      }
    }
    
    .qr-error {
      .error-icon {
        font-size: 2rem;
        color: var(--el-color-error);
        margin-bottom: 1rem;
      }
    }
    
    .qr-image {
      .qr-img {
        max-width: 100%;
        height: auto;
        border-radius: 8px;
        border: 1px solid var(--el-border-color-lighter);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      }
      
      .qr-info {
        margin-top: 1rem;
        
        h4 {
          margin: 0 0 0.5rem 0;
          font-size: 1.125rem;
          color: var(--el-text-color-primary);
        }
        
        p {
          margin: 0;
          font-size: 0.875rem;
          color: var(--el-text-color-secondary);
          line-height: 1.4;
        }
      }
    }
  }
  
  .qr-actions {
    display: flex;
    justify-content: center;
    gap: 0.75rem;
    margin-bottom: 1rem;
    flex-wrap: wrap;
    
    .el-button {
      flex: 1;
      min-width: auto;
    }
  }
  
  .qr-settings {
    margin-top: 1rem;
    
    :deep(.el-collapse-item__header) {
      font-size: 0.875rem;
      padding: 0.75rem 0;
    }
    
    :deep(.el-collapse-item__content) {
      padding-bottom: 0;
    }
    
    .el-form {
      .el-form-item {
        margin-bottom: 1rem;
        
        :deep(.el-form-item__label) {
          font-size: 0.875rem;
        }
      }
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: center;
}

@media (max-width: 768px) {
  .qr-content {
    .qr-actions {
      flex-direction: column;
      
      .el-button {
        width: 100%;
      }
    }
  }
}
</style>

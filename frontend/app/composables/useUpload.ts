/**
 * 文件上传 Composable - 简洁而强大
 * 提供文件上传、预览和管理功能
 */

export interface UploadFile {
  id: string
  file: File
  name: string
  size: number
  type: string
  url?: string
  preview?: string
  progress: number
  status: 'pending' | 'uploading' | 'success' | 'error'
  error?: string
}

export interface UploadOptions {
  accept?: string
  multiple?: boolean
  maxSize?: number // bytes
  maxFiles?: number
  autoUpload?: boolean
  endpoint?: string
}

export function useUpload(options: UploadOptions = {}) {
  const {
    accept = '*/*',
    multiple = false,
    maxSize = 10 * 1024 * 1024, // 10MB
    maxFiles = 10,
    autoUpload = true,
    endpoint = '/upload'
  } = options

  const files = ref<UploadFile[]>([])
  const uploading = ref(false)
  const dragOver = ref(false)

  // 计算属性
  const totalFiles = computed(() => files.value.length)
  const uploadedFiles = computed(() => files.value.filter(f => f.status === 'success'))
  const failedFiles = computed(() => files.value.filter(f => f.status === 'error'))
  const pendingFiles = computed(() => files.value.filter(f => f.status === 'pending'))
  const uploadingFiles = computed(() => files.value.filter(f => f.status === 'uploading'))
  
  const totalProgress = computed(() => {
    if (files.value.length === 0) return 0
    const total = files.value.reduce((sum, file) => sum + file.progress, 0)
    return Math.round(total / files.value.length)
  })

  // Toast 辅助函数
  const toast = useToast()
  const showError = (title: string, description?: string) => {
    toast.add({ title, description, color: 'error' })
  }
  const success = (title: string, description?: string) => {
    toast.add({ title, description, color: 'success' })
  }
  const warning = (title: string, description?: string) => {
    toast.add({ title, description, color: 'warning' })
  }

  /**
   * 创建文件ID
   */
  const createFileId = () => {
    return `file_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }

  /**
   * 验证文件
   */
  const validateFile = (file: File): string | null => {
    // 检查文件大小
    if (file.size > maxSize) {
      return `文件大小不能超过 ${formatFileSize(maxSize)}`
    }

    // 检查文件类型
    if (accept !== '*/*') {
      const acceptTypes = accept.split(',').map(type => type.trim())
      const isValidType = acceptTypes.some(type => {
        if (type.startsWith('.')) {
          return file.name.toLowerCase().endsWith(type.toLowerCase())
        }
        return file.type.match(type.replace('*', '.*'))
      })
      
      if (!isValidType) {
        return `不支持的文件类型: ${file.type}`
      }
    }

    return null
  }

  /**
   * 创建文件预览
   */
  const createPreview = (file: File): Promise<string | undefined> => {
    return new Promise((resolve) => {
      if (file.type.startsWith('image/')) {
        const reader = new FileReader()
        reader.onload = (e) => resolve(e.target?.result as string)
        reader.onerror = () => resolve(undefined)
        reader.readAsDataURL(file)
      } else {
        resolve(undefined)
      }
    })
  }

  /**
   * 添加文件
   */
  const addFiles = async (fileList: FileList | File[]) => {
    const newFiles: UploadFile[] = []
    const fileArray = Array.from(fileList)

    // 检查文件数量限制
    if (files.value.length + fileArray.length > maxFiles) {
      showError('文件数量超限', `最多只能上传 ${maxFiles} 个文件`)
      return
    }

    for (const file of fileArray) {
      // 验证文件
      const error = validateFile(file)
      if (error) {
        showError('文件验证失败', error)
        continue
      }

      // 创建上传文件对象
      const uploadFile: UploadFile = {
        id: createFileId(),
        file,
        name: file.name,
        size: file.size,
        type: file.type,
        progress: 0,
        status: 'pending'
      }

      // 创建预览
      uploadFile.preview = await createPreview(file)

      newFiles.push(uploadFile)
    }

    files.value.push(...newFiles)

    // 自动上传
    if (autoUpload && newFiles.length > 0) {
      await uploadFiles(newFiles.map(f => f.id))
    }
  }

  /**
   * 上传单个文件
   */
  const uploadSingleFile = async (fileId: string): Promise<boolean> => {
    const fileToUpload = files.value.find(f => f.id === fileId)
    if (!fileToUpload || fileToUpload.status === 'uploading') {
      return false
    }

    fileToUpload.status = 'uploading'
    fileToUpload.progress = 0
    fileToUpload.error = undefined

    try {
      const formData = new FormData()
      formData.append('file', fileToUpload.file)

      // 使用 XMLHttpRequest 以支持进度回调
      const response = await new Promise<any>((resolve, reject) => {
        const xhr = new XMLHttpRequest()

        xhr.upload.onprogress = (event) => {
          if (event.lengthComputable) {
            fileToUpload.progress = Math.round((event.loaded / event.total) * 100)
          }
        }

        xhr.onload = () => {
          if (xhr.status >= 200 && xhr.status < 300) {
            try {
              const result = JSON.parse(xhr.responseText)
              resolve(result)
            } catch (error) {
              reject(new Error('响应解析失败'))
            }
          } else {
            reject(new Error(`上传失败: ${xhr.statusText}`))
          }
        }

        xhr.onerror = () => reject(new Error('网络错误'))
        xhr.ontimeout = () => reject(new Error('上传超时'))

        // 设置请求
        xhr.open('POST', endpoint)
        
        // 添加认证头
        const authStore = useAuthStore()
        if (authStore.token) {
          xhr.setRequestHeader('Authorization', `Bearer ${authStore.token}`)
        }

        xhr.timeout = 60000 // 60秒超时
        xhr.send(formData)
      })

      // 上传成功
      fileToUpload.status = 'success'
      fileToUpload.progress = 100
      fileToUpload.url = response.url || response.file_url

      return true
    } catch (error: any) {
      fileToUpload.status = 'error'
      fileToUpload.error = error.message || '上传失败'
      
      showError('上传失败', `${fileToUpload.name}: ${fileToUpload.error}`)
      
      return false
    }
  }

  /**
   * 批量上传文件
   */
  const uploadFiles = async (fileIds?: string[]): Promise<boolean> => {
    const targetFiles = fileIds 
      ? files.value.filter(f => fileIds.includes(f.id))
      : files.value.filter(f => f.status === 'pending')

    if (targetFiles.length === 0) return true

    uploading.value = true

    try {
      const results = await Promise.all(
        targetFiles.map(f => uploadSingleFile(f.id))
      )

      const successCount = results.filter(Boolean).length
      
      if (successCount === targetFiles.length) {
        success('上传完成', `成功上传 ${successCount} 个文件`)
      } else {
        warning('部分上传失败', `${successCount}/${targetFiles.length} 个文件上传成功`)
      }

      return successCount === targetFiles.length
    } finally {
      uploading.value = false
    }
  }

  /**
   * 重试上传
   */
  const retryUpload = async (fileId: string) => {
    const fileToRetry = files.value.find(f => f.id === fileId)
    if (fileToRetry && fileToRetry.status === 'error') {
      fileToRetry.status = 'pending'
      await uploadSingleFile(fileId)
    }
  }

  /**
   * 删除文件
   */
  const removeFile = (fileId: string) => {
    const index = files.value.findIndex(f => f.id === fileId)
    if (index !== -1) {
      files.value.splice(index, 1)
    }
  }

  /**
   * 清空所有文件
   */
  const clearFiles = () => {
    files.value = []
  }

  /**
   * 格式化文件大小
   */
  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
  }

  /**
   * 获取文件类型图标
   */
  const getFileIcon = (type: string): string => {
    if (type.startsWith('image/')) return 'heroicons:photo'
    if (type.startsWith('video/')) return 'heroicons:video-camera'
    if (type.startsWith('audio/')) return 'heroicons:musical-note'
    if (type.includes('pdf')) return 'heroicons:document-text'
    if (type.includes('word')) return 'heroicons:document-text'
    if (type.includes('excel') || type.includes('spreadsheet')) return 'heroicons:table-cells'
    if (type.includes('zip') || type.includes('rar')) return 'heroicons:archive-box'
    return 'heroicons:document'
  }

  /**
   * 拖拽处理
   */
  const onDragOver = (event: DragEvent) => {
    event.preventDefault()
    dragOver.value = true
  }

  const onDragLeave = (event: DragEvent) => {
    event.preventDefault()
    dragOver.value = false
  }

  const onDrop = async (event: DragEvent) => {
    event.preventDefault()
    dragOver.value = false

    const droppedFiles = event.dataTransfer?.files
    if (droppedFiles && droppedFiles.length > 0) {
      await addFiles(droppedFiles)
    }
  }

  /**
   * 文件选择处理
   */
  const onFileSelect = async (event: Event) => {
    const target = event.target as HTMLInputElement
    const selectedFiles = target.files
    
    if (selectedFiles && selectedFiles.length > 0) {
      await addFiles(selectedFiles)
    }

    // 清空 input 值，允许重复选择同一文件
    target.value = ''
  }

  return {
    // 状态
    files: readonly(files),
    uploading: readonly(uploading),
    dragOver: readonly(dragOver),

    // 计算属性
    totalFiles: readonly(totalFiles),
    uploadedFiles: readonly(uploadedFiles),
    failedFiles: readonly(failedFiles),
    pendingFiles: readonly(pendingFiles),
    uploadingFiles: readonly(uploadingFiles),
    totalProgress: readonly(totalProgress),

    // 方法
    addFiles,
    uploadFile: uploadSingleFile,
    uploadFiles,
    retryUpload,
    removeFile,
    clearFiles,
    formatFileSize,
    getFileIcon,

    // 事件处理
    onDragOver,
    onDragLeave,
    onDrop,
    onFileSelect
  }
}

/**
 * 文件相关工具函数
 */

/**
 * 根据 MIME 类型获取文件图标
 */
export function getFileIcon(type: string): string {
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
 * 验证文件类型
 */
export function validateFileType(file: File, accept: string): boolean {
  if (accept === '*/*') return true
  
  const acceptTypes = accept.split(',').map(type => type.trim())
  return acceptTypes.some(type => {
    if (type.startsWith('.')) {
      return file.name.toLowerCase().endsWith(type.toLowerCase())
    }
    return file.type.match(type.replace('*', '.*'))
  })
}

/**
 * 验证文件大小
 */
export function validateFileSize(file: File, maxSize: number): boolean {
  return file.size <= maxSize
}

/**
 * 格式化文件大小
 */
export function formatFileSize(bytes: number | null | undefined): string {
  if (!bytes || bytes === 0) return '0 B'
  
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}


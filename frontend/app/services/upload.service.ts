/**
 * 文件上传服务
 * 提供文件上传、多部分上传等功能
 * 
 * 符合 API 规范 3.16 节
 */

import { BaseService } from './base.service'
import type {
  UploadMediaData,
  InitiateUploadData,
  CompleteUploadRequest,
  UploadStatusData
} from '~/types/api'

/**
 * 文件上传服务类
 */
export class UploadService extends BaseService {
  constructor() {
    super('/api/v1/protected/media')
  }

  /**
   * 上传文件（≤100MB）
   * 3.16.1. POST /api/v1/protected/media/upload_file - 上传文件（≤100MB）
   */
  async uploadFile(file: File, category: 'avatar' | 'group_image' | 'moment_image' | 'file'): Promise<UploadMediaData> {
    return this.upload<UploadMediaData>('/upload_file', file, { category })
  }

  /**
   * 初始化大文件上传（>100MB）
   * 3.16.2. POST /api/v1/protected/media/initiate_upload - 初始化大文件上传（>100MB）
   */
  async initiateUpload(request: {
    file_name: string
    file_size: number
    mime_type: string
    chunk_size?: number
  }): Promise<InitiateUploadData> {
    return this.post<InitiateUploadData>('/initiate_upload', request)
  }

  /**
   * 完成大文件上传
   * 3.16.3. POST /api/v1/protected/media/complete_upload - 完成大文件上传
   */
  async completeUpload(request: CompleteUploadRequest): Promise<UploadMediaData> {
    return this.post<UploadMediaData>('/complete_upload', request)
  }

  /**
   * 查询上传状态
   * 3.16.4. POST /api/v1/protected/media/get_upload_status - 查询上传状态
   */
  async getUploadStatus(uploadId: string): Promise<UploadStatusData> {
    return this.post<UploadStatusData>('/get_upload_status', { upload_id: uploadId })
  }

  /**
   * 取消大文件上传
   * 3.16.5. POST /api/v1/protected/media/abort_upload - 取消大文件上传
   */
  async abortUpload(uploadId: string): Promise<void> {
    await this.post<void>('/abort_upload', { upload_id: uploadId })
  }

  /**
   * 获取文件预览URL（客户端工具方法）
   */
  getPreviewUrl(file: File): string {
    return URL.createObjectURL(file)
  }

  /**
   * 释放预览URL（客户端工具方法）
   */
  revokePreviewUrl(url: string): void {
    URL.revokeObjectURL(url)
  }

  /**
   * 格式化文件大小（客户端工具方法）
   */
  formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 B'
    
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }
}

/**
 * 使用上传服务的 Composable
 */
export function useUploadService() {
  return new UploadService()
}



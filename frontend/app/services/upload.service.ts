import { BaseService } from './base.service'
import { MEDIA } from '~/config/api'
import type { UploadMediaData, CompleteUploadRequest } from '~/types/api'

// ============ 上传类型定义 ============

export interface InitiateUploadRequest {
  file_name: string
  file_size: number
  mime_type: string
  chunk_size?: number
}

export interface InitiateUploadData {
  upload_id: string
  chunk_count: number
  upload_urls: string[]
}

export interface UploadChunkData {
  part_number: number
  etag: string
}

export interface GetUploadStatusRequest {
  upload_id: string
}

export interface UploadStatusData {
  upload_id: string
  status: 'in_progress' | 'completed' | 'failed'
  uploaded_parts: number[]
  total_parts: number
  created_at: string
  expires_at: string
}

export interface AbortUploadRequest {
  upload_id: string
}

// ============ 服务类 ============

class UploadService extends BaseService {
  async uploadFile(file: File, category: 'avatar' | 'group_image' | 'moment_image' | 'file') {
    return this.upload<UploadMediaData>(MEDIA.UPLOAD_FILE, file, { category })
  }

  async initiateUpload(request: InitiateUploadRequest) {
    return this.post<InitiateUploadData>(MEDIA.INITIATE_UPLOAD, request)
  }

  async uploadChunk(uploadId: string, partNumber: number, chunk: Blob) {
    const formData = new FormData()
    formData.append('upload_id', uploadId)
    formData.append('part_number', String(partNumber))
    formData.append('chunk', chunk)

    const config = useRuntimeConfig()
    const authStore = useAuthStore()
    const headers: Record<string, string> = {}
    if (authStore.token) {
      headers.Authorization = `Bearer ${authStore.token}`
    }

    const response = await $fetch<{ code: number; data: UploadChunkData }>(MEDIA.UPLOAD_CHUNK, {
      method: 'POST',
      baseURL: config.public.apiBaseUrl as string,
      headers,
      body: formData
    })

    if (response.code !== 0) {
      throw new Error('分块上传失败')
    }
    return response.data
  }

  async completeUpload(request: CompleteUploadRequest) {
    return this.post<UploadMediaData>(MEDIA.COMPLETE_UPLOAD, request)
  }

  async getUploadStatus(request: GetUploadStatusRequest) {
    return this.post<UploadStatusData>(MEDIA.GET_UPLOAD_STATUS, request)
  }

  async abortUpload(request: AbortUploadRequest) {
    return this.post<void>(MEDIA.ABORT_UPLOAD, request)
  }
}

export const uploadService = new UploadService()
export function useUploadService() { return uploadService }

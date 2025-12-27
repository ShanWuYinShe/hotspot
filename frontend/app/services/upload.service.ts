import { BaseService } from './base.service'
import { MEDIA } from '~/config/api'
import type { 
  UploadMediaData, 
  InitiateUploadRequest,
  InitiateUploadData,
  UploadChunkData,
  CompleteUploadRequest,
  UploadStatusRequest,
  UploadStatusData,
  AbortUploadRequest
} from '~/types/api'

// ============ 服务类 ============

type UploadCategory = 'avatar' | 'group_avatar' | 'moment_image' | 'chat_image' | 'chat_video' | 'chat_audio' | 'chat_file';

class UploadService extends BaseService {
  async uploadFile(file: File, category: UploadCategory) {
    return this.upload<UploadMediaData>(MEDIA.UPLOAD, file, { category })
  }

  async initiateUpload(request: InitiateUploadRequest) {
    return this.post<InitiateUploadData>(MEDIA.INITIATE_UPLOAD, request)
  }

  async uploadChunk(uploadId: string, partNumber: number, chunk: Blob) {
    const formData = new FormData()
    formData.append('upload_id', uploadId)
    formData.append('part_number', String(partNumber))
    formData.append('chunk', chunk)

    return this.post<UploadChunkData>(MEDIA.UPLOAD_CHUNK, formData)
  }

  async completeUpload(request: CompleteUploadRequest) {
    return this.post<UploadMediaData>(MEDIA.COMPLETE_UPLOAD, request)
  }

  async getUploadStatus(request: UploadStatusRequest) {
    return this.post<UploadStatusData>(MEDIA.UPLOAD_STATUS, request)
  }

  async abortUpload(request: AbortUploadRequest) {
    return this.post<void>(MEDIA.ABORT_UPLOAD, request)
  }
}

export const uploadService = new UploadService()
export function useUploadService() { return uploadService }

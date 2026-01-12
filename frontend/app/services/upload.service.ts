import { BaseService } from './base.service'
import { MEDIA } from '~/config/api'
import type {
  UploadMediaRequest,
  UploadMediaData,
  PresignUploadRequest,
  PresignUploadData,
  CompleteUploadRequest
} from '~/types/api'

// ============ 服务类 ============

type UploadCategory = 'avatar' | 'group_avatar' | 'moment_image' | 'chat_image' | 'chat_video' | 'chat_audio' | 'chat_file';

class UploadService extends BaseService {
  /**
   * 上传文件（简单上传模式：Presigned URL）
   * 流程：
   * 1. 请求后端获取预签名 URL (POST /presign_upload)
   * 2. 直接 PUT 文件到对象存储
   * 3. 通知后端完成上传 (POST /complete_upload)
   */
  async uploadFile(file: File, category: UploadCategory): Promise<UploadMediaData> {
    // 1. 获取预签名 URL
    const presignReq: PresignUploadRequest = {
      file_name: file.name,
      file_size: file.size,
      mime_type: file.type,
      category: category
    }
    const presignData = await this.post<PresignUploadData>(MEDIA.PRESIGN_UPLOAD, presignReq)

    // 2. 上传文件到对象存储
    await this.uploadToS3(presignData.upload_url, file, presignData.headers)

    // 3. 完成上传
    const completeReq: CompleteUploadRequest = {
      upload_id: presignData.upload_id
    }
    return this.post<UploadMediaData>(MEDIA.COMPLETE_UPLOAD, completeReq)
  }

  /**
   * 直接上传到 S3/MinIO (使用 fetch PUT)
   */
  private async uploadToS3(url: string, file: File, headers: Record<string, string>) {
    try {
      const response = await fetch(url, {
        method: 'PUT',
        headers: {
          ...headers,
          'Content-Type': file.type
        },
        body: file
      })

      if (!response.ok) {
        throw new Error(`Upload failed: ${response.statusText}`)
      }
    } catch (error) {
      console.error('S3 Upload Error:', error)
      throw error
    }
  }

  // Chunked Upload related methods would go here (omitted for now as backend implementation is partial)
}

export const uploadService = new UploadService()
export function useUploadService() { return uploadService }

/**
 * 文件上传服务
 */

import { post, generateIdempotencyKey } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  PresignUploadData,
  UploadMediaData,
  UploadCategory,
} from '@/types/api';

export interface PresignParams {
  file_name: string;
  file_size: number;
  mime_type: string;
  category: UploadCategory;
  md5?: string;
}

export const uploadService = {
  /**
   * 获取预签名上传 URL
   */
  async presign(params: PresignParams): Promise<ApiResponse<PresignUploadData>> {
    return post<PresignUploadData>(ENDPOINTS.MEDIA.PRESIGN, params, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 完成上传
   */
  async complete(uploadId: string): Promise<ApiResponse<UploadMediaData>> {
    return post<UploadMediaData>(ENDPOINTS.MEDIA.COMPLETE, { upload_id: uploadId }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 上传文件到预签名 URL
   */
  async uploadToPresignedUrl(
    presignedUrl: string,
    file: File,
    headers: Record<string, string>,
    onProgress?: (progress: number) => void
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      
      xhr.upload.addEventListener('progress', (event) => {
        if (event.lengthComputable && onProgress) {
          const progress = (event.loaded / event.total) * 100;
          onProgress(progress);
        }
      });

      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve();
        } else {
          reject(new Error(`Upload failed with status ${xhr.status}`));
        }
      });

      xhr.addEventListener('error', () => {
        reject(new Error('Upload failed'));
      });

      xhr.open('PUT', presignedUrl);
      
      Object.entries(headers).forEach(([key, value]) => {
        xhr.setRequestHeader(key, value);
      });

      xhr.send(file);
    });
  },

  /**
   * 完整的文件上传流程
   */
  async uploadFile(
    file: File,
    category: UploadCategory,
    onProgress?: (progress: number) => void
  ): Promise<UploadMediaData> {
    // 1. 获取预签名 URL
    const presignResponse = await this.presign({
      file_name: file.name,
      file_size: file.size,
      mime_type: file.type,
      category,
    });

    if (presignResponse.code !== 0 || !presignResponse.data) {
      throw new Error(presignResponse.message ?? 'Failed to get presigned URL');
    }

    const { upload_id, upload_url, headers } = presignResponse.data;

    // 2. 上传文件到预签名 URL
    await this.uploadToPresignedUrl(upload_url, file, headers, onProgress);

    // 3. 完成上传
    const completeResponse = await this.complete(upload_id);

    if (completeResponse.code !== 0 || !completeResponse.data) {
      throw new Error(completeResponse.message ?? 'Failed to complete upload');
    }

    return completeResponse.data;
  },
};

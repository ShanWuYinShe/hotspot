/**
 * 媒体上传服务
 */

import { post } from '../client';
import { MEDIA } from '../endpoints';
import type { MediaCategory, PresignedUpload, UploadedMedia } from '$lib/types/api';

export const mediaService = {
	/**
	 * 申请上传预签名 URL
	 */
	presignUpload(data: {
		category: MediaCategory;
		file_name: string;
		mime_type: string;
		file_size: number;
		md5?: string;
	}) {
		return post<PresignedUpload>(MEDIA.PRESIGN_UPLOAD, data);
	},

	/**
	 * 批量申请上传预签名 URL
	 */
	batchPresignUpload(data: {
		files: Array<{
			category: MediaCategory;
			file_name: string;
			mime_type: string;
			file_size: number;
			md5?: string;
		}>;
	}) {
		return post<{ uploads: PresignedUpload[] }>(MEDIA.BATCH_PRESIGN_UPLOAD, data);
	},

	/**
	 * 完成上传
	 */
	completeUpload(data: { upload_id: string }) {
		return post<UploadedMedia>(MEDIA.COMPLETE_UPLOAD, data, { idempotencyKey: crypto.randomUUID() });
	},

	/**
	 * 上传文件（封装完整流程）
	 */
	async uploadFile(
		file: File,
		category: MediaCategory,
		onProgress?: (progress: number) => void
	): Promise<UploadedMedia> {
		// 1. 获取预签名 URL
		const presignResponse = await this.presignUpload({
			category,
			file_name: file.name,
			mime_type: file.type,
			file_size: file.size
		});

		if (presignResponse.code !== 0 || !presignResponse.data) {
			throw new Error(presignResponse.message || '获取上传地址失败');
		}

		const { upload_id, upload_url, headers } = presignResponse.data;

		// 2. 上传文件到预签名 URL
		await new Promise<void>((resolve, reject) => {
			const xhr = new XMLHttpRequest();

			xhr.upload.addEventListener('progress', (e) => {
				if (e.lengthComputable && onProgress) {
					onProgress((e.loaded / e.total) * 100);
				}
			});

			xhr.addEventListener('load', () => {
				if (xhr.status >= 200 && xhr.status < 300) {
					resolve();
				} else {
					reject(new Error(`上传失败: ${xhr.status}`));
				}
			});

			xhr.addEventListener('error', () => {
				reject(new Error('上传失败'));
			});

			xhr.open('PUT', upload_url);
			Object.entries(headers || {}).forEach(([key, value]) => {
				xhr.setRequestHeader(key, value);
			});
			xhr.send(file);
		});

		// 3. 完成上传
		const completeResponse = await this.completeUpload({ upload_id });

		if (completeResponse.code !== 0 || !completeResponse.data) {
			throw new Error(completeResponse.message || '完成上传失败');
		}

		return completeResponse.data;
	}
};

// 媒体上传流程封装（spec: api_spec_media.md）。
// 三步：presign_upload 申请预签名 → PUT 直传对象存储 → complete_upload 换业务 file_id。
// 上传成功后返回 file_id，由业务 API（update_account / groups/create 等）消费。

import { presignUpload, completeUpload } from '@/api';
import type { MediaCategory } from '@/api/types';

/** 上传进度（0-1，单文件粒度）。 */
export type UploadProgress = (fraction: number) => void;

/**
 * PUT 直传（XMLHttpRequest：fetch 不暴露上传进度，大图慢网下用户无反馈）。
 * resolve 后不解析响应体——对象存储 PUT 成功即 200 空体/ETC 头，业务结果由
 * complete_upload 换 file_id 确认。
 */
function putWithProgress(url: string, headers: Record<string, string>, body: File, onProgress?: UploadProgress): Promise<void> {
	return new Promise((resolve, reject) => {
		const xhr = new XMLHttpRequest();
		xhr.open('PUT', url);
		for (const [k, v] of Object.entries(headers)) {
			xhr.setRequestHeader(k, v);
		}
		xhr.upload.onprogress = (e) => {
			if (e.lengthComputable && onProgress) onProgress(e.loaded / e.total);
		};
		xhr.onload = () => {
			if (xhr.status >= 200 && xhr.status < 300) resolve();
			else reject(new Error('上传失败，请重试'));
		};
		xhr.onerror = () => reject(new Error('上传失败，请检查网络'));
		xhr.ontimeout = () => reject(new Error('上传超时，请重试'));
		xhr.send(body);
	});
}

export async function uploadMediaFile(file: File, category: MediaCategory): Promise<string> {
	const presign = await presignUpload({
		files: [{
			file_name: file.name,
			file_size: file.size,
			mime_type: file.type || 'application/octet-stream',
			category
		}]
	});
	if (presign.code !== 0 || !presign.data) {
		throw new Error(presign.message ?? '上传申请失败');
	}

	const { upload_id, upload_url, headers } = presign.data.uploads[0];
	await putWithProgress(upload_url, headers, file);

	const complete = await completeUpload({ upload_ids: [upload_id] });
	if (complete.code !== 0 || !complete.data) {
		throw new Error(complete.message ?? '上传确认失败');
	}
	return complete.data.uploads[0].file_id;
}

/**
 * 批量上传（瞬间多图等场景）。流程：batch_presign → 并发 PUT 直传 → batch_complete。
 * spec 保证 uploads 与请求 files 一一对应、顺序一致，返回 file_id[] 与入参 files 顺序一致。
 */
export async function uploadMediaFiles(
	files: File[],
	category: MediaCategory,
	onProgress?: (fileIndex: number, fraction: number) => void
): Promise<string[]> {
	if (files.length === 0) return [];
	const presign = await presignUpload({
		files: files.map((f) => ({
			file_name: f.name,
			file_size: f.size,
			mime_type: f.type || 'application/octet-stream',
			category
		}))
	});
	if (presign.code !== 0 || !presign.data) {
		throw new Error(presign.message ?? '批量上传申请失败');
	}

	const uploads = presign.data.uploads;
	// 并发 PUT 直传（uploads[i] 对应 files[i]，spec 保证顺序一致）；进度按文件粒度上报
	await Promise.all(
		uploads.map((u, i) =>
			putWithProgress(u.upload_url, u.headers, files[i], onProgress ? (fraction) => onProgress(i, fraction) : undefined)
				.catch((err: unknown) => {
					throw err instanceof Error && err.message !== '上传失败，请重试' ? err : new Error(`第 ${i + 1} 张图片上传失败`);
				})
		)
	);

	const complete = await completeUpload({ upload_ids: uploads.map((u) => u.upload_id) });
	if (complete.code !== 0 || !complete.data) {
		throw new Error(complete.message ?? '批量上传确认失败');
	}
	return complete.data.uploads.map((u) => u.file_id);
}

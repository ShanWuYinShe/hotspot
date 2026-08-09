// 媒体上传流程封装（spec: api_spec_media.md）。
// 三步：presign_upload 申请预签名 → PUT 直传对象存储 → complete_upload 换业务 file_id。
// 上传成功后返回 file_id，由业务 API（update_account / groups/create 等）消费。

import { presignUpload, completeUpload } from '@/api';
import type { MediaCategory } from '@/api/types';

export async function uploadMediaFile(file: File, category: MediaCategory): Promise<string> {
	const presign = await presignUpload({
		file_name: file.name,
		file_size: file.size,
		mime_type: file.type || 'application/octet-stream',
		category
	});
	if (presign.code !== 0 || !presign.data) {
		throw new Error(presign.message ?? '上传申请失败');
	}

	const { upload_id, upload_url, headers } = presign.data;
	const putRes = await fetch(upload_url, {
		method: 'PUT',
		headers,
		body: file
	});
	if (!putRes.ok) {
		throw new Error('上传失败，请重试');
	}

	const complete = await completeUpload({ upload_id });
	if (complete.code !== 0 || !complete.data) {
		throw new Error(complete.message ?? '上传确认失败');
	}
	return complete.data.file_id;
}

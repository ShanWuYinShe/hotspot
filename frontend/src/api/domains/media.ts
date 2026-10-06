// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	PresignUploadRequest,
	PresignedUploads,
	CompletedUploads,
	CompleteUploadRequest,
} from '@/api/types';

// Media（spec: api_spec_media.md）—— presign → PUT 直传 → complete 三步流程的 API 包装
// presign_upload/complete_upload 均为数组语义，天然覆盖单/批，无独立批量端点。
export async function presignUpload(params: PresignUploadRequest) {
	return request<PresignedUploads>(Endpoints.Media.PresignUpload, params);
}
export async function completeUpload(params: CompleteUploadRequest) {
	return request<CompletedUploads>(Endpoints.Media.CompleteUpload, params);
}

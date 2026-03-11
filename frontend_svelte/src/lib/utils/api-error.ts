import type { ClientApiResponse } from '$lib/api/client';
import { ApiError, isSuccess } from '$lib/api/client';

/**
 * 将 ApiResponse 统一转换为错误对象（如果失败），便于上层按错误类型处理。
 * 成功时返回 null，不触发任何 UI。
 */
export function toApiError<T>(response: ClientApiResponse<T>): ApiError | null {
	return isSuccess(response) ? null : new ApiError(response);
}

/**
 * 统一的错误提示工具：优先使用业务 message，其次使用兜底文案。
 * 调用方可在外层根据 error.code 做分支（如鉴权失败、幂等冲突等）。
 */
export function showApiError(
	toast: { error: (msg: string) => void },
	response: ClientApiResponse<unknown>,
	fallbackMessage = '操作失败'
): void {
	const error = toApiError(response);
	if (!error) return;
	toast.error(error.message || fallbackMessage);
}

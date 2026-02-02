/**
 * API 调用 Hook
 * 
 * 统一管理 API 调用的加载状态、错误处理和成功回调
 * 使用 Svelte 5 runes 提供响应式状态
 */

import { isSuccess } from '$lib/api/client';
import { toast } from '$lib/stores/toast.svelte';
import type { ApiResponse } from '$lib/types/api';

export interface UseApiCallOptions<T> {
	/** 成功回调 */
	onSuccess?: (data: T) => void;
	/** 错误回调 */
	onError?: (error: ApiResponse<T>) => void;
	/** 是否显示成功提示（默认 false） */
	showSuccessToast?: boolean;
	/** 成功提示消息 */
	successMessage?: string;
	/** 是否显示错误提示（默认 true） */
	showErrorToast?: boolean;
	/** 自定义错误消息 */
	errorMessage?: string;
}

/**
 * API 调用 Hook
 * 
 * 使用示例：
 * ```typescript
 * const { call, loading, error, data } = useApiCall<UserProfile>();
 * 
 * async function handleFollow() {
 *   const result = await call(
 *     () => usersService.follow({ user_id: userId }),
 *     {
 *       onSuccess: (data) => {
 *         isFollowing = data.is_following;
 *       },
 *       successMessage: '关注成功'
 *     }
 *   );
 * }
 * ```
 */
export function useApiCall<T>() {
	let loading = $state(false);
	let error = $state<string | null>(null);
	let data = $state<T | null>(null);

	/**
	 * 执行 API 调用
	 * @returns 成功时返回数据，失败时返回 null
	 */
	async function call(
		apiCall: () => Promise<ApiResponse<T>>,
		options: UseApiCallOptions<T> = {}
	): Promise<T | null> {
		loading = true;
		error = null;

		try {
			const response = await apiCall();

			if (isSuccess(response) && response.data) {
				// 成功处理
				data = response.data;
				error = null;

				// 执行成功回调
				options.onSuccess?.(response.data);

				// 显示成功提示
				if (options.showSuccessToast && options.successMessage) {
					toast.success(options.successMessage);
				}

				return response.data;
			} else {
				// 失败处理
				const errorMsg =
					options.errorMessage || response.message || '操作失败';
				error = errorMsg;
				data = null;

				// 执行错误回调
				options.onError?.(response);

				// 显示错误提示
				if (options.showErrorToast !== false) {
					toast.error(errorMsg);
				}

				return null;
			}
		} catch (err) {
			// 异常处理
			const errorMsg =
				options.errorMessage ||
				(err instanceof Error ? err.message : '操作失败，请稍后重试');
			error = errorMsg;
			data = null;

			// 显示错误提示
			if (options.showErrorToast !== false) {
				toast.error(errorMsg);
			}

			return null;
		} finally {
			loading = false;
		}
	}

	return {
		call,
		loading,
		error,
		data
	};
}

/**
 * 简化版 API 调用 Hook（仅加载状态）
 * 
 * 使用示例：
 * ```typescript
 * const { execute, loading } = useSimpleApiCall();
 * 
 * const handleSubmit = execute(async () => {
 *   const response = await service.submit(data);
 *   if (isSuccess(response)) {
 *     toast.success('提交成功');
 *     close();
 *   } else {
 *     toast.error(response.message);
 *   }
 * });
 * ```
 */
export function useSimpleApiCall() {
	let loading = $state(false);

	function execute<T extends unknown[]>(
		action: (...args: T) => Promise<void>
	): (...args: T) => Promise<void> {
		return async (...args: T) => {
			loading = true;
			try {
				await action(...args);
			} finally {
				loading = false;
			}
		};
	}

	return {
		execute,
		loading
	};
}

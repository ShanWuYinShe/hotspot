/**
 * 加载动画最短展示时间工具
 *
 * 业界最佳实践（Firefox Design System / Stanford Research 2025 / NNG）：
 * - 加载动画一旦显示，最少展示 400ms，避免"一闪而过"
 * - 中等速度动画比极快动画更能降低用户感知等待时间
 * - 即使操作出错，也应保持最短展示时间，让用户感知到"系统在工作"
 */

/** 全屏加载默认最短展示时间（ms） */
export const MIN_LOADING_FULL_SCREEN = 500;

/** 操作类加载默认最短展示时间（ms） */
export const MIN_LOADING_ACTION = 400;

/**
 * 执行异步任务并确保至少经过 minimumMs 毫秒后才返回/抛出。
 *
 * @example
 * ```ts
 * isLoading = true;
 * try {
 *   const response = await withMinimumLoadingTime(() => api.fetchData());
 *   items = response.data.items;
 * } catch (e) {
 *   toast.error('加载失败');
 * } finally {
 *   isLoading = false;
 * }
 * ```
 */
export async function withMinimumLoadingTime<T>(
	task: () => Promise<T>,
	minimumMs: number = MIN_LOADING_FULL_SCREEN
): Promise<T> {
	const timer = new Promise<void>((resolve) => setTimeout(resolve, minimumMs));

	try {
		const result = await task();
		await timer;
		return result;
	} catch (error) {
		await timer;
		throw error;
	}
}

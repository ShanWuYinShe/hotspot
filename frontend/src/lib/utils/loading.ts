/**
 * 加载动画最短展示时间工具
 */

export const MIN_LOADING_FULL_SCREEN = 500;
export const MIN_LOADING_ACTION = 400;

export async function withMinimumLoadingTime<T>(
  task: () => Promise<T>,
  minimumMs: number = MIN_LOADING_FULL_SCREEN,
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

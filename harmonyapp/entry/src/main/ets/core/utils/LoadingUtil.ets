/**
 * 加载动画最短展示时间工具
 *
 * 业界最佳实践：
 * - 加载动画一旦显示，最少展示 400ms，避免"一闪而过"
 * - 即使操作出错，也应保持最短展示时间，让用户感知到"系统在工作"
 */
export class LoadingUtil {
  /** 全屏加载默认最短展示时间（ms） */
  static readonly DEFAULT_FULL_SCREEN_MS: number = 500;

  /** 操作类加载默认最短展示时间（ms） */
  static readonly DEFAULT_ACTION_MS: number = 400;

  /**
   * 执行异步任务并确保至少经过 minimumMs 毫秒后才返回/抛出。
   *
   * @param task 需要执行的异步任务
   * @param minimumMs 最短展示时间，默认 500ms
   * @returns 任务返回值
   */
  static async withMinimumTime<T>(
    task: () => Promise<T>,
    minimumMs: number = LoadingUtil.DEFAULT_FULL_SCREEN_MS
  ): Promise<T> {
    const timerPromise = new Promise<void>((resolve) => setTimeout(resolve, minimumMs));

    let result: T | undefined = undefined;
    let caughtError: Error | null = null;

    try {
      result = await task();
    } catch (error) {
      caughtError = error instanceof Error ? error : new Error(String(error));
    }

    await timerPromise;

    if (caughtError !== null) {
      throw caughtError;
    }
    return result!;
  }
}

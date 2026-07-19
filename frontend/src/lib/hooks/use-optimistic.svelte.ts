import { toast } from '$lib/stores/toast.svelte';
import { isSuccess, type ApiResponse } from '$lib/api/client';
import { t } from '$lib/i18n';

interface OptimisticOptions<T> {
  /** Initial value (can be a getter for reactivity) */
  initialValue: boolean | (() => boolean);
  /** Initial count (optional, can be a getter) */
  initialCount?: number | (() => number);
  /** API action to perform when toggling to true */
  onTrue: () => Promise<ApiResponse<T>>;
  /** API action to perform when toggling to false */
  onFalse: () => Promise<ApiResponse<T>>;
  /** Error message to show on failure */
  errorMessage?: string;
}

/**
 * 乐观开关 hook。
 *
 * 设计:外部值 (`initialValue` / `initialCount` getter) 通过 `$derived` 派生,
 * 再叠加一个 pending 的乐观覆盖 `$state`。
 *
 * - 外部值变化时(包括引用变化但值相同的场景),派生自动重算。
 * - pending override 在 toggle 中设置;**成功后保留**(因为父组件未必立即刷新外部数据,
 *   保留可避免回退到陈旧外部值导致的按钮闪烁)。
 * - 当外部值真正发生变化(父组件刷新了数据)时,$effect.pre 清掉 pending,
 *   让派生接管 —— 此时外部值已是最新的,无闪烁。
 * - 失败时清掉 pending,回退到外部值(外部值未变,等价于回滚到原始状态)。
 * - 避免了旧实现中"外部值引用变化但值相同 → 不同步 → 乐观回滚后状态错位"的问题,
 *   也修复了"乐观回滚后父组件以同一引用重渲染 → sync 漏触发"的 bug。
 */
export function useOptimisticToggle<T>(options: OptimisticOptions<T>) {
  const getExternalValue = () =>
    typeof options.initialValue === 'function' ? options.initialValue() : options.initialValue;

  const getExternalCount = () =>
    options.initialCount !== undefined
      ? typeof options.initialCount === 'function'
        ? options.initialCount()
        : options.initialCount
      : 0;

  // 派生的外部值 —— 任何依赖项变化都会重算
  const externalValue = $derived(getExternalValue());
  const externalCount = $derived(getExternalCount());

  // pending 的乐观覆盖:null 表示回落到外部值
  let pendingValue = $state<boolean | null>(null);
  let pendingCount = $state<number | null>(null);
  let isLoading = $state(false);
  let alive = true;

  $effect(() => {
    return () => {
      alive = false;
    };
  });

  // 外部值真正变化时,清掉 pending override,让派生接管。
  // 用 prev-tracking 检测"值变化"(而非引用变化)—— 因为派生已保证引用变化时重算,
  // 这里只需在值实际不同时清 pending,避免清掉刚写入的乐观值。
  let prevExternalValue = externalValue;
  let prevExternalCount = externalCount;
  $effect.pre(() => {
    const ev = externalValue;
    const ec = externalCount;
    if (ev !== prevExternalValue || ec !== prevExternalCount) {
      prevExternalValue = ev;
      prevExternalCount = ec;
      if (pendingValue !== null) pendingValue = null;
      if (pendingCount !== null) pendingCount = null;
    }
  });

  const isOn = $derived(pendingValue !== null ? pendingValue : externalValue);
  const count = $derived(pendingCount !== null ? pendingCount : externalCount);

  async function toggle() {
    if (isLoading) return;

    const previousIsOn = isOn;
    const previousCount = count;
    const nextIsOn = !previousIsOn;

    // Optimistic update —— 写入 pending 层
    pendingValue = nextIsOn;
    if (options.initialCount !== undefined) {
      pendingCount = previousCount + (nextIsOn ? 1 : -1);
    }

    isLoading = true;

    try {
      const action = nextIsOn ? options.onTrue : options.onFalse;
      const response = await action();

      if (!isSuccess(response)) {
        throw new Error(response.message || 'Operation failed');
      }
      // 成功:保留 pending(父组件未必立即刷新外部数据;待外部值真正变化时由
      // 上面的 $effect.pre 清除,自然过渡,无闪烁)。
    } catch (error) {
      if (!alive) return;
      // Rollback: 清掉 pending,回退到外部值(失败时外部值未变,等价于回滚到原始状态)
      pendingValue = null;
      pendingCount = null;
      toast.error(options.errorMessage || (error instanceof Error ? error.message : t('errors.operationFailed')));
    } finally {
      if (alive) isLoading = false;
    }
  }

  return {
    get isOn() {
      return isOn;
    },
    get count() {
      return count;
    },
    get isLoading() {
      return isLoading;
    },
    toggle
  };
}

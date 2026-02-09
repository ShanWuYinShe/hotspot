import { toast } from '$lib/stores/toast.svelte';
import { isSuccess, type ApiResponse } from '$lib/api/client';

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

export function useOptimisticToggle<T>(options: OptimisticOptions<T>) {
	// Initialize state directly (no need for $effect)
	const getInitialValue = () => 
		typeof options.initialValue === 'function' 
			? options.initialValue() 
			: options.initialValue;
	
	const getInitialCount = () => 
		options.initialCount !== undefined
			? (typeof options.initialCount === 'function'
				? options.initialCount()
				: options.initialCount)
			: 0;
	
	let isOn = $state(getInitialValue());
	let count = $state(getInitialCount());
	let isLoading = $state(false);

	async function toggle() {
		if (isLoading) return;

		const previousIsOn = isOn;
		const previousCount = count;

		// Optimistic update
		isOn = !previousIsOn;
		if (options.initialCount !== undefined) {
			count += isOn ? 1 : -1;
		}

		isLoading = true;

		try {
			const action = isOn ? options.onTrue : options.onFalse;
			const response = await action();

			if (!isSuccess(response)) {
				throw new Error(response.message || 'Operation failed');
			}
		} catch (error) {
			// Rollback
			isOn = previousIsOn;
			count = previousCount;
			toast.error(options.errorMessage || (error instanceof Error ? error.message : '操作失败'));
		} finally {
			isLoading = false;
		}
	}

	return {
		get isOn() { return isOn; },
		get count() { return count; },
		get isLoading() { return isLoading; },
		toggle
	};
}

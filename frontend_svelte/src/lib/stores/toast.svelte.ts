/**
 * Toast 消息状态
 */

import { toast as sonnerToast } from 'svelte-sonner';

export const toast = {
	success(message: string, description?: string) {
		sonnerToast.success(message, { description });
	},

	error(message: string, description?: string) {
		sonnerToast.error(message, { description });
	},

	warning(message: string, description?: string) {
		sonnerToast.warning(message, { description });
	},

	info(message: string, description?: string) {
		sonnerToast.info(message, { description });
	},

	loading(message: string) {
		return sonnerToast.loading(message);
	},

	dismiss(id?: string | number) {
		sonnerToast.dismiss(id);
	},

	promise<T>(
		promise: Promise<T>,
		options: {
			loading: string;
			success: string | ((data: T) => string);
			error: string | ((error: unknown) => string);
		}
	) {
		return sonnerToast.promise(promise, options);
	}
};

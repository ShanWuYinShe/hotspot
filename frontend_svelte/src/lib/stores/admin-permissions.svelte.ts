import { browser } from '$app/environment';
import { adminRbacService } from '$lib/api/services';
import { isSuccess } from '$lib/api/client';
import { authStore } from '$lib/stores/auth.svelte';
import type { AdminPermissions } from '$lib/types/api';

type AdminPermissionStatus = 'idle' | 'loading' | 'ready' | 'forbidden' | 'error';

class AdminPermissionsStore {
	#status = $state<AdminPermissionStatus>('idle');
	#permissions = $state<AdminPermissions | null>(null);
	#loadedUserId = $state<string | null>(null);
	#requestVersion = 0;
	#destroyEffectRoot: (() => void) | null = null;

	constructor() {
		if (!browser) return;

		this.#destroyEffectRoot = $effect.root(() => {
			$effect(() => {
				const isAuthenticated = authStore.isAuthenticated;
				const currentUserId = authStore.currentUserId;
				if (!isAuthenticated || !currentUserId) {
					this.reset();
					return;
				}

				if (this.#loadedUserId === currentUserId || this.#status === 'loading') {
					return;
				}

				void this.ensureLoaded(currentUserId);
			});
		});
	}

	status = $derived(this.#status);
	permissions = $derived(this.#permissions);
	loadedUserId = $derived(this.#loadedUserId);
	isLoading = $derived(this.#status === 'loading' || this.#status === 'idle');
	isAdmin = $derived(this.#status === 'ready' && !!this.#permissions);
	isForbidden = $derived(this.#status === 'forbidden');
	isError = $derived(this.#status === 'error');

	async ensureLoaded(userId = authStore.currentUserId): Promise<void> {
		if (!browser || !authStore.isAuthenticated || !userId) {
			this.reset();
			return;
		}
		if (this.#loadedUserId === userId || this.#status === 'loading') {
			return;
		}

		const version = ++this.#requestVersion;
		this.#status = 'loading';
		this.#permissions = null;

		try {
			const res = await adminRbacService.getMyPermissions();
			if (version !== this.#requestVersion || authStore.currentUserId !== userId || !authStore.isAuthenticated) {
				return;
			}

			this.#loadedUserId = userId;
			if (isSuccess(res) && res.data) {
				this.#permissions = res.data;
				this.#status = 'ready';
			} else {
				this.#status = 'forbidden';
			}
		} catch {
			if (version !== this.#requestVersion || authStore.currentUserId !== userId || !authStore.isAuthenticated) {
				return;
			}
			this.#loadedUserId = userId;
			this.#permissions = null;
			this.#status = 'error';
		}
	}

	refresh() {
		if (!browser) return;
		this.#loadedUserId = null;
		this.#permissions = null;
		if (authStore.isAuthenticated && authStore.currentUserId) {
			void this.ensureLoaded(authStore.currentUserId);
			return;
		}
		this.#status = 'idle';
	}

	reset() {
		this.#requestVersion += 1;
		this.#status = 'idle';
		this.#permissions = null;
		this.#loadedUserId = null;
	}

	destroy() {
		this.#destroyEffectRoot?.();
		this.#destroyEffectRoot = null;
	}
}

export const adminPermissionsStore = new AdminPermissionsStore();

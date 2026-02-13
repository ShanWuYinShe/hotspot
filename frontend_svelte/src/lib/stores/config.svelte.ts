/**
 * 客户端配置状态管理 (Svelte 5 Runes)
 *
 * 启动时从服务端拉取 ClientConfig 并缓存。
 * 提供 features / limits / contact / defaults 的便捷访问。
 */

import { browser } from '$app/environment';
import type { ClientConfig } from '$lib/types/api';
import { systemService } from '$lib/api/services/system';

class ConfigStore {
	#config = $state<ClientConfig | null>(null);
	#loaded = $state(false);
	#error = $state<string | null>(null);

	// ==================== Derived State ====================

	/** 完整配置对象 */
	config = $derived(this.#config);

	/** 是否已加载 */
	loaded = $derived(this.#loaded);

	/** 加载错误信息 */
	error = $derived(this.#error);

	/** 功能开关 */
	features = $derived(this.#config?.features ?? null);

	/** 应用限制 */
	limits = $derived(this.#config?.limits ?? null);

	/** 联系信息 */
	contact = $derived(this.#config?.contact ?? null);

	/** 默认值 */
	defaults = $derived(this.#config?.defaults ?? null);

	/** 默认位置 */
	defaultLocation = $derived(this.#config?.defaults?.default_location ?? null);

	// ==================== Actions ====================

	/**
	 * 从服务端拉取 ClientConfig 并缓存
	 */
	async fetch(): Promise<void> {
		if (!browser) return;

		try {
			const res = await systemService.getClientConfig();
			if (res.code === 0 && res.data) {
				this.#config = res.data;
				this.#error = null;
			} else {
				this.#error = res.message || '获取配置失败';
			}
		} catch (e) {
			this.#error = e instanceof Error ? e.message : '获取配置失败';
		} finally {
			this.#loaded = true;
		}
	}

	/**
	 * 获取搜索半径上限（米），回退到 50000
	 */
	get searchRadiusMax(): number {
		return this.#config?.limits?.search_radius_max ?? 50000;
	}

	/**
	 * 获取瞬间最大图片数
	 */
	get maxMomentImages(): number {
		return this.#config?.limits?.max_moment_images ?? 9;
	}

	/**
	 * 获取瞬间最大标签数
	 */
	get maxMomentTags(): number {
		return this.#config?.limits?.max_moment_tags ?? 10;
	}

	/**
	 * 获取瞬间描述最大长度
	 */
	get momentDescriptionMaxLength(): number {
		return this.#config?.limits?.moment_description_max_length ?? 2000;
	}

	/**
	 * 获取评论最大长度
	 */
	get commentMaxLength(): number {
		return this.#config?.limits?.comment_max_length ?? 500;
	}
}

export const configStore = new ConfigStore();

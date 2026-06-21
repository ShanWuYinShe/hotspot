/**
 * 客户端配置状态管理 (Svelte 5 Runes)
 *
 * 启动时从服务端拉取 ClientConfig 并缓存。
 * 提供 features / limits / contact / defaults 的便捷访问。
 */

import { browser } from '$app/environment';
import type { ClientConfig } from '$lib/types/api';
import { isSuccess } from '$lib/api/client';
import { systemService } from '$lib/api/services/system';

class ConfigStore {
  #config = $state.raw<ClientConfig | null>(null);
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
      if (isSuccess(res) && res.data) {
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

  /** 搜索半径上限（米）默认使用 50000（当服务端未配置时） */
  searchRadiusMax = $derived(this.#config?.limits?.search_radius_max ?? 50000);

  /** 瞬间最大图片数 */
  maxMomentImages = $derived(this.#config?.limits?.max_moment_images ?? 9);

  /** 瞬间最大标签数 */
  maxMomentTags = $derived(this.#config?.limits?.max_moment_tags ?? 10);

  /** 瞬间描述最大长度 */
  momentDescriptionMaxLength = $derived(
    this.#config?.limits?.moment_description_max_length ?? 2000
  );

  /** 评论最大长度 */
  commentMaxLength = $derived(this.#config?.limits?.comment_max_length ?? 500);
}

export const configStore = new ConfigStore();

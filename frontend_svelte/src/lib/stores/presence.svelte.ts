/**
 * 用户在线状态管理 (Svelte 5 Runes)
 * 
 * 管理用户的在线状态（online/idle/dnd/invisible/offline）
 * 通过 WebSocket 事件实时更新
 */

import type { UserPresenceStatus } from '$lib/websocket/types';

interface UserPresence {
	user_id: number;
	status: UserPresenceStatus;
	status_text?: string;
	updated_at: string;
	last_seen?: string; // 离线时的最后在线时间
}

class PresenceStore {
	// 用户状态映射：user_id -> UserPresence
	#presences = $state<Map<number, UserPresence>>(new Map());

	// ==================== Getters ====================

	/**
	 * 获取用户状态
	 * @param user_id 用户ID
	 * @returns 用户状态，如果不存在则返回 null
	 */
	getPresence(user_id: number): UserPresence | null {
		return this.#presences.get(user_id) ?? null;
	}

	/**
	 * 获取用户状态类型
	 * @param user_id 用户ID
	 * @returns 状态类型，如果不存在则返回 'offline'
	 */
	getStatus(user_id: number): UserPresenceStatus {
		return this.#presences.get(user_id)?.status ?? 'offline';
	}

	/**
	 * 获取用户自定义状态文本
	 * @param user_id 用户ID
	 * @returns 状态文本，如果不存在则返回 undefined
	 */
	getStatusText(user_id: number): string | undefined {
		return this.#presences.get(user_id)?.status_text;
	}

	/**
	 * 检查用户是否在线（online 或 idle）
	 * @param user_id 用户ID
	 * @returns 是否在线
	 */
	isOnline(user_id: number): boolean {
		const status = this.getStatus(user_id);
		return status === 'online' || status === 'idle';
	}

	/**
	 * 检查用户是否离线
	 * @param user_id 用户ID
	 * @returns 是否离线
	 */
	isOffline(user_id: number): boolean {
		const status = this.getStatus(user_id);
		return status === 'offline';
	}

	/**
	 * 获取用户最后在线时间
	 * @param user_id 用户ID
	 * @returns 最后在线时间字符串，如果不存在则返回 undefined
	 */
	getLastSeen(user_id: number): string | undefined {
		return this.#presences.get(user_id)?.last_seen;
	}

	/**
	 * 批量获取多个用户的状态
	 * @param user_ids 用户ID数组
	 * @returns 用户ID到状态的映射
	 */
	getBatchStatuses(user_ids: number[]): Map<number, UserPresenceStatus> {
		const result = new Map<number, UserPresenceStatus>();
		for (const user_id of user_ids) {
			result.set(user_id, this.getStatus(user_id));
		}
		return result;
	}

	/**
	 * 获取所有已知状态的用户ID
	 * @returns 用户ID数组
	 */
	getAllUserIds(): number[] {
		return Array.from(this.#presences.keys());
	}

	// ==================== Actions ====================

	/**
	 * 更新用户状态（用于 UserPresenceUpdate 事件）
	 * @param user_id 用户ID
	 * @param status 新状态
	 * @param status_text 自定义状态文本（可选）
	 * @param updated_at 更新时间
	 */
	updatePresence(
		user_id: number,
		status: UserPresenceStatus,
		status_text: string | undefined,
		updated_at: string
	) {
		const existing = this.#presences.get(user_id);
		this.#presences.set(user_id, {
			user_id,
			status,
			status_text,
			updated_at,
			last_seen: existing?.last_seen
		});
		// 触发响应式更新
		this.#presences = this.#presences;
	}

	/**
	 * 设置用户在线（用于 UserOnline 事件）
	 * @param user_id 用户ID
	 * @param status 状态类型
	 * @param status_text 自定义状态文本（可选）
	 * @param online_at 上线时间
	 */
	setOnline(
		user_id: number,
		status: UserPresenceStatus,
		status_text: string | undefined,
		online_at: string
	) {
		const existing = this.#presences.get(user_id);
		this.#presences.set(user_id, {
			user_id,
			status,
			status_text,
			updated_at: online_at,
			last_seen: existing?.last_seen
		});
		// 触发响应式更新
		this.#presences = this.#presences;
	}

	/**
	 * 设置用户离线（用于 UserOffline 事件）
	 * @param user_id 用户ID
	 * @param offline_at 离线时间
	 * @param last_seen 最后在线时间（可选）
	 */
	setOffline(user_id: number, offline_at: string, last_seen?: string) {
		const existing = this.#presences.get(user_id);
		this.#presences.set(user_id, {
			user_id,
			status: 'offline',
			updated_at: offline_at,
			last_seen: last_seen ?? existing?.last_seen ?? offline_at
		});
		// 触发响应式更新
		this.#presences = this.#presences;
	}

	/**
	 * 清除用户状态（用户登出或不再需要跟踪时）
	 * @param user_id 用户ID
	 */
	clearPresence(user_id: number) {
		this.#presences.delete(user_id);
		// 触发响应式更新
		this.#presences = this.#presences;
	}

	/**
	 * 批量清除用户状态
	 * @param user_ids 用户ID数组
	 */
	clearBatchPresences(user_ids: number[]) {
		for (const user_id of user_ids) {
			this.#presences.delete(user_id);
		}
		// 触发响应式更新
		this.#presences = this.#presences;
	}

	/**
	 * 清除所有状态
	 */
	clearAll() {
		this.#presences.clear();
		// 触发响应式更新
		this.#presences = this.#presences;
	}
}

// 导出单例
export const presenceStore = new PresenceStore();

/**
 * 用户在线状态管理 (Svelte 5 Runes)
 *
 * 通过 WebSocket 事件 (UserOnline/UserOffline) 实时更新。
 * spec 仅定义 online/offline 两种状态，前端扩展了 idle/dnd/invisible 供未来使用。
 */

import type { UserPresenceStatus } from '$lib/types/api';

// 前端扩展的在线状态（spec 仅 online/offline，此处预留更细粒度）
export type ExtendedPresenceStatus = UserPresenceStatus | 'idle' | 'dnd' | 'invisible';

interface UserPresence {
	user_id: string;
	status: ExtendedPresenceStatus;
	status_text?: string;
	updated_at: string;
	last_seen?: string; // 离线时的最后在线时间
}

class PresenceStore {
	#presences = $state.raw<Map<string, UserPresence>>(new Map());

	// ============ 内部工具 ============

	/** 不可变更新 Map：clone → mutate → reassign */
	#updateMap(mutator: (map: Map<string, UserPresence>) => boolean) {
		const next = new Map(this.#presences);
		const changed = mutator(next);
		if (!changed) return;
		this.#presences = next;
	}

	// ============ 查询 ============

	getPresence(user_id: string): UserPresence | null {
		return this.#presences.get(user_id) ?? null;
	}

	getStatus(user_id: string): ExtendedPresenceStatus {
		return this.#presences.get(user_id)?.status ?? 'offline';
	}

	getStatusText(user_id: string): string | undefined {
		return this.#presences.get(user_id)?.status_text;
	}

	isOnline(user_id: string): boolean {
		const status = this.getStatus(user_id);
		return status === 'online' || status === 'idle';
	}

	isOffline(user_id: string): boolean {
		return this.getStatus(user_id) === 'offline';
	}

	getLastSeen(user_id: string): string | undefined {
		return this.#presences.get(user_id)?.last_seen;
	}

	getBatchStatuses(user_ids: string[]): Map<string, ExtendedPresenceStatus> {
		const result = new Map<string, ExtendedPresenceStatus>();
		for (const id of user_ids) {
			result.set(id, this.getStatus(id));
		}
		return result;
	}

	getAllUserIds(): string[] {
		return Array.from(this.#presences.keys());
	}

	// ============ 状态更新 ============

	/** 设置用户在线（WebSocket UserOnline 事件） */
	setOnline(
		user_id: string,
		status: ExtendedPresenceStatus,
		status_text: string | undefined,
		online_at: string
	) {
		this.#updateMap((map) => {
			const existing = map.get(user_id);
			if (
				existing?.status === status &&
				existing?.status_text === status_text &&
				existing?.updated_at === online_at
			) {
				return false;
			}
			map.set(user_id, {
				user_id,
				status,
				status_text,
				updated_at: online_at,
				last_seen: existing?.last_seen
			});
			return true;
		});
	}

	/** 设置用户离线（WebSocket UserOffline 事件） */
	setOffline(user_id: string, offline_at: string, last_seen?: string) {
		this.#updateMap((map) => {
			const existing = map.get(user_id);
			const nextLastSeen = last_seen ?? existing?.last_seen ?? offline_at;
			if (
				existing?.status === 'offline' &&
				existing?.updated_at === offline_at &&
				existing?.last_seen === nextLastSeen
			) {
				return false;
			}
			map.set(user_id, {
				user_id,
				status: 'offline',
				updated_at: offline_at,
				last_seen: nextLastSeen
			});
			return true;
		});
	}

	/** 清除单个用户状态 */
	clearPresence(user_id: string) {
		this.#updateMap((map) => map.delete(user_id));
	}

	/** 批量清除用户状态 */
	clearBatchPresences(user_ids: string[]) {
		this.#updateMap((map) => {
			let changed = false;
			for (const id of user_ids) {
				changed = map.delete(id) || changed;
			}
			return changed;
		});
	}

	/** 清除所有状态 */
	clearAll() {
		this.#presences = new Map();
	}
}

export const presenceStore = new PresenceStore();

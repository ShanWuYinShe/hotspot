// API 公共面契约测试（refactor/api-client-split 回归护栏）。
//
// client.ts 用 `export * from './core' + './domains/*'` 聚合 15 个模块，而 TS/ESM 对
// 「两个星号导出同名符号」是**静默剔除**（不报错），调用方只会拿到 undefined；
// 显式导出又会遮蔽星号导出、形成同名双实现（netError 曾同时存在于 core.ts 与
// client.ts，两份可静默漂移）。本文件把这两条不变量锁进测试。
import { describe, expect, it } from 'vitest';
import * as api from './index';
import * as core from './core';
import * as admin from './domains/admin';
import * as auth from './domains/auth';
import * as chat from './domains/chat';
import * as comments from './domains/comments';
import * as groups from './domains/groups';
import * as map from './domains/map';
import * as media from './domains/media';
import * as moments from './domains/moments';
import * as notify from './domains/notify';
import * as reports from './domains/reports';
import * as search from './domains/search';
import * as share from './domains/share';
import * as system from './domains/system';
import * as users from './domains/users';

const modules: Record<string, Record<string, unknown>> = {
	core,
	admin,
	auth,
	chat,
	comments,
	groups,
	map,
	media,
	moments,
	notify,
	reports,
	search,
	share,
	system,
	users,
};

function fnNames(mod: Record<string, unknown>): string[] {
	return Object.keys(mod).filter((key) => typeof mod[key] === 'function');
}

describe('API 公共面（facade 聚合不变量）', () => {
	it('每个模块的运行时导出都能在 @/api 上取到，且引用同一个实现', () => {
		const missing: string[] = [];
		const mismatched: string[] = [];
		for (const [modName, mod] of Object.entries(modules)) {
			for (const fn of fnNames(mod)) {
				const exported = (api as Record<string, unknown>)[fn];
				if (exported === undefined) missing.push(`${modName}.${fn}`);
				else if (exported !== mod[fn]) mismatched.push(`${modName}.${fn}`);
			}
		}
		expect({ missing, mismatched }).toEqual({ missing: [], mismatched: [] });
	});

	it('模块之间没有重名运行时导出（重名会被 export * 静默剔除）', () => {
		const owners = new Map<string, string>();
		const duplicates: string[] = [];
		for (const [modName, mod] of Object.entries(modules)) {
			for (const fn of fnNames(mod)) {
				const prev = owners.get(fn);
				if (prev !== undefined) duplicates.push(`${fn}: ${prev} 与 ${modName}`);
				else owners.set(fn, modName);
			}
		}
		expect(duplicates).toEqual([]);
	});

	it('传输核心与端点常量经 @/api 暴露', () => {
		expect(typeof api.request).toBe('function');
		expect(typeof api.netError).toBe('function');
		expect(typeof api.setApiBaseUrl).toBe('function');
		expect(typeof api.newIdempotencyKey).toBe('function');
		expect(api.Endpoints).toBeTruthy();
	});
});

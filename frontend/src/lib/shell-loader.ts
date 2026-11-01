// 四端外壳的按平台动态加载（唯一加载入口）。
//
// 为什么动态 import：MainLayout 静态 import 四套端壳会把当前端之外的端壳
// （以及其独有组件）一并打进首屏依赖图；动态化后只下载当前端。
//
// 为什么要在这里集中：根布局需要在启动阶段（BootSplash 等待 auth/session 时）
// 就预取当前端外壳 chunk，与 auth 请求并行——否则外壳 chunk 要等 BootSplash
// 结束、MainLayout 挂载后才开始下载，慢网下等于凭空多一个串行 RTT。
// MainLayout 与根布局共用同一份 loader，模块级 promise 去重保证只请求一次。

import type { Component, Snippet } from 'svelte';
import type { Orientation, Platform } from '@/stores/platform.svelte';

export type ShellKind = 'desktop' | 'tablet' | 'landscape' | 'portrait';

export type ShellComponent = Component<{ children: Snippet }>;

// 每个端一个 loader：动态 import 用字面量，打包器才能把四端壳切成独立 chunk。
const SHELL_LOADERS: Record<ShellKind, () => Promise<{ default: ShellComponent }>> = {
	desktop: () => import('@/components/layout/desktop/DesktopLayout.svelte'),
	tablet: () => import('@/components/layout/tablet/TabletLayout.svelte'),
	landscape: () => import('@/components/layout/mobile/MobileLandscape.svelte'),
	portrait: () => import('@/components/layout/mobile/MobilePortrait.svelte')
};

/** 当前设备对应的端壳（与 MainLayout 分派规则同口径）。 */
export function shellKindFor(devicePlatform: Platform, orientation: Orientation): ShellKind {
	if (devicePlatform === 'desktop') return 'desktop';
	if (devicePlatform === 'tablet') return 'tablet';
	return orientation === 'landscape' ? 'landscape' : 'portrait';
}

const pending = new Map<ShellKind, Promise<{ default: ShellComponent }>>();

/** 加载（或复用）指定端外壳模块。失败不吞错：调用方决定重试/占位策略。 */
export function loadShell(kind: ShellKind): Promise<{ default: ShellComponent }> {
	let p = pending.get(kind);
	if (!p) {
		p = SHELL_LOADERS[kind]();
		pending.set(kind, p);
		// 失败后清掉缓存，允许下次调用重试；成功的 promise 保留复用。
		void p.catch(() => pending.delete(kind));
	}
	return p;
}

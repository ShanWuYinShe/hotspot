<script lang="ts">
	import type { Snippet } from 'svelte';
	import { platform } from '@/stores/platform.svelte';
	import { loadShell, shellKindFor, type ShellComponent } from '@/lib/shell-loader';
	import BootSplash from '@/components/shared/BootSplash.svelte';

	// MainLayout — 四端壳分派：桌面 → DesktopLayout，平板 → TabletLayout，
	// 移动横屏 → MobileLandscape，移动竖屏 → MobilePortrait。
	//
	// 四端壳一律按平台动态 import（loader 见 lib/shell-loader）：只加载当前端，
	// 其余端壳不进入首屏依赖图；根布局会在 BootSplash 等待 auth 期间预取当前端
	// chunk，所以这里通常直接命中已下载模块。外壳就绪前用与启动占位同款的
	// BootSplash 占位，避免空画布 / 布局闪烁。

	let { children }: { children: Snippet } = $props();

	const shellKind = $derived(shellKindFor(platform.platform, platform.orientation));

	// 当前已解析的外壳组件；loadedKind 用于「同一端不重复加载」（仅横竖屏切换时换壳）。
	let Shell = $state<ShellComponent | null>(null);
	let loadedKind = $state<string | null>(null);
	// 外壳 chunk 加载失败（弱网 / 部署滚动窗口）时重试一次，避免永久停在占位。
	let loadAttempt = $state(0);

	$effect(() => {
		const kind = shellKind;
		void loadAttempt; // 重试信号：变化时重新执行本 effect
		if (loadedKind === kind) return;
		let cancelled = false;
		void loadShell(kind).then(
			(mod) => {
				if (cancelled) return;
				Shell = mod.default;
				loadedKind = kind;
			},
			() => {
				if (cancelled) return;
				setTimeout(() => {
					if (!cancelled) loadAttempt += 1;
				}, 1000);
			}
		);
		return () => {
			cancelled = true;
		};
	});
</script>

{#if Shell}
	<Shell>{@render children()}</Shell>
{:else}
	<!-- 外壳 chunk 下载期间：与根布局启动占位同款（BootSplash 已在根布局静态依赖内，零额外体积） -->
	<div class="fixed inset-0 isolate grid place-items-center overflow-hidden bg-surface-page">
		<BootSplash />
	</div>
{/if}

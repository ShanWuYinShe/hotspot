<script lang="ts">
	import { goto } from '$app/navigation';
	import { CheckCircle2, ChevronRight, Circle, ShieldCheck } from '@lucide/svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import { cn } from '@/lib/utils';

	// 资料状态卡（个人中心桌面/移动同构）：完整度 + 建议补全项。
	// 补全项直接给可点入口（编辑资料 / 附近群组），而不是只画一根进度条让用户自己猜
	// 「差在哪、去哪补」——这是 R13 对个人中心「资料状态」的核心体验修正。
	// 完成度 <100% 时用「分段点」表达（每项一点，避免一根长条读作装饰横线）；
	// 100% 时不再画条，直接给完成态说明（原实现 100% 的进度条读起来像一条分割线）。
	// items 由页面按资料接口实际返回的字段组装（不改数据获取逻辑）。

	export type CompletenessItem = {
		id: string;
		/** 缺失项文案（如「上传一张头像」） */
		label: string;
		done: boolean;
		/** 点击补全项的去向 */
		href: string;
	};

	let {
		items,
		class: className = ''
	}: {
		items: CompletenessItem[];
		class?: string;
	} = $props();

	const missing = $derived(items.filter((i) => !i.done));
	const filledCount = $derived(items.length - missing.length);
	const pct = $derived(items.length === 0 ? 100 : Math.round((filledCount / items.length) * 100));
	const hint = $derived(`再补 ${missing.length} 项就完整了——完善后的主页更容易被附近的人记住。`);
</script>

<aside class={cn('rounded-lg bg-surface-island p-5 shadow-xs', className)} aria-label="资料状态">
	<SectionHead title="资料状态" icon={ShieldCheck} count={`${filledCount}/${items.length} 项`} />

	{#if missing.length === 0}
		<!-- 完成态：不画进度条（100% 长条会被读成一条装饰横线），直接给状态说明 -->
		<div class="mt-1 flex items-center gap-3">
			<span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-status-online-bg text-text-status-online-on-soft">
				<CheckCircle2 class="size-5" strokeWidth={1.75} />
			</span>
			<div class="min-w-0">
				<p class="text-sm font-medium text-text-primary">资料已完善</p>
				<p class="mt-0.5 text-xs leading-relaxed text-text-secondary">主页会更容易被附近的人记住，随时可以在编辑页更新。</p>
			</div>
		</div>
	{:else}
		<div class="mt-1 flex items-end gap-2">
			<span class="text-2xl font-semibold leading-none tracking-tight tabular-nums text-text-primary">{pct}%</span>
			<span class="pb-0.5 text-xs text-text-secondary">资料完整度</span>
		</div>
		<!-- 分段点：已填项 accent 实心，未填项淡底；读起来是「进度」而不是一根横线 -->
		<div
			class="mt-3 flex gap-1.5"
			role="progressbar"
			 aria-label="资料完整度"
			aria-valuemin={0}
			aria-valuemax={100}
			aria-valuenow={pct}
		>
			{#each items as item (item.id)}
				<span
					aria-hidden="true"
					class="h-1.5 flex-1 rounded-full {item.done ? 'bg-accent-default' : 'bg-surface-subtle'}"
				></span>
			{/each}
		</div>
		<p class="mt-3 text-xs leading-relaxed text-text-secondary">{hint}</p>

		<div class="mt-4 border-t border-border-subtle pt-3">
			<p class="text-xxs font-medium uppercase tracking-eyebrow text-text-secondary">建议补全</p>
			<div class="mt-1.5 flex flex-col gap-0.5">
				{#each missing as item (item.id)}
					<button
						type="button"
						class="hover-bg -mx-2 flex items-center gap-2 rounded-sm px-2 py-2 text-left focus-ring"
						onclick={() => void goto(item.href)}
					>
						<Circle class="size-3.5 shrink-0 text-text-secondary opacity-subtle" strokeWidth={1.75} />
						<span class="min-w-0 flex-1 truncate text-xs text-text-primary">{item.label}</span>
						<ChevronRight class="size-3.5 shrink-0 text-text-secondary" strokeWidth={1.5} />
					</button>
				{/each}
			</div>
		</div>
	{/if}
</aside>

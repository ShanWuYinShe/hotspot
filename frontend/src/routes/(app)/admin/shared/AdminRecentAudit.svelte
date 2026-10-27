<script lang="ts">
	import type { Component } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		History, Flag, KeyRound, Megaphone, ScrollText, Settings2, ShieldCheck, Tags, UserCog, UsersRound
	} from '@lucide/svelte';
	import { adminListAuditLogs } from '@/api';
	import type { AuditLogItem } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { ROUTES } from '@/lib/routes';
	import { formatDateTime } from '@/lib/utils';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import Skeleton from '@/components/data-display/Skeleton.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import Button from '@/components/ui/button.svelte';
	import { AUDIT_ACTION_LABELS, AUDIT_TARGET_LABELS } from './labels';

	// /admin 概览的「最近审计活动」预览（私有组件，自取数据 limit 6）：
	// 行结构 = 动作（中文）+ 对象 + 操作者 + 时间；右侧「查看全部」进入审计日志页。
	// 无 admin.audit.read 权限时整块不渲染（诚实降级，不留空壳）。

	const canRead = $derived(hasAdminPermission('admin.audit.read'));

	let items = $state<AuditLogItem[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let reloadKey = $state(0);

	$effect(() => {
		if (!canRead) return;
		void reloadKey;
		let cancelled = false;
		loading = true;
		error = null;
		void adminListAuditLogs({ pagination: { limit: 6 } })
			.then((res) => {
				if (cancelled) return;
				if (res.code === 0 && res.data) items = res.data.items ?? [];
				else error = res.message ?? '加载失败';
			})
			.catch(() => {
				if (!cancelled) error = '网络异常，请稍后重试';
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});
		return () => {
			cancelled = true;
		};
	});

	// 目标类型 → 行首图标砖（同类操作一眼可辨；untyped 回退通用图标）
	const TARGET_ICONS: Record<string, Component> = {
		user: UserCog,
		group: UsersRound,
		announcement: Megaphone,
		role: KeyRound,
		admin: ShieldCheck,
		report: Flag,
		tag: Tags,
		config: Settings2
	};
	const iconFor = (log: AuditLogItem): Component => TARGET_ICONS[log.target_type] ?? ScrollText;
	const objectText = (log: AuditLogItem): string =>
		`${AUDIT_TARGET_LABELS[log.target_type] ?? log.target_type} ${log.target_name ?? log.target_id}`;
</script>

{#if canRead}
	<section aria-label="最近审计活动" class="flex flex-col">
		<SectionHead
			title="最近审计活动"
			icon={History}
			count={loading ? undefined : items.length}
			action="查看全部"
			onAction={() => void goto(`${ROUTES.admin}/audit`)}
		/>
		<div class="rounded-lg bg-surface-island p-2 shadow-xs">
			{#if loading}
				<!-- 加载骨架与行结构同几何（3 行，避免数据到达时跳高） -->
				<div class="flex flex-col gap-y-0.5" role="status" aria-busy="true" aria-label="加载最近审计活动…">
					{#each Array(3) as _, i (i)}
						<div class="flex items-center gap-3 px-3 py-2.5">
							<Skeleton class="size-8 shrink-0 rounded-full" />
							<div class="flex min-w-0 flex-1 flex-col gap-2">
								<Skeleton class="rounded-sm" style="height: 13px; width: 52%;" />
								<Skeleton class="rounded-sm" style="height: 11px; width: 30%;" />
							</div>
						</div>
					{/each}
				</div>
			{:else if error}
				<EmptyState bare class="py-8" title="审计活动加载失败" description={error}>
					{#snippet action()}
						<Button type="button" variant="secondary" size="sm" onclick={() => (reloadKey += 1)}>重试</Button>
					{/snippet}
				</EmptyState>
			{:else if items.length === 0}
				<EmptyState bare class="py-8" title="暂无审计活动" description="平台管理操作发生后会出现在这里。" />
			{:else}
				<ul class="flex flex-col gap-y-0.5">
					{#each items as log (log.log_id)}
						{@const Icon = iconFor(log)}
						<li class="flex items-center gap-3 rounded-sm px-3 py-2.5">
							<span
								class="flex size-8 shrink-0 items-center justify-center rounded-full bg-surface-subtle text-text-secondary"
								aria-hidden="true"
							>
								<Icon class="size-4" strokeWidth={1.75} />
							</span>
							<div class="min-w-0 flex-1">
								<p class="truncate text-sm text-text-primary">
									<span class="font-medium">{AUDIT_ACTION_LABELS[log.action] ?? log.action}</span>
									<span class="mx-1.5 text-text-secondary" aria-hidden="true">·</span>
									<span class="text-text-secondary">{objectText(log)}</span>
								</p>
								<p class="mt-0.5 truncate text-xs text-text-secondary">
									{log.operator?.nickname ?? '系统'}
								</p>
							</div>
							<time datetime={log.created_at} class="shrink-0 text-xs tabular-nums text-text-secondary">
								{formatDateTime(log.created_at)}
							</time>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</section>
{/if}

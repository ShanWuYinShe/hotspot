<script lang="ts">
	import { adminGetAppConfig, adminUpdateAppConfig } from '@/api';
	import type { AppConfig, ContactInfo, FeatureSwitches, AppLimits } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import Button from '@/components/ui/button.svelte';
	import { ROW_BTN_PRIMARY } from '../shared/table';
	import { formatDateTime } from '@/lib/utils';

	// 系统配置：按 section Patch 更新（spec api_spec_admin.md）。
	// UI 覆盖客户端可见三项（features/limits/contact，构成 ClientConfig 投影）；
	// 仅服务端的 upload/defaults/rate_limits/response_defaults 以只读 JSON 展示（写入仍可经 API 扩展）。

	const canRead = $derived(hasAdminPermission('admin.app_config.read'));
	const canWrite = $derived(hasAdminPermission('admin.app_config.write'));

	let config = $state<AppConfig | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let pending = $state('');

	// 各 section 的本地编辑副本
	let features = $state<FeatureSwitches | null>(null);
	let limits = $state<AppLimits | null>(null);
	let contact = $state<ContactInfo | null>(null);

	async function load() {
		loading = true;
		error = null;
		const res = await adminGetAppConfig().catch(() => ({ code: -1, message: '网络异常', data: null }) as const);
		if (res.code === 0 && res.data) {
			config = res.data;
			features = { ...res.data.features };
			limits = { ...res.data.limits, reaction_emojis: [...res.data.limits.reaction_emojis] };
			contact = { ...res.data.contact };
		} else {
			error = res.message ?? '加载失败';
		}
		loading = false;
	}
	$effect(() => {
		if (canRead) void load();
	});

	const saveSection = async (section: 'features' | 'limits' | 'contact') => {
		if (pending || !config) return;
		pending = section;
		try {
			const body =
				section === 'features' ? { features }
				: section === 'limits' ? { limits }
				: { contact };
			const res = await adminUpdateAppConfig(body);
			if (res.code !== 0) throw new Error(res.message || '保存失败');
			showToast('配置已保存', 'success');
			void load();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '保存失败，请重试', 'danger');
		} finally {
			pending = '';
		}
	};

	const SECTION_CARD = 'rounded-lg border border-border-default bg-surface-primary p-5 shadow-raised-sm';
	const LABEL = 'text-xs text-text-secondary';
	const INPUT = 'input-surface w-full rounded-md px-3 py-2 text-sm text-text-primary';
	const FEATURE_LABELS: Record<keyof FeatureSwitches, string> = {
		enable_public_registration: '开放注册',
		enable_guest_register: '访客浏览',
		enable_moments: '瞬间功能',
		enable_nearby_people: '附近的人'
	};
</script>

<svelte:head>
	<title>系统配置 · HotSpot</title>
</svelte:head>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.app_config.read 权限" />
{:else if loading}
	<InlineLoading label="加载应用配置…" />
{:else if error}
	<LoadError message={error} onRetry={() => void load()} />
{:else if config && features && limits && contact}
	<div class="flex flex-col gap-5 max-w-5xl">
		{#if config.updated_at}
			<p class="text-xs text-text-secondary">最后更新：{formatDateTime(config.updated_at)}{config.updated_by_user_id ? ` · ${config.updated_by_user_id}` : ''}</p>
		{/if}

		<!-- 功能开关 -->
		<section class={SECTION_CARD} aria-label="功能开关">
			<div class="flex items-center justify-between">
				<div>
					<h2 class="text-sm font-semibold text-text-primary">功能开关</h2>
					<p class="mt-1 text-xs text-text-secondary">客户端可见（ClientConfig 投影），保存后全端生效。</p>
				</div>
				{#if canWrite}
					<button type="button" class={ROW_BTN_PRIMARY} disabled={pending !== ''} onclick={() => void saveSection('features')}>保存</button>
				{/if}
			</div>
			<div class="mt-4 grid grid-cols-1 gap-2 sm:grid-cols-2">
				{#each Object.keys(FEATURE_LABELS) as fk (fk)}
					<label class="flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm text-text-primary hover:bg-interactive-hover">
						<input type="checkbox" bind:checked={features[fk as keyof FeatureSwitches]} disabled={!canWrite} />
						{FEATURE_LABELS[fk as keyof FeatureSwitches]}
					</label>
				{/each}
			</div>
		</section>

		<!-- 应用限制 -->
		<section class={SECTION_CARD} aria-label="应用限制">
			<div class="flex items-center justify-between">
				<div>
					<h2 class="text-sm font-semibold text-text-primary">应用限制</h2>
					<p class="mt-1 text-xs text-text-secondary">数值上限类配置（正整数）。</p>
				</div>
				{#if canWrite}
					<button type="button" class={ROW_BTN_PRIMARY} disabled={pending !== ''} onclick={() => void saveSection('limits')}>保存</button>
				{/if}
			</div>
			<div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
				{#each Object.entries(limits) as [key, value] (key)}
					{#if typeof value === 'number'}
						<label class="flex flex-col gap-1">
							<span class={LABEL}>{key}</span>
							<input type="number" min={0} bind:value={limits[key as keyof AppLimits]} disabled={!canWrite} class={INPUT} />
						</label>
					{/if}
				{/each}
			</div>
		</section>

		<!-- 联系/合规信息 -->
		<section class={SECTION_CARD} aria-label="联系信息">
			<div class="flex items-center justify-between">
				<div>
					<h2 class="text-sm font-semibold text-text-primary">联系与合规</h2>
					<p class="mt-1 text-xs text-text-secondary">support_email 须为有效 Email，*_url 须为有效 URL（服务端校验）。</p>
				</div>
				{#if canWrite}
					<button type="button" class={ROW_BTN_PRIMARY} disabled={pending !== ''} onclick={() => void saveSection('contact')}>保存</button>
				{/if}
			</div>
			<div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
				{#each Object.keys(contact) as key (key)}
					<label class="flex flex-col gap-1">
						<span class={LABEL}>{key}</span>
						<input bind:value={contact[key as keyof ContactInfo]} disabled={!canWrite} class={INPUT} />
					</label>
				{/each}
			</div>
		</section>

		<!-- 仅服务端 sections：只读展示 -->
		<section class={SECTION_CARD} aria-label="服务端配置（只读）">
			<h2 class="text-sm font-semibold text-text-primary">服务端运行参数（只读）</h2>
			<p class="mt-1 text-xs text-text-secondary">upload / defaults / rate_limits / response_defaults——修改走 app_config/update API（Patch 语义）。</p>
			<pre class="mt-3 max-h-80 overflow-auto rounded-md bg-surface-subtle p-3 font-mono text-xxs leading-relaxed text-text-secondary">{JSON.stringify({ upload: config.upload, defaults: config.defaults, rate_limits: config.rate_limits, response_defaults: config.response_defaults }, null, 2)}</pre>
		</section>
	</div>
{/if}

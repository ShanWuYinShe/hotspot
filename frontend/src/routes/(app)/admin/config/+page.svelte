<script lang="ts">
	import { adminGetAppConfig, adminUpdateAppConfig } from '@/api';
	import type { AppConfig, AuthConfig, ContactInfo, FeatureSwitches, AppLimits } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import Button from '@/components/ui/button.svelte';
	import { formatDateTime } from '@/lib/utils';

	// 系统配置：按 section Patch 更新（spec api_spec_admin.md）。
	// UI 覆盖客户端可见三项（features/limits/contact，构成 ClientConfig 投影）；
	// 认证安全参数（auth：验证码 TTL / 登录锁定阈值）提供编辑表单；
	// 其余仅服务端 sections（upload/defaults/rate_limits/response_defaults）以只读 JSON 展示。

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
	let auth = $state<AuthConfig | null>(null);

	async function load() {
		loading = true;
		error = null;
		const res = await adminGetAppConfig().catch(() => ({ code: -1, message: '网络异常', data: null }) as const);
		if (res.code === 0 && res.data) {
			config = res.data;
			features = { ...res.data.features };
			limits = { ...res.data.limits, reaction_emojis: [...res.data.limits.reaction_emojis] };
			contact = { ...res.data.contact };
			auth = { ...res.data.auth };
			snapshot = JSON.stringify({ features, limits, contact, auth });
		} else {
			error = res.message ?? '加载失败';
		}
		loading = false;
	}
	$effect(() => {
		if (canRead) void load();
	});

	// 脏检测基线：load 成功后的四段序列化快照（r60：全局「保存更改/重置」取代各卡散落保存）
	let snapshot = $state('');

	type ConfigSection = 'features' | 'limits' | 'contact' | 'auth';
	const dirtySections = $derived.by<ConfigSection[]>(() => {
		if (!snapshot) return [];
		try {
			const s = JSON.parse(snapshot) as Record<string, unknown>;
			const out: ConfigSection[] = [];
			if (JSON.stringify(features) !== JSON.stringify(s.features)) out.push('features');
			if (JSON.stringify(limits) !== JSON.stringify(s.limits)) out.push('limits');
			if (JSON.stringify(contact) !== JSON.stringify(s.contact)) out.push('contact');
			if (JSON.stringify(auth) !== JSON.stringify(s.auth)) out.push('auth');
			return out;
		} catch {
			return [];
		}
	});
	const hasDirty = $derived(dirtySections.length > 0);

	// 认证安全参数服务端要求正整数（spec api_spec_admin.md「应用配置子类型定义」）
	const authInvalid = $derived(
		auth ? Object.values(auth).some((v) => !Number.isInteger(v) || v <= 0) : false
	);

	const resetEdits = () => {
		if (!config) return;
		features = { ...config.features };
		limits = { ...config.limits, reaction_emojis: [...config.limits.reaction_emojis] };
		contact = { ...config.contact };
		auth = { ...config.auth };
	};

	// 全局保存：仅逐个提交脏 section（API 为按 section Patch 语义）
	const saveDirty = async () => {
		if (pending || !config) return;
		if (dirtySections.includes('auth') && authInvalid) {
			showToast('认证安全参数须为正整数', 'danger');
			return;
		}
		pending = 'all';
		try {
			for (const section of dirtySections) {
				const body =
					section === 'features' ? { features }
					: section === 'limits' ? { limits }
					: section === 'contact' ? { contact }
					: { auth };
				const res = await adminUpdateAppConfig(body);
				if (res.code !== 0) throw new Error(res.message || '保存失败');
			}
			showToast('配置已保存', 'success');
			void load();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '保存失败，请重试', 'danger');
		} finally {
			pending = '';
		}
	};

	const SECTION_CARD = 'rounded-lg bg-surface-island p-5 shadow-xs';
	const LABEL = 'text-xs text-text-secondary';
	const INPUT = 'input-surface w-full rounded-md px-3 py-2 text-sm text-text-primary';
	const FEATURE_LABELS: Record<keyof FeatureSwitches, string> = {
		enable_public_registration: '开放注册',
		enable_guest_register: '访客浏览',
		enable_moments: '瞬间功能',
		enable_nearby_people: '附近的人'
	};
	// 字段名中文化（r60：运维可读性,英文键名仅作只读 JSON 呈现）
	const LIMIT_LABELS: Record<string, string> = {
		max_group_members: '群成员上限',
		max_moment_tags: '单条瞬间标签数上限',
		max_moment_images: '单条瞬间图片数上限',
		moment_description_max_length: '瞬间描述字数上限',
		comment_max_length: '评论字数上限',
		search_radius_max: '最大搜索半径（km）',
		max_chat_files: '单条消息附件数上限',
		message_recall_window_seconds: '消息可撤回时长（秒）',
		stranger_message_daily_quota: '陌生人私信每日条数',
		group_invite_ttl_hours: '群邀请有效时长（小时）',
		group_application_ttl_hours: '入群申请有效时长（小时）'
	};
	// 认证安全参数（auth section，spec api_spec_admin.md AppConfig.auth）
	const AUTH_LABELS: Record<string, string> = {
		verification_code_ttl_minutes: '验证码有效期（分钟）',
		login_lockout_threshold: '登录失败锁定阈值（次）',
		login_lockout_minutes: '锁定时长（分钟，同时为失败计数衰减窗口）'
	};
	const CONTACT_LABELS: Record<string, string> = {
		support_email: '客服邮箱',
		privacy_policy_url: '隐私政策链接',
		terms_of_service_url: '服务条款链接',
		official_website_url: '官方网站链接'
	};

	// 只读 JSON 可滚动区域：运行期设置 tabindex=0，键盘用户可聚焦后用方向键/PageUp 滚动
	//（axe scrollable-region-focusable；静态 tabindex 会触发 svelte a11y_noninteractive_tabindex 误报）。
	function focusableScroll(node: HTMLElement) {
		node.setAttribute('tabindex', '0');
		return {};
	}
</script>

<svelte:head>
	<title>系统配置 · HotSpot</title>
</svelte:head>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.app_config.read 权限" bare />
{:else if loading}
	<InlineLoading label="加载应用配置…" />
{:else if error}
	<LoadError message={error} onRetry={() => void load()} />
{:else if config && features && limits && contact && auth}
	<div class="mx-auto flex w-full max-w-5xl flex-col gap-6">
		{#if config.updated_at}
			<p class="text-xs text-text-secondary">最后更新：{formatDateTime(config.updated_at)}{config.updated_by_user_id ? ` · ${config.updated_by_user_id}` : ''}</p>
		{/if}

		<!-- 功能开关 -->
		<section class={SECTION_CARD} aria-label="功能开关">
			<div class="flex items-center justify-between">
				<div>
					<h2 class="text-md font-semibold text-text-primary">功能开关</h2>
					<p class="mt-1 text-xs text-text-secondary">客户端可见（ClientConfig 投影），保存后全端生效。</p>
				</div>
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
					<h2 class="text-md font-semibold text-text-primary">应用限制</h2>
					<p class="mt-1 text-xs text-text-secondary">数值上限类配置（正整数）。</p>
				</div>
			</div>
			<div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
				{#each Object.entries(limits) as [key, value] (key)}
					{#if typeof value === 'number'}
						<label class="flex flex-col gap-1">
							<span class={LABEL}>{LIMIT_LABELS[key] ?? key}</span>
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
					<h2 class="text-md font-semibold text-text-primary">联系与合规</h2>
					<p class="mt-1 text-xs text-text-secondary">support_email 须为有效 Email，*_url 须为有效 URL（服务端校验）。</p>
				</div>
			</div>
			<div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
				{#each Object.keys(contact) as key (key)}
					<label class="flex flex-col gap-1">
						<span class={LABEL}>{CONTACT_LABELS[key] ?? key}</span>
						<input bind:value={contact[key as keyof ContactInfo]} disabled={!canWrite} class={INPUT} />
					</label>
				{/each}
			</div>
		</section>

		<!-- 认证安全 -->
		<section class={SECTION_CARD} aria-label="认证安全">
			<div class="flex items-center justify-between">
				<div>
					<h2 class="text-md font-semibold text-text-primary">认证安全</h2>
					<p class="mt-1 text-xs text-text-secondary">验证码有效期与登录爆破锁定参数（正整数），保存后即时生效。</p>
				</div>
			</div>
			<div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
				{#each Object.keys(AUTH_LABELS) as key (key)}
					<label class="flex flex-col gap-1">
						<span class={LABEL}>{AUTH_LABELS[key]}</span>
						<input type="number" min={1} step={1} bind:value={auth[key as keyof AuthConfig]} disabled={!canWrite} class={INPUT} />
					</label>
				{/each}
			</div>
		</section>

		<!-- 仅服务端 sections：只读展示 -->
		<section class={SECTION_CARD} aria-label="服务端配置（只读）">
			<h2 class="text-md font-semibold text-text-primary">服务端运行参数（只读）</h2>
			<p class="mt-1 text-xs text-text-secondary">upload / defaults / rate_limits / response_defaults——修改走 app_config/update API（Patch 语义）。</p>
			<pre use:focusableScroll role="region" aria-label="服务端配置 JSON（只读，可滚动）" class="mt-3 max-h-80 overflow-auto rounded-md bg-surface-subtle p-3 font-mono text-xxs leading-relaxed text-text-secondary">{JSON.stringify({ upload: config.upload, defaults: config.defaults, rate_limits: config.rate_limits, response_defaults: config.response_defaults }, null, 2)}</pre>
		</section>

		<!-- 全局保存栏：仅存在未保存修改时出现（r60：卡内散落保存收敛为全局「保存更改/重置」） -->
		{#if canWrite && hasDirty}
			<div class="sticky bottom-4 z-[var(--cst-z-overlay)] flex items-center justify-between gap-3 rounded-lg bg-surface-primary/95 px-4 py-3 shadow-raised-md backdrop-blur-md" role="status">
				<p class="text-xs text-text-secondary" aria-live="polite">{dirtySections.length} 个分组有未保存的修改</p>
				<div class="flex items-center gap-2">
					<Button type="button" variant="secondary" size="sm" onclick={resetEdits} disabled={pending !== ''}>重置</Button>
					<Button type="button" variant="primary" size="sm" onclick={() => void saveDirty()} loading={pending !== ''}>保存更改</Button>
				</div>
			</div>
		{/if}
	</div>
{/if}

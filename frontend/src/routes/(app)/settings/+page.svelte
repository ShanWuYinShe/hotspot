<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { Bell, Lock, ChevronRight, LogOut, Info, MapPin, Palette, Trees, Moon, Sun } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import Toggle from '@/components/forms/Toggle.svelte';
	import SettingRow from '@/components/forms/SettingRow.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import { logout } from '@/stores/auth.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { theme, setTheme, toggleDark, THEME_OPTIONS } from '@/stores/theme.svelte';
	import { getAccount, getSettings, updateSettings , netError } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import pkg from '../../../../package.json';

	// 设置页：主题/深色、通知、隐私开关（iOS 式）、关于、退出登录。
	// 服务端字段对齐 spec UserSettings：位置共享 → location_visible、私密账号 →
	// privacy_level；推送通知 spec 无服务端字段，是本地客户端偏好，
	// 用 localStorage 持久化（不冒充服务端设置）。

	const APP_VERSION = `v ${pkg.version}`;

	// 本地偏好（推送通知）——localStorage 持久化
	const NOTIF_KEY = 'hotspot:notif';
	const device = $derived(platform.platform);
	const readLocal = (key: string, fallback: boolean) => {
		try {
			const v = localStorage.getItem(key);
			return v === null ? fallback : v === '1';
		} catch {
			return fallback;
		}
	};
	const writeLocal = (key: string, value: boolean) => {
		try {
			localStorage.setItem(key, value ? '1' : '0');
		} catch {}
	};

	let notif = $state(readLocal(NOTIF_KEY, true));
	let privateProfile = $state(false);
	let locationSharing = $state(true);
	let accountName = $state('');
	let accountEmail = $state('');
	let accountAvatar = $state('');
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	// 服务端设置开关：乐观更新 + 失败回滚并提示（spec: users/update_settings；
	// 业务失败 code!==0 与网络抛错统一走 catch 回滚 + toast）
	const toggleLocationSharing = () => {
		const next = !locationSharing;
		locationSharing = next;
		void updateSettings({ settings: { location_visible: next } })
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message ?? '设置失败');
			})
			.catch(() => {
				locationSharing = !next;
				showToast('设置失败，请稍后重试', 'danger');
			});
	};
	const togglePrivateProfile = () => {
		const next = !privateProfile;
		privateProfile = next;
		void updateSettings({ settings: { privacy_level: next ? 'private' : 'public' } })
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message ?? '设置失败');
			})
			.catch(() => {
				privateProfile = !next;
				showToast('设置失败，请稍后重试', 'danger');
			});
	};
	const toggleNotif = () => {
		notif = !notif;
		writeLocal(NOTIF_KEY, notif);
	};

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchData() {
			try {
				loading = true;
				loadError = null;
				const [accountRes, settingsRes] = await Promise.all([
					getAccount().catch(() => netError(null)),
					getSettings().catch(() => netError(null))
				]);
				if (cancelled) return;

				const accountData = accountRes.code === 0 ? accountRes.data : null;
				const settingsData = settingsRes.code === 0 ? settingsRes.data : null;

				if (accountData) {
					accountName = accountData.nickname ?? '';
					accountEmail = accountData.email ?? '';
					accountAvatar = accountData.avatar_url ?? '';
				}

				if (settingsData) {
					if (settingsData.location_visible !== undefined) locationSharing = settingsData.location_visible;
					if (settingsData.privacy_level !== undefined) privateProfile = settingsData.privacy_level === 'private';
				}

				// 两个请求都失败才整页报错：任一成功则页面主体（开关）仍可用，避免误报
				if (accountRes.code !== 0 && settingsRes.code !== 0) {
					loadError = accountRes.message ?? '加载失败，请稍后重试';
				}
			} catch {
				if (!cancelled) loadError = '加载失败，请稍后重试';
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchData();
		return () => {
			cancelled = true;
		};
	});

	const handleLogout = async () => {
		// auth store logout clears client auth state and keeps the user on the
		// current route (anonymous browsing). No more redirect to a dead /login.
		await logout();
		void goto(ROUTES.home);
	};
</script>
<svelte:head>
	<title>设置 · HotSpot</title>
</svelte:head>


<!-- contentMaxWidth 560：设置页是表单/列表型页面，走 .page-shell--form 同档窄栏；
	默认全宽会让每行设置项在桌面端拉满视口、Toggle 远离标题 -->
<StandardPageLayout contentMaxWidth={560} contentClassName="animate-fade-in pt-4 pb-10">
	{#snippet header()}
		<PageHeader title="设置" onBack={() => history.back()} showBackOnDesktop maxWidth={560} titleSize={device === 'desktop' ? 'xl' : '2xl'} />
	{/snippet}

		{#if loadError}
		<LoadError message={loadError} onRetry={() => reloadKey++} />
		{:else}
	<!-- User card — hero with ambient glow（loading 期间显示占位骨架） -->
		<div class="ambient-glow mb-3">
			<button
				type="button"
				class="hover-btn flex items-center gap-4 px-4 py-3.5 rounded-lg w-full focus-ring text-left surface-hero"
				onclick={() => void goto(ROUTES.meEdit)}
			>
				<Avatar class="border-2 border-surface-primary shadow-glow-md" src={accountAvatar} name={accountName} size={48}  />
				<div class="flex-1 min-w-0">
					{#if loading}
						<p class="text-sm tracking-tight text-text-secondary">加载中…</p>
					{:else}
						<p class="text-sm tracking-tight text-text-primary">{accountName || '用户'}</p>
						<p class="text-xs mt-0.5 text-text-secondary">{accountEmail || '未绑定邮箱'}</p>
					{/if}
				</div>
				<ChevronRight class="size-4 shrink-0 text-text-secondary"   strokeWidth={1.5} />
			</button>
		</div>

	<!-- Appearance（对齐 Figma Make SettingsPage：应用主题二选一 + 深色模式） -->
	<p class="eyebrow px-1 mt-6 mb-2 text-text-primary opacity-emphasis">外观</p>
	<div class="card-raised rounded-lg overflow-hidden">
	<ListContainer>
			<SettingRow title="应用主题" description="选择一套全局视觉风格">
				{#snippet icon()}
					<Palette class="size-4 text-accent-default"   strokeWidth={1.5} />
				{/snippet}
					{#snippet extension()}
						<div class="grid grid-cols-2 gap-2">
							{#each THEME_OPTIONS as t}
								{@const selected = theme.theme === t.id}
								<!-- 分段选择器显式两态（与 AppearanceMenu 主题按钮同一套 token 组合）：
									未选中带边框常显可点区域，不用 surface-ghost（其未选中边框透明，
									在 surface-flat 大底上第二个选项完全隐形） -->
								<button
									type="button"
									aria-pressed={selected}
									onclick={() => setTheme(t.id)}
									class="flex items-center gap-2 rounded-sm border px-3 py-2.5 text-left outline-none transition-colors focus-visible:ring-2 focus-visible:ring-accent-default {selected
										? 'border-accent-border bg-accent-soft text-accent-default'
										: 'border-border-default text-text-secondary hover:bg-interactive-hover'}"
								>
									{#if t.id === 'forest'}
										<Trees class="size-4"   strokeWidth={1.75} />
									{:else}
										<Palette class="size-4"   strokeWidth={1.75} />
									{/if}
									<span class="text-xs">{t.label}</span>
								</button>
							{/each}
						</div>
					{/snippet}
			</SettingRow>
			<SettingRow title="深色模式" description={theme.isDark ? '当前使用深色外观' : '当前使用浅色外观'}>
				{#snippet icon()}
					{#if theme.isDark}
						<Moon class="size-4 text-accent-default"   strokeWidth={1.5} />
					{:else}
						<Sun class="size-4 text-accent-default"   strokeWidth={1.5} />
					{/if}
				{/snippet}
				<Toggle on={theme.isDark} onToggle={toggleDark} label="深色模式" />
			</SettingRow>
	</ListContainer>
	</div>

	<!-- Notify -->
	<p class="eyebrow px-1 mt-6 mb-2 text-text-primary opacity-emphasis">通知</p>
	<div class="card-raised rounded-lg overflow-hidden">
	<ListContainer>
			<SettingRow title="推送通知" description="消息、提及与新动态提醒">
				{#snippet icon()}
					<Bell class="size-4 text-accent-default"   strokeWidth={1.5} />
				{/snippet}
				<Toggle on={notif} onToggle={toggleNotif} label="推送通知" />
			</SettingRow>
	</ListContainer>
	</div>

	<!-- Privacy -->
	<p class="eyebrow px-1 mt-6 mb-2 text-text-primary opacity-emphasis">隐私与安全</p>
	<div class="card-raised rounded-lg overflow-hidden">
	<ListContainer>
			<SettingRow title="私密账号" description="仅关注者可查看你的内容">
				{#snippet icon()}
					<Lock class="size-4 text-accent-default"   strokeWidth={1.5} />
				{/snippet}
				<Toggle on={privateProfile} onToggle={togglePrivateProfile} label="私密账号" />
			</SettingRow>
			<SettingRow title="位置共享" description="允许附近用户发现你">
				{#snippet icon()}
					<MapPin class="size-4 text-accent-default"   strokeWidth={1.5} />
				{/snippet}
				<Toggle on={locationSharing} onToggle={toggleLocationSharing} label="位置共享" />
			</SettingRow>
	</ListContainer>
	</div>

	<!-- General -->
	<p class="eyebrow px-1 mt-6 mb-2 text-text-primary opacity-emphasis">通用</p>
	<div class="card-raised rounded-lg overflow-hidden">
	<ListContainer>
			<!-- 关于：复用 SettingRow 消除手写重复（版本号取自 package.json） -->
			<SettingRow title="关于" description="HotSpot">
				{#snippet icon()}
					<Info class="size-4 text-accent-default"   strokeWidth={1.5} />
				{/snippet}
				<span class="text-sm shrink-0 text-text-secondary">{APP_VERSION}</span>
			</SettingRow>
	</ListContainer>
	</div>

	<!-- Log Out — flat, less visually heavy -->
	<div class="mt-5">
		<button
			type="button"
			class="hover-btn flex items-center justify-center gap-2 px-4 py-3 rounded-md w-full focus-ring text-accent-default"
			onclick={() => void handleLogout()}
		>
			<LogOut class="size-3.5"  strokeWidth={1.75} />
			<span class="text-sm">退出登录</span>
		</button>
	</div>
		{/if}
</StandardPageLayout>

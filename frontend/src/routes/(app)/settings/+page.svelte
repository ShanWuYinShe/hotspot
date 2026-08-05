<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { Bell, Lock, ChevronRight, LogOut, Globe, Info, Volume2, MapPin, User, Palette, Trees, Moon, Sun } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
import Avatar from '@/components/shared/Avatar.svelte';
import Toggle from '@/components/forms/Toggle.svelte';
import EmbeddedList from '@/components/data-display/EmbeddedList.svelte';
import InteractiveListRow from '@/components/navigation/InteractiveListRow.svelte';
	import { logout } from '@/stores/auth.svelte';
	import { theme, setTheme, toggleDark, type AppTheme } from '@/stores/theme.svelte';
	import { getAccount, getSettings } from '@/api';
	import { ROUTES } from '@/lib/routes';

	// 设置页：通知/声音/隐私开关（iOS 式）、语言、账号信息、退出登录。

	let notif = $state(true);
	let sounds = $state(true);
	let privateProfile = $state(false);
	let locationSharing = $state(true);
	let accountName = $state('');
	let accountHandle = $state('');
	let accountEmail = $state('');
	let accountAvatar = $state('');
	let loading = $state(true);

	$effect(() => {
		let cancelled = false;
		async function fetchData() {
			try {
				loading = true;
				const [accountRes, settingsRes] = await Promise.all([
					getAccount().catch(() => ({ code: -1, data: null })),
					getSettings().catch(() => ({ code: -1, data: null }))
				]);
				if (cancelled) return;

				const accountData = (accountRes.code === 0 ? accountRes.data : null) as any;
				const settingsData = (settingsRes.code === 0 ? settingsRes.data : null) as any;

				if (accountData) {
					accountName = accountData.name ?? accountData.nickname ?? '';
					accountHandle = accountData.handle ?? accountData.username ?? '';
					accountEmail = accountData.email ?? '';
					accountAvatar = accountData.avatar_url ?? accountData.avatar ?? '';
				}

				if (settingsData) {
					if (settingsData.notifications !== undefined) notif = settingsData.notifications;
					if (settingsData.sounds !== undefined) sounds = settingsData.sounds;
					if (settingsData.private_profile !== undefined) privateProfile = settingsData.private_profile;
					if (settingsData.location_sharing !== undefined) locationSharing = settingsData.location_sharing;
				}
			} catch {
				// empty data is normal
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

<StandardPageLayout contentClassName="py-5 pb-10">
	{#snippet header()}
		<PageHeader title="设置" sticky={false} onBack={() => history.back()} showBackOnDesktop />
	{/snippet}

	<!-- User card — hero with ambient glow -->
	<div class="ambient-glow mb-3">
		<button type="button" class="hover-btn flex items-center gap-4 px-4 py-3.5 rounded-panel w-full outline-none text-left surface-hero">
			<Avatar src={accountAvatar} name={accountName} size={48} style="border:2px solid var(--surface-primary);box-shadow:0 4px 12px var(--glow-accent);" />
			<div class="flex-1 min-w-0">
				<p class="text-[15px] tracking-tight text-text-primary">{accountName || '用户'}</p>
				<p class="text-xs mt-0.5 text-text-secondary">{accountHandle}{accountEmail ? ` · ${accountEmail}` : ''}</p>
			</div>
			<ChevronRight style="width:15px;height:15px;color:var(--text-secondary);shrink:0;" strokeWidth={1.5} />
		</button>
	</div>

	<!-- Appearance（对齐 Figma Make SettingsPage：应用主题二选一 + 深色模式） -->
	<p class="px-1 mt-3 mb-1.5 text-text-secondary text-xxs tracking-[0.14em] uppercase">外观</p>
	<div class="rounded-panel overflow-hidden surface-flat">
		<EmbeddedList>
			<div class="px-4 py-3.5">
				<div class="flex items-center gap-3">
					<div class="flex items-center justify-center rounded-full shrink-0 icon-badge" style="width:36px;height:36px;">
						<Palette style="width:15px;height:15px;color:var(--accent-default);" strokeWidth={1.5} />
					</div>
					<div class="flex-1 min-w-0">
						<p class="text-[14px] text-text-primary">应用主题</p>
						<p class="text-xs text-text-secondary">选择一套全局视觉风格</p>
					</div>
				</div>
				<div class="mt-3 grid grid-cols-2 gap-2">
					{#each ([{ id: 'forest', label: '清野' }, { id: 'vibrant', label: '星际' }] as { id: AppTheme; label: string }[]) as t}
						<button
							type="button"
							aria-pressed={theme.theme === t.id}
							onclick={() => setTheme(t.id)}
							class="flex items-center gap-2 border px-3 py-2.5 text-left outline-none transition-colors focus-visible:ring-2 focus-visible:ring-accent-default"
							style="border-color:{theme.theme === t.id ? 'var(--accent-default)' : 'var(--border-default)'};background:{theme.theme === t.id ? 'var(--accent-soft)' : 'transparent'};color:var(--text-primary);"
						>
							{#if t.id === 'forest'}
								<Trees style="width:15px;height:15px;color:var(--accent-default);" strokeWidth={1.7} />
							{:else}
								<Palette style="width:15px;height:15px;color:var(--accent-default);" strokeWidth={1.7} />
							{/if}
							<span class="text-xs">{t.label}</span>
						</button>
					{/each}
				</div>
			</div>
			<div class="flex items-center gap-3 px-4 py-3.5">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge" style="width:36px;height:36px;">
					{#if theme.isDark}
						<Moon style="width:15px;height:15px;color:var(--accent-default);" strokeWidth={1.5} />
					{:else}
						<Sun style="width:15px;height:15px;color:var(--accent-default);" strokeWidth={1.5} />
					{/if}
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-[14px] text-text-primary">深色模式</p>
					<p class="text-xs text-text-secondary">{theme.isDark ? '当前使用深色外观' : '当前使用浅色外观'}</p>
				</div>
				<Toggle on={theme.isDark} onToggle={toggleDark} label="深色模式" />
			</div>
		</EmbeddedList>
	</div>

	<!-- Notifications -->
	<p class="px-1 mt-3 mb-1.5 text-text-secondary text-xxs tracking-[0.14em] uppercase">通知</p>
	<div class="rounded-panel overflow-hidden surface-flat">
		<EmbeddedList>
			<div class="flex items-center gap-3 px-4 py-3.5">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge" style="width:36px;height:36px;">
					<Bell style="width:15px;height:15px;color:var(--text-secondary);" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-[14px] text-text-primary">推送通知</p>
					<p class="text-xs text-text-secondary">消息、提及与新动态提醒</p>
				</div>
				<Toggle on={notif} onToggle={() => (notif = !notif)} />
			</div>
			<div class="flex items-center gap-3 px-4 py-3.5">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge" style="width:36px;height:36px;">
					<Volume2 style="width:15px;height:15px;color:var(--text-secondary);" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-[14px] text-text-primary">应用声音</p>
					<p class="text-xs text-text-secondary">操作反馈音效</p>
				</div>
				<Toggle on={sounds} onToggle={() => (sounds = !sounds)} />
			</div>
		</EmbeddedList>
	</div>

	<!-- Privacy -->
	<p class="px-1 mt-3 mb-1.5 text-text-secondary text-xxs tracking-[0.14em] uppercase">隐私与安全</p>
	<div class="rounded-panel overflow-hidden surface-flat">
		<EmbeddedList>
			<div class="flex items-center gap-3 px-4 py-3.5">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge" style="width:36px;height:36px;">
					<Lock style="width:15px;height:15px;color:var(--text-secondary);" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-[14px] text-text-primary">私密账号</p>
					<p class="text-xs text-text-secondary">仅关注者可查看你的内容</p>
				</div>
				<Toggle on={privateProfile} onToggle={() => (privateProfile = !privateProfile)} />
			</div>
			<div class="flex items-center gap-3 px-4 py-3.5">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge" style="width:36px;height:36px;">
					<MapPin style="width:15px;height:15px;color:var(--text-secondary);" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-[14px] text-text-primary">位置共享</p>
					<p class="text-xs text-text-secondary">允许附近用户发现你</p>
				</div>
				<Toggle on={locationSharing} onToggle={() => (locationSharing = !locationSharing)} />
			</div>
		</EmbeddedList>
	</div>

	<!-- General -->
	<p class="px-1 mt-3 mb-1.5 text-text-secondary text-xxs tracking-[0.14em] uppercase">通用</p>
	<div class="rounded-panel overflow-hidden surface-flat">
		<EmbeddedList>
			<InteractiveListRow class="flex items-center gap-3 px-4 py-3.5" ariaLabel="语言设置">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge" style="width:36px;height:36px;">
					<Globe style="width:15px;height:15px;color:var(--text-secondary);" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-[14px] text-text-primary">语言</p>
				</div>
				<span class="text-sm shrink-0 text-text-secondary">中文</span>
				<ChevronRight style="width:15px;height:15px;color:var(--text-secondary);shrink:0;" strokeWidth={1.5} />
			</InteractiveListRow>
			<InteractiveListRow class="flex items-center gap-3 px-4 py-3.5" ariaLabel="账号信息">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge" style="width:36px;height:36px;">
					<User style="width:15px;height:15px;color:var(--text-secondary);" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-[14px] text-text-primary">账号信息</p>
				</div>
				<span class="text-sm shrink-0 text-text-secondary"></span>
				<ChevronRight style="width:15px;height:15px;color:var(--text-secondary);shrink:0;" strokeWidth={1.5} />
			</InteractiveListRow>
			<InteractiveListRow class="flex items-center gap-3 px-4 py-3.5" ariaLabel="关于">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge" style="width:36px;height:36px;">
					<Info style="width:15px;height:15px;color:var(--text-secondary);" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-[14px] text-text-primary">关于</p>
				</div>
				<span class="text-sm shrink-0 text-text-secondary">v 1.0.0</span>
				<ChevronRight style="width:15px;height:15px;color:var(--text-secondary);shrink:0;" strokeWidth={1.5} />
			</InteractiveListRow>
		</EmbeddedList>
	</div>

	<!-- Log Out — flat, less visually heavy -->
	<div class="mt-5">
		<button
			type="button"
			class="hover-btn flex items-center justify-center gap-2 px-4 py-3 rounded-control w-full outline-none surface-flat text-accent-default"
			onclick={() => void handleLogout()}
		>
			<LogOut style="width:14px;height:14px;" strokeWidth={1.75} />
			<span class="text-[14px]">退出登录</span>
		</button>
	</div>
</StandardPageLayout>

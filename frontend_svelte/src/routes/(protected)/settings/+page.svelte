<script lang="ts">
	import { authStore } from '$lib/stores/auth.svelte';
	import { themeStore, type ThemeFamily } from '$lib/stores/theme.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { t } from '$lib/i18n';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import ThemeToggle from '$lib/components/common/theme-toggle.svelte';
	import User from 'lucide-svelte/icons/user';
	import Bell from 'lucide-svelte/icons/bell';
	import Shield from 'lucide-svelte/icons/shield';
	import Lock from 'lucide-svelte/icons/lock';
	import Palette from 'lucide-svelte/icons/palette';
	import LogOut from 'lucide-svelte/icons/log-out';
	import Trash2 from 'lucide-svelte/icons/trash-2';
	import ChevronRight from 'lucide-svelte/icons/chevron-right';
	import Share2 from 'lucide-svelte/icons/share-2';
	import ShieldAlert from 'lucide-svelte/icons/shield-alert';

	const settingsSections = [
		{
			title: '账户',
			items: [
				{ icon: User, label: '个人资料', href: '/profile', description: '修改头像、昵称、简介' },
				{ icon: Lock, label: '修改密码', href: '/settings/password', description: '更改登录密码' },
				{ icon: Shield, label: '会话管理', href: '/settings/sessions', description: '管理登录设备' }
			]
		},
		{
			title: '隐私',
			items: [
				{ icon: Shield, label: '隐私设置', href: '/settings/privacy', description: '谁可以看到你的信息' }
			]
		},
		{
			title: '通知',
			items: [
				{ icon: Bell, label: '通知设置', href: '/settings/notifications', description: '管理推送通知' }
			]
		},
		{
			title: '记录与分享',
			items: [
				{ icon: Share2, label: '我的分享', href: '/settings/shares', description: '查看并撤销已创建分享' },
				{ icon: ShieldAlert, label: '我的举报记录', href: '/settings/reports', description: '跟踪举报处理进度' }
			]
		}
	];

	const currentThemeFamily = $derived(themeStore.themeFamily);

	function handleLogout() {
		authStore.logout();
		toast.success('已退出登录');
	}

	function setThemeFamily(themeFamily: ThemeFamily) {
		themeStore.setThemeFamily(themeFamily);
	}
</script>

<svelte:head>
	<title>设置 - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-3xl space-y-5">
	<div class="rounded-[1.75rem] border border-border/80 bg-card/92 p-4 shadow-[10px_10px_22px_var(--shadow-dark),-10px_-10px_22px_var(--shadow-light)] backdrop-blur-md">
		<p class="text-sm text-muted-foreground">Settings 页保持与 Harmony 一致的分区顺序：外观主题、隐私安全、通知订阅、账号与记录。</p>
	</div>
	<PageHeader
		title="设置"
		description="统一管理账户、安全、隐私、分享记录与举报进度"
	/>

	{#each settingsSections as section (section.title)}
		<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
			<CardHeader>
				<CardTitle class="text-lg">{section.title}</CardTitle>
			</CardHeader>
			<CardContent class="space-y-2 p-4 pt-0">
				{#each section.items as item (item.href)}
					<a
						href={item.href}
						class="flex items-center justify-between rounded-2xl border border-border/50 bg-background/65 px-4 py-4 transition-all hover:border-primary/25 hover:bg-accent/45"
					>
						<div class="flex items-center gap-4">
							<div class="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
								<item.icon class="h-5 w-5" />
							</div>
							<div>
								<p class="font-medium">{item.label}</p>
								<p class="text-sm text-muted-foreground">{item.description}</p>
							</div>
						</div>
						<ChevronRight class="h-5 w-5 text-muted-foreground" />
					</a>
				{/each}
			</CardContent>
		</Card>
	{/each}

	<!-- 外观 -->
	<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
		<CardHeader>
			<CardTitle class="text-lg">外观</CardTitle>
		</CardHeader>
		<CardContent class="space-y-3 pt-0">
			<div class="flex items-center justify-between rounded-2xl border border-border/50 bg-background/65 px-4 py-4">
				<div class="flex items-center gap-4">
					<div class="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
						<Palette class="h-5 w-5" />
					</div>
					<div>
						<p class="font-medium">{t('sidebar.toggleTheme')}</p>
						<p class="text-sm text-muted-foreground">{t('sidebar.systemMode')} / {t('sidebar.light')} / {t('sidebar.dark')}</p>
					</div>
				</div>
				<ThemeToggle />
			</div>

			<div class="grid gap-3 md:grid-cols-2">
				<button
					type="button"
					class="rounded-3xl border px-4 py-4 text-left transition-all {currentThemeFamily === 'default_clean' ? 'border-primary bg-primary/8 shadow-md' : 'border-border/50 bg-background/65 hover:border-primary/25'}"
					onclick={() => setThemeFamily('default_clean')}
				>
					<div class="mb-3 flex items-center justify-between">
						<p class="font-semibold">{t('sidebar.defaultClean')}</p>
						{#if currentThemeFamily === 'default_clean'}
							<span class="rounded-full bg-primary px-2 py-1 text-xs text-primary-foreground">当前</span>
						{/if}
					</div>
					<p class="text-sm text-muted-foreground">{t('sidebar.defaultCleanDesc')}</p>
				</button>

				<button
					type="button"
					class="rounded-3xl border px-4 py-4 text-left transition-all {currentThemeFamily === 'figma_neu' ? 'border-primary bg-primary/8 shadow-md' : 'border-border/50 bg-background/65 hover:border-primary/25'}"
					onclick={() => setThemeFamily('figma_neu')}
				>
					<div class="mb-3 flex items-center justify-between">
						<p class="font-semibold">{t('sidebar.figmaNeu')}</p>
						{#if currentThemeFamily === 'figma_neu'}
							<span class="rounded-full bg-primary px-2 py-1 text-xs text-primary-foreground">当前</span>
						{/if}
					</div>
					<p class="text-sm text-muted-foreground">{t('sidebar.figmaNeuDesc')}</p>
				</button>
			</div>
		</CardContent>
	</Card>

	<!-- 危险操作 -->
	<Card class="rounded-3xl border border-destructive/25 bg-card/80 shadow-sm backdrop-blur-md">
		<CardHeader>
			<CardTitle class="text-lg text-destructive">危险操作</CardTitle>
		</CardHeader>
		<CardContent class="space-y-4">
			<Button variant="outline" class="h-11 w-full justify-start rounded-2xl border-border/60 bg-background/70" onclick={handleLogout}>
				<LogOut class="mr-2 h-4 w-4" />
				退出登录
			</Button>
			<Button variant="outline" class="h-11 w-full justify-start rounded-2xl border-destructive/40 text-destructive hover:bg-destructive hover:text-destructive-foreground">
				<Trash2 class="mr-2 h-4 w-4" />
				删除账户
			</Button>
		</CardContent>
	</Card>
</div>

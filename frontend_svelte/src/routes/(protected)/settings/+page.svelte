<script lang="ts">
	import { authStore } from '$lib/stores/auth.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Separator } from '$lib/components/ui/separator';
	import ThemeToggle from '$lib/components/common/theme-toggle.svelte';
	import User from 'lucide-svelte/icons/user';
	import Bell from 'lucide-svelte/icons/bell';
	import Shield from 'lucide-svelte/icons/shield';
	import Lock from 'lucide-svelte/icons/lock';
	import Palette from 'lucide-svelte/icons/palette';
	import LogOut from 'lucide-svelte/icons/log-out';
	import Trash2 from 'lucide-svelte/icons/trash-2';
	import ChevronRight from 'lucide-svelte/icons/chevron-right';

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
		}
	];

	function handleLogout() {
		authStore.logout();
		toast.success('已退出登录');
	}
</script>

<svelte:head>
	<title>设置 - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-2xl space-y-6">
	<h1 class="text-2xl font-bold">设置</h1>

	{#each settingsSections as section}
		<Card>
			<CardHeader>
				<CardTitle class="text-lg">{section.title}</CardTitle>
			</CardHeader>
			<CardContent class="space-y-1 p-0">
				{#each section.items as item, i}
					{#if i > 0}
						<Separator />
					{/if}
					<a
						href={item.href}
						class="flex items-center justify-between px-6 py-4 transition-colors hover:bg-accent/50"
					>
						<div class="flex items-center gap-4">
							<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
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
	<Card>
		<CardHeader>
			<CardTitle class="text-lg">外观</CardTitle>
		</CardHeader>
		<CardContent>
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-4">
					<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
						<Palette class="h-5 w-5" />
					</div>
					<div>
						<p class="font-medium">主题</p>
						<p class="text-sm text-muted-foreground">切换深色/浅色模式</p>
					</div>
				</div>
				<ThemeToggle />
			</div>
		</CardContent>
	</Card>

	<!-- 危险操作 -->
	<Card>
		<CardHeader>
			<CardTitle class="text-lg text-destructive">危险操作</CardTitle>
		</CardHeader>
		<CardContent class="space-y-4">
			<Button variant="outline" class="w-full justify-start" onclick={handleLogout}>
				<LogOut class="mr-2 h-4 w-4" />
				退出登录
			</Button>
			<Button variant="outline" class="w-full justify-start text-destructive hover:bg-destructive hover:text-destructive-foreground">
				<Trash2 class="mr-2 h-4 w-4" />
				删除账户
			</Button>
		</CardContent>
	</Card>
</div>

<script lang="ts">
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { notificationStore } from '$lib/stores/notification.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Switch } from '$lib/components/ui/switch';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import Bell from 'lucide-svelte/icons/bell';
	import CheckCheck from 'lucide-svelte/icons/check-check';
	import ExternalLink from 'lucide-svelte/icons/external-link';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	type NotificationPreferenceKey = 'social' | 'interaction' | 'group' | 'system';
	type NotificationPreferences = Record<NotificationPreferenceKey, boolean>;

	const STORAGE_KEY = 'hotspot_notification_preferences';
	const DEFAULT_PREFERENCES: NotificationPreferences = {
		social: true,
		interaction: true,
		group: true,
		system: true
	};

	const preferenceItems: Array<{
		key: NotificationPreferenceKey;
		title: string;
		description: string;
	}> = [
		{ key: 'social', title: '社交关系提醒', description: '新关注、互相关注等社交关系变动' },
		{ key: 'interaction', title: '互动提醒', description: '点赞、评论、回复、提及等互动通知' },
		{ key: 'group', title: '群组提醒', description: '群邀请、入群申请和群组相关通知' },
		{ key: 'system', title: '系统公告', description: '平台公告、维护提醒和系统消息' }
	];

	let preferences = $state<NotificationPreferences>({ ...DEFAULT_PREFERENCES });
	let isSaving = $state(false);
	let isMarkingAllRead = $state(false);

	function loadPreferences() {
		if (!browser) return;
		try {
			const raw = localStorage.getItem(STORAGE_KEY);
			if (!raw) return;
			const parsed = JSON.parse(raw) as Partial<NotificationPreferences>;
			preferences = { ...DEFAULT_PREFERENCES, ...parsed };
		} catch {
			preferences = { ...DEFAULT_PREFERENCES };
		}
	}

	function togglePreference(key: NotificationPreferenceKey, checked: boolean) {
		preferences = { ...preferences, [key]: checked };
	}

	async function savePreferences() {
		if (!browser) return;
		isSaving = true;
		try {
			localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences));
			toast.success('通知偏好已保存到当前浏览器');
		} catch {
			toast.error('保存通知偏好失败');
		} finally {
			isSaving = false;
		}
	}

	async function markAllRead() {
		isMarkingAllRead = true;
		try {
			const ok = await notificationStore.markAllAsRead();
			if (ok) {
				toast.success('已将全部通知标记为已读');
			} else {
				toast.error('标记全部已读失败');
			}
		} catch {
			toast.error('标记全部已读失败');
		} finally {
			isMarkingAllRead = false;
		}
	}

	onMount(() => {
		loadPreferences();
		notificationStore.fetchUnreadCount();
	});
</script>

<svelte:head>
	<title>通知设置 - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-3xl space-y-5">
	<PageHeader title="通知设置" description="管理通知偏好、未读状态和通知中心入口" />

	<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
		<CardHeader>
			<CardTitle class="flex items-center gap-2 text-base">
				<Bell class="h-5 w-5 text-muted-foreground" />
				通知状态
			</CardTitle>
			<CardDescription>
				快速查看当前未读情况，并进入通知中心处理通知。
			</CardDescription>
		</CardHeader>
		<CardContent class="space-y-4 pt-0">
			<div class="flex items-center justify-between rounded-2xl border border-border/50 bg-background/65 px-4 py-4">
				<div>
					<p class="font-medium">当前未读通知</p>
					<p class="text-sm text-muted-foreground">未读数量会和通知下拉、通知列表保持一致</p>
				</div>
				<div class="text-2xl font-semibold tabular-nums">{notificationStore.unreadCount}</div>
			</div>

			<div class="flex flex-col gap-3 sm:flex-row">
				<Button class="h-11 flex-1 rounded-2xl" onclick={() => goto('/notifications')}>
					<ExternalLink class="mr-2 h-4 w-4" />
					进入通知中心
				</Button>
				<Button
					variant="outline"
					class="h-11 flex-1 rounded-2xl"
					onclick={markAllRead}
					disabled={isMarkingAllRead || notificationStore.unreadCount === 0}
				>
					{#if isMarkingAllRead}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
					{:else}
						<CheckCheck class="mr-2 h-4 w-4" />
					{/if}
					全部标记为已读
				</Button>
			</div>
		</CardContent>
	</Card>

	<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
		<CardHeader>
			<CardTitle class="text-base">偏好说明</CardTitle>
			<CardDescription>
				当前仓库尚未接入服务端通知偏好协议，所以这里先提供本地偏好保存与通知管理入口。
			</CardDescription>
		</CardHeader>
		<CardContent class="pt-0">
			<p class="rounded-2xl border border-border/50 bg-background/65 px-4 py-3 text-sm text-muted-foreground">
				这些开关用于保留你在当前浏览器的通知偏好。真正收到的通知仍以服务端与 WebSocket 推送为准。
			</p>
		</CardContent>
	</Card>

	<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
		<CardHeader>
			<CardTitle class="text-base">当前浏览器通知偏好</CardTitle>
			<CardDescription>
				本页设置仅保存在当前浏览器，用于保留你的本地通知偏好；不会影响其他设备。
			</CardDescription>
		</CardHeader>
		<CardContent class="space-y-4 pt-0">
			{#each preferenceItems as item (item.key)}
				<div class="flex items-center justify-between rounded-2xl border border-border/50 bg-background/65 px-4 py-3.5">
					<div>
						<p class="font-medium">{item.title}</p>
						<p class="text-sm text-muted-foreground">{item.description}</p>
					</div>
					<Switch checked={preferences[item.key]} onchange={(checked: boolean) => togglePreference(item.key, checked)} />
				</div>
			{/each}

			<Button class="h-11 w-full rounded-2xl" onclick={savePreferences} disabled={isSaving}>
				{#if isSaving}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				保存通知偏好
			</Button>
		</CardContent>
	</Card>
</div>

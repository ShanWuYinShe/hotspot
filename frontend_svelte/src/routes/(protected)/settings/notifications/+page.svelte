<script lang="ts">
	import { onMount } from 'svelte';
	import { subscriptionsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import type { Subscriptions } from '$lib/types/api';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	type SubscriptionKey = keyof Subscriptions;

	let settings = $state<Subscriptions | null>(null);
	let isLoading = $state(true);
	let isSaving = $state(false);

	const sections: Array<{
		title: string;
		description: string;
		items: Array<{ key: SubscriptionKey; label: string; hint: string }>;
	}> = [
		{
			title: '关注与社交',
			description: '关注与关系变化通知',
			items: [
				{ key: 'new_follower_enabled', label: '新粉丝', hint: '有人关注你' },
				{ key: 'mutual_follow_enabled', label: '互相关注', hint: '你们成为好友' },
				{ key: 'follow_request_enabled', label: '关注请求', hint: '收到关注申请' },
				{ key: 'follow_request_handled_enabled', label: '关注处理结果', hint: '申请通过或拒绝' }
			]
		},
		{
			title: '群组',
			description: '群组相关通知',
			items: [
				{ key: 'group_invite_enabled', label: '群组邀请', hint: '收到群组邀请' },
				{ key: 'group_invite_response_enabled', label: '邀请回应', hint: '对方接受/拒绝邀请' },
				{ key: 'group_application_enabled', label: '入群申请', hint: '收到入群申请' },
				{ key: 'group_application_response_enabled', label: '申请结果', hint: '入群申请处理结果' },
				{ key: 'group_announcement_enabled', label: '群公告', hint: '群公告更新' }
			]
		},
		{
			title: '互动',
			description: '点赞、评论与提及',
			items: [
				{ key: 'moment_like_enabled', label: '动态点赞', hint: '有人点赞你的动态' },
				{ key: 'moment_comment_enabled', label: '动态评论', hint: '有人评论你的动态' },
				{ key: 'comment_reply_enabled', label: '评论回复', hint: '有人回复你的评论' },
				{ key: 'message_mention_enabled', label: '消息提及', hint: '在群聊中提到你' }
			]
		},
		{
			title: '系统',
			description: '系统消息通知',
			items: [
				{ key: 'system_announcement_enabled', label: '系统公告', hint: '平台公告提醒' }
			]
		}
	];

	async function loadSettings() {
		isLoading = true;
		try {
			const response = await subscriptionsService.getSettings();
			if (isSuccess(response) && response.data) {
				settings = response.data;
			}
		} catch {
			toast.error('加载通知设置失败');
		} finally {
			isLoading = false;
		}
	}

	function toggleSetting(key: SubscriptionKey) {
		if (!settings) return;
		settings = { ...settings, [key]: !settings[key] };
	}

	async function saveSettings() {
		if (!settings) return;
		isSaving = true;
		try {
			const response = await subscriptionsService.updateSettings(settings);
			if (isSuccess(response)) {
				toast.success('通知设置已保存');
			} else {
				toast.error(response.message || '保存失败');
			}
		} catch {
			toast.error('保存失败');
		} finally {
			isSaving = false;
		}
	}

	onMount(() => {
		loadSettings();
	});
</script>

<svelte:head>
	<title>通知设置 - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-3xl space-y-6">
	<div>
		<h1 class="text-2xl font-bold">通知设置</h1>
		<p class="text-sm text-muted-foreground">选择你想接收的消息类型</p>
	</div>

	{#if isLoading}
		<Card>
			<CardContent class="space-y-4 p-6">
				{#each Array(6) as _}
					<Skeleton class="h-10 w-full" />
				{/each}
			</CardContent>
		</Card>
	{:else if settings}
		<div class="space-y-4">
			{#each sections as section}
				<Card>
					<CardHeader>
						<CardTitle class="text-base">{section.title}</CardTitle>
						<CardDescription>{section.description}</CardDescription>
					</CardHeader>
					<CardContent class="space-y-3">
						{#each section.items as item}
							<div class="flex items-center justify-between rounded-lg border border-border/60 bg-background/60 px-4 py-3">
								<div>
									<p class="font-medium">{item.label}</p>
									<p class="text-xs text-muted-foreground">{item.hint}</p>
								</div>
								<input
									type="checkbox"
									class="h-4 w-4"
									checked={settings[item.key]}
									onchange={() => toggleSetting(item.key)}
								/>
							</div>
						{/each}
					</CardContent>
				</Card>
			{/each}
		</div>
		<Button class="w-full" onclick={saveSettings} disabled={isSaving}>
			{#if isSaving}
				<Loader2 class="mr-2 h-4 w-4 animate-spin" />
			{/if}
			保存设置
		</Button>
	{/if}
</div>

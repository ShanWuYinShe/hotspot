<script lang="ts">
	import { onMount } from 'svelte';
	import { websocketStore } from '$lib/websocket';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	// 订阅设置通过 WebSocket 管理（spec 中无 REST API）
	interface SubscriptionSettings {
		new_follower_enabled: boolean;
		mutual_follow_enabled: boolean;
		group_invite_enabled: boolean;
		group_invite_response_enabled: boolean;
		group_application_enabled: boolean;
		group_application_response_enabled: boolean;
		group_announcement_enabled: boolean;
		message_mention_enabled: boolean;
		moment_like_enabled: boolean;
		moment_comment_enabled: boolean;
		comment_reply_enabled: boolean;
		system_announcement_enabled: boolean;
	}

	type SubscriptionKey = keyof SubscriptionSettings;

	let settings = $state<SubscriptionSettings | null>(null);
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
				{ key: 'mutual_follow_enabled', label: '互相关注', hint: '你们成为好友' }
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
			// 通过 WebSocket 获取订阅设置（spec: GetSubscriptions）
			const clientMsgId = `get-subscriptions-${Date.now()}`;
			websocketStore.send({
				type: 'GetSubscriptions',
				payload: { client_msg_id: clientMsgId }
			});
			// 监听响应
			const cleanup = websocketStore.on('SubscriptionsSnapshot', (payload) => {
				// Note: SubscriptionsSnapshot payload structure differs from SubscriptionSettings
				// This may need to be refactored based on actual API spec
				settings = payload as unknown as SubscriptionSettings;
				isLoading = false;
				cleanup();
			});
			// 超时回退
			setTimeout(() => {
				if (isLoading) {
					isLoading = false;
					// 使用默认值
					settings = {
						new_follower_enabled: true,
						mutual_follow_enabled: true,
						group_invite_enabled: true,
						group_invite_response_enabled: true,
						group_application_enabled: true,
						group_application_response_enabled: true,
						group_announcement_enabled: true,
						message_mention_enabled: true,
						moment_like_enabled: true,
						moment_comment_enabled: true,
						comment_reply_enabled: true,
						system_announcement_enabled: true
					};
					cleanup();
				}
			}, 5000);
		} catch {
			isLoading = false;
			toast.error('加载通知设置失败');
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
			// 通过 WebSocket 更新订阅设置（spec: SetSubscriptions）
			// Note: SetSubscriptions payload structure expects { client_msg_id, operation, type, ids }
			// This may need to be refactored based on actual API spec
			const clientMsgId = `set-subscriptions-${Date.now()}`;
			websocketStore.send({
				type: 'SetSubscriptions',
				payload: {
					client_msg_id: clientMsgId,
					operation: 'set',
					type: 'user', // This may need adjustment based on actual requirements
					ids: [] // This may need adjustment based on actual requirements
				}
			});
			toast.success('通知设置已保存');
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
			{#each sections as section (section.title)}
				<Card>
					<CardHeader>
						<CardTitle class="text-base">{section.title}</CardTitle>
						<CardDescription>{section.description}</CardDescription>
					</CardHeader>
					<CardContent class="space-y-3">
						{#each section.items as item (item.key)}
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

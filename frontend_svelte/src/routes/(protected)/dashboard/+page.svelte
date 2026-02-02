<script lang="ts">
	import { authStore } from '$lib/stores/auth.svelte';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import Image from 'lucide-svelte/icons/image';
	import Users from 'lucide-svelte/icons/users';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import MapPin from 'lucide-svelte/icons/map-pin';
</script>

<svelte:head>
	<title>仪表板 - HotSpot</title>
</svelte:head>

<div class="space-y-6">
	<h1 class="text-2xl font-bold">欢迎回来，{authStore.currentAccount?.nickname}！</h1>
	<p class="text-muted-foreground">这是你的个人仪表板</p>

	<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
		<Card class="cursor-pointer transition-shadow hover:shadow-lg">
			<a href="/moments">
				<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
					<CardTitle class="text-sm font-medium">我的瞬间</CardTitle>
					<Image class="h-4 w-4 text-muted-foreground" />
				</CardHeader>
				<CardContent>
					<p class="text-xs text-muted-foreground">查看和管理你的瞬间</p>
				</CardContent>
			</a>
		</Card>

		<Card class="cursor-pointer transition-shadow hover:shadow-lg">
			<a href="/groups">
				<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
					<CardTitle class="text-sm font-medium">我的群组</CardTitle>
					<Users class="h-4 w-4 text-muted-foreground" />
				</CardHeader>
				<CardContent>
					<p class="text-xs text-muted-foreground">查看加入的群组</p>
				</CardContent>
			</a>
		</Card>

		<Card class="cursor-pointer transition-shadow hover:shadow-lg">
			<a href="/chat">
				<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
					<CardTitle class="text-sm font-medium">消息</CardTitle>
					<MessageCircle class="h-4 w-4 text-muted-foreground" />
				</CardHeader>
				<CardContent>
					<p class="text-xs text-muted-foreground">查看私信和群聊</p>
				</CardContent>
			</a>
		</Card>

		<Card class="cursor-pointer transition-shadow hover:shadow-lg">
			<a href="/users/nearby">
				<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
					<CardTitle class="text-sm font-medium">附近</CardTitle>
					<MapPin class="h-4 w-4 text-muted-foreground" />
				</CardHeader>
				<CardContent>
					<p class="text-xs text-muted-foreground">发现附近的人和动态</p>
				</CardContent>
			</a>
		</Card>
	</div>
</div>

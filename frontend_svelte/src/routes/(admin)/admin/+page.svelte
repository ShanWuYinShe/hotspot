<script lang="ts">
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import Users from 'lucide-svelte/icons/users';
	import UsersRound from 'lucide-svelte/icons/users-round';
	import FileText from 'lucide-svelte/icons/file-text';
	import Flag from 'lucide-svelte/icons/flag';
</script>

<svelte:head>
	<title>管理后台 - HotSpot</title>
</svelte:head>

			<div class="space-y-6">
	<PageHeader title="管理后台" description="系统管理概览" />

	<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
		<Card>
			<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle class="text-sm font-medium">用户总数</CardTitle>
				<Users class="h-4 w-4 text-muted-foreground" />
			</CardHeader>
			<CardContent>
				<div class="text-2xl font-bold">--</div>
				<p class="text-xs text-muted-foreground">注册用户数量</p>
			</CardContent>
		</Card>

		<Card>
			<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle class="text-sm font-medium">群组总数</CardTitle>
				<UsersRound class="h-4 w-4 text-muted-foreground" />
			</CardHeader>
			<CardContent>
				<div class="text-2xl font-bold">--</div>
				<p class="text-xs text-muted-foreground">创建的群组数量</p>
			</CardContent>
		</Card>

		<Card>
			<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle class="text-sm font-medium">待处理举报</CardTitle>
				<Flag class="h-4 w-4 text-muted-foreground" />
			</CardHeader>
			<CardContent>
				<div class="text-2xl font-bold">--</div>
				<p class="text-xs text-muted-foreground">等待审核的举报</p>
			</CardContent>
		</Card>

		<Card>
			<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle class="text-sm font-medium">审计日志</CardTitle>
				<FileText class="h-4 w-4 text-muted-foreground" />
			</CardHeader>
			<CardContent>
				<div class="text-2xl font-bold">--</div>
				<p class="text-xs text-muted-foreground">今日操作记录</p>
			</CardContent>
		</Card>
	</div>

	<div class="grid gap-6 lg:grid-cols-2">
		<Card>
			<CardHeader>
				<CardTitle>快捷操作</CardTitle>
				<CardDescription>常用管理功能入口</CardDescription>
			</CardHeader>
			<CardContent class="grid gap-2">
				<a href="/admin/users" class="rounded-lg border p-4 transition-colors hover:bg-accent">
					<div class="flex items-center gap-3">
						<Users class="h-5 w-5" />
						<div>
							<p class="font-medium">用户管理</p>
							<p class="text-sm text-muted-foreground">搜索、封禁、解封用户</p>
						</div>
					</div>
				</a>
				<a href="/admin/reports" class="rounded-lg border p-4 transition-colors hover:bg-accent">
					<div class="flex items-center gap-3">
						<Flag class="h-5 w-5" />
						<div>
							<p class="font-medium">举报审核</p>
							<p class="text-sm text-muted-foreground">处理用户举报</p>
						</div>
					</div>
				</a>
			</CardContent>
		</Card>

		<Card>
			<CardHeader>
				<CardTitle>系统状态</CardTitle>
				<CardDescription>服务运行状态</CardDescription>
			</CardHeader>
			<CardContent>
				<div class="space-y-4">
					<div class="flex items-center justify-between">
						<span>API 服务</span>
						<span class="flex items-center gap-2 text-sm text-green-500">
							<span class="h-2 w-2 rounded-full bg-green-500"></span>
							正常
						</span>
					</div>
					<div class="flex items-center justify-between">
						<span>数据库</span>
						<span class="flex items-center gap-2 text-sm text-green-500">
							<span class="h-2 w-2 rounded-full bg-green-500"></span>
							正常
						</span>
					</div>
					<div class="flex items-center justify-between">
						<span>缓存服务</span>
						<span class="flex items-center gap-2 text-sm text-green-500">
							<span class="h-2 w-2 rounded-full bg-green-500"></span>
							正常
						</span>
					</div>
				</div>
			</CardContent>
		</Card>
	</div>
</div>

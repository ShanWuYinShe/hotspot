<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import Bell from 'lucide-svelte/icons/bell';
</script>

<svelte:head>
	<title>通知设置 - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-3xl space-y-5">
	<div class="rounded-3xl border border-border/60 bg-linear-to-br from-background via-card/85 to-muted/40 p-5 shadow-sm">
		<h1 class="text-2xl font-bold">通知设置</h1>
		<p class="mt-1 text-sm text-muted-foreground">管理你的通知偏好</p>
	</div>

	<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
		<CardHeader>
			<CardTitle class="flex items-center gap-2 text-base">
				<Bell class="h-5 w-5 text-muted-foreground" />
				功能开发中
			</CardTitle>
			<CardDescription>
				通知偏好设置功能正在开发中，敬请期待。当前所有通知类型默认开启。
			</CardDescription>
		</CardHeader>
		<CardContent class="pt-0">
			<p class="rounded-2xl border border-border/50 bg-background/65 px-4 py-3 text-sm text-muted-foreground">
				你可以在通知列表中查看和管理已收到的通知。
			</p>
		</CardContent>
	</Card>
</div>

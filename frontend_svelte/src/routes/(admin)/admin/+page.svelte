<script lang="ts">
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

	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
		{#each [
			{ icon: Users, label: '用户总数', value: '--', sub: '注册用户数量' },
			{ icon: UsersRound, label: '群组总数', value: '--', sub: '创建的群组数量' },
			{ icon: Flag, label: '待处理举报', value: '--', sub: '等待审核的举报' },
			{ icon: FileText, label: '审计日志', value: '--', sub: '今日操作记录' }
		] as stat}
			<div
				class="flex flex-col items-center justify-center rounded-[24px] py-5"
				style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
			>
				<div
					class="mb-2 flex items-center justify-center rounded-full"
					style="width: 36px; height: 36px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
				>
					<stat.icon size={16} strokeWidth={1.75} style="color: var(--neu-accent-primary);" />
				</div>
				<span style="font-size: 22px; color: var(--neu-text); font-weight: 700;">{stat.value}</span>
				<span style="font-size: 11px; color: var(--neu-text); font-weight: 500; margin-top: 2px;">{stat.label}</span>
				<span style="font-size: 10px; color: var(--neu-text-muted); margin-top: 2px;">{stat.sub}</span>
			</div>
		{/each}
	</div>

	<div class="grid gap-6 lg:grid-cols-2">
		<!-- 快捷操作 -->
		<div
			class="rounded-[24px] p-5"
			style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
		>
			<div class="mb-1" style="font-size: 16px; font-weight: 600; color: var(--neu-text);">快捷操作</div>
			<div class="mb-4" style="font-size: 12px; color: var(--neu-text-muted);">常用管理功能入口</div>
			<div style="height: 1px; background: var(--neu-border); margin-bottom: 16px;"></div>
			<div class="space-y-3">
				{#each [
					{ href: '/admin/users', icon: Users, title: '用户管理', desc: '搜索、封禁、解封用户' },
					{ href: '/admin/reports', icon: Flag, title: '举报审核', desc: '处理用户举报' }
				] as action}
					<a
						href={action.href}
						class="flex items-center gap-3 rounded-[18px] p-4 transition-all duration-200"
						style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border-pressed);"
					>
						<div
							class="flex items-center justify-center rounded-full"
							style="width: 36px; height: 36px; box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
						>
							<action.icon size={16} strokeWidth={1.75} style="color: var(--neu-accent-primary);" />
						</div>
						<div>
							<p style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{action.title}</p>
							<p style="font-size: 12px; color: var(--neu-text-muted);">{action.desc}</p>
						</div>
					</a>
				{/each}
			</div>
		</div>

		<!-- 系统状态 -->
		<div
			class="rounded-[24px] p-5"
			style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
		>
			<div class="mb-1" style="font-size: 16px; font-weight: 600; color: var(--neu-text);">系统状态</div>
			<div class="mb-4" style="font-size: 12px; color: var(--neu-text-muted);">服务运行状态</div>
			<div style="height: 1px; background: var(--neu-border); margin-bottom: 16px;"></div>
			<div class="space-y-4">
				{#each ['API 服务', '数据库', '缓存服务'] as service}
					<div
						class="flex items-center justify-between rounded-[18px] px-4 py-3"
						style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border-pressed);"
					>
						<span style="font-size: 14px; color: var(--neu-text);">{service}</span>
						<div class="flex items-center gap-2">
							<span style="width: 8px; height: 8px; border-radius: 50%; background: var(--neu-success);"></span>
							<span style="font-size: 13px; color: var(--neu-success);">正常</span>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</div>
</div>

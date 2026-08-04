<script lang="ts">
	import { Camera, User, FileText, Check } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import { getAccount, updateAccount } from '@/api';

	// 编辑资料：昵称/简介/兴趣标签（5 个上限）+ 注销按钮。

	const TAG_OPTIONS = ['摄影', '旅行', '音乐', '美食', '运动', '阅读', '艺术', '科技', '游戏', '户外', '时尚', '电影'];

	let nickname = $state('');
	let bio = $state('');
	let tags = $state<Set<string>>(new Set());
	let saved = $state(false);
	let avatar = $state('');
	let userId = $state('');

	$effect(() => {
		let cancelled = false;
		async function fetchAccount() {
			try {
				const res = await getAccount().catch(() => ({ code: -1, data: null }));
				if (cancelled) return;
				const data = res.code === 0 ? res.data : null;
				if (data) {
					// spec: AccountData.nickname, .bio, .tags, .avatar_url, .user_id
					nickname = data.nickname;
					bio = data.bio ?? '';
					avatar = data.avatar_url ?? '';
					userId = data.user_id;
					if (data.tags) tags = new Set(data.tags);
				}
			} finally {
				// ignore fetch error
			}
		}
		void fetchAccount();
		return () => {
			cancelled = true;
		};
	});

	const toggleTag = (t: string) => {
		const n = new Set(tags);
		n.has(t) ? n.delete(t) : n.add(t);
		tags = n;
	};

	const handleSave = async () => {
		try {
			// spec: updateAccount { nickname?, avatar_file_id?, bio?, tags?, gender? }
			await updateAccount({
				nickname: nickname || undefined,
				bio: bio || undefined,
				tags: tags.size > 0 ? Array.from(tags) : undefined
			});
		} catch {
			// ignore save error
		}
		saved = true;
		setTimeout(() => {
			saved = false;
			history.back();
		}, 800);
	};
</script>

<StandardPageLayout contentClassName="py-5 pb-10">
	{#snippet header()}
		<PageHeader title="编辑资料" onBack={() => history.back()} sticky={false}>
	{#snippet trailing()}
		<button
			type="button"
			class="hover-btn flex items-center gap-1.5 px-4 py-2 rounded-full outline-none transition-all text-[13px] {saved ? 'bg-[var(--status-online)]' : 'bg-[var(--accent-default)]'} text-white"
			style="box-shadow:0 4px 14px {saved ? 'color-mix(in srgb, var(--status-online) 35%, transparent)' : 'var(--glow-accent)'};border:none;"
			onclick={() => void handleSave()}
		>
			{#if saved}
				<Check style="width:14px;height:14px;color:var(--text-on-accent);" strokeWidth={2.5} />
			{:else}
				保存
			{/if}
		</button>
	{/snippet}
	</PageHeader>
{/snippet}

	<div class="animate-fade-in flex flex-col gap-5">
	<!-- Avatar section -->
	<div class="relative rounded-[24px] overflow-hidden" style="height:clamp(120px, 18vw, 160px);">
		<div class="absolute inset-0 bg-[var(--surface-primary)]"></div>
		<button
			type="button"
			class="hover-btn absolute top-3 right-3 flex items-center gap-1.5 rounded-full outline-none bg-[var(--surface-overlay-strong)] text-white text-[12px]"
			style="padding:6px 12px;backdrop-filter:blur(10px);border:1px solid rgba(255,255,255,0.25);"
		>
			<Camera style="width:13px;height:13px;" strokeWidth={1.75} />
			更换横幅
		</button>
	</div>

	<!-- Avatar edit -->
	<div class="flex items-center gap-4 -mt-8 ml-2">
		<div class="relative flex-shrink-0">
			<div class="rounded-[22px] overflow-hidden" style="width:72px;height:72px;border:3px solid var(--surface-primary);box-shadow:0 6px 20px rgba(0,0,0,0.22);">
				<img src={avatar} alt="avatar" class="w-full h-full object-cover" />
			</div>
			<button
				type="button"
				class="hover-btn absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none bg-[var(--accent-default)]"
				style="width:26px;height:26px;border:2px solid var(--surface-primary);box-shadow:0 2px 8px var(--glow-accent);"
			>
				<Camera style="width:12px;height:12px;color:var(--text-on-accent);" strokeWidth={2} />
			</button>
		</div>
		<div>
			<p class="text-[var(--text-primary)] text-[16px]">{nickname || '未设置昵称'}</p>
			{#if userId}
				<p class="text-[var(--text-secondary)] text-[12px] mt-0.5">@{userId}</p>
			{/if}
		</div>
	</div>

	<!-- Form fields — spec: nickname, bio, tags are the only user-editable profile fields. -->
	<div class="flex flex-col gap-1.5">
		<div class="flex items-center gap-1.5">
			<User style="width:13px;height:13px;color:var(--accent-default);" strokeWidth={1.75} />
			<label class="text-[var(--text-secondary)] text-[11px] uppercase tracking-[0.1em]" for="edit-nickname">昵称</label>
		</div>
		<div class="px-4 py-3 rounded-[18px] input-surface">
			<input
				id="edit-nickname"
				bind:value={nickname}
				oninput={(e) => (nickname = e.currentTarget.value.slice(0, 30))}
				placeholder="你的名字"
				class="w-full bg-transparent outline-none text-[var(--text-primary)] text-[14px]"
			/>
		</div>
	</div>

	<div class="flex flex-col gap-1.5">
		<div class="flex items-center gap-1.5">
			<FileText style="width:13px;height:13px;color:var(--accent-default);" strokeWidth={1.75} />
			<label class="text-[var(--text-secondary)] text-[11px] uppercase tracking-[0.1em]" for="edit-bio">个人简介</label>
		</div>
		<div class="px-4 py-3 rounded-[18px] input-surface">
			<div>
				<textarea
					id="edit-bio"
					bind:value={bio}
					oninput={(e) => (bio = e.currentTarget.value.slice(0, 150))}
					placeholder="介绍一下自己…"
					rows={3}
					class="w-full bg-transparent outline-none resize-none text-[var(--text-primary)] text-[14px] leading-[1.65]"
				></textarea>
				<div class="flex justify-end mt-1">
					<span class="{bio.length > 127 ? 'text-[var(--accent-default)]' : 'text-[var(--text-secondary)]'} text-[11px]">
						{bio.length}/150
					</span>
				</div>
			</div>
		</div>
	</div>

	<!-- Interest tags -->
	<fieldset class="flex flex-col gap-2.5 border-0 p-0 m-0 min-w-0">
		<legend class="flex items-center gap-1.5 text-[var(--text-secondary)] text-[11px] uppercase tracking-[0.1em]">
			兴趣标签
			<span class="opacity-60">({tags.size}/5)</span>
		</legend>
		<div class="flex flex-wrap gap-2">
			{#each TAG_OPTIONS as t (t)}
				{@const active = tags.has(t)}
				{@const disabled = !active && tags.size >= 5}
				<button
					type="button"
					class="hover-btn px-3.5 py-1.5 rounded-full outline-none transition-all text-[12px] {active ? 'bg-[var(--accent-default)] text-white' : 'bg-[var(--surface-primary)] text-[var(--text-secondary)]'}"
					style={active
						? 'box-shadow:0 3px 10px var(--glow-accent);border:none;'
						: 'border:1px solid var(--border-default);opacity:0.4;box-shadow:3px 3px 6px var(--elevation-shadow-dark), -3px -3px 6px var(--elevation-shadow-light);'}
					onclick={() => !disabled && toggleTag(t)}
				>
					#{t}
				</button>
			{/each}
		</div>
	</fieldset>

	<!-- Danger zone -->
	<div class="mt-3">
		<p class="uppercase mb-3 text-[var(--text-secondary)] text-[10px] tracking-[0.14em]">账号</p>
		<button
			type="button"
			class="hover-btn w-full text-left px-4 py-3.5 rounded-[18px] outline-none bg-[var(--surface-primary)] text-[var(--status-danger)] text-[14px]"
			style="box-shadow:5px 5px 12px var(--elevation-shadow-dark), -5px -5px 12px var(--elevation-shadow-light);border:1px solid var(--border-default);"
		>
			注销账号
		</button>
	</div>
	</div>
</StandardPageLayout>

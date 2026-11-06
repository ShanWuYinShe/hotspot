import { goto } from '$app/navigation';
import { logout } from '@/stores/auth.svelte';
import { showToast } from '@/stores/toast.svelte';
import { getAccount, updateAccount, deleteAccount, netError } from '@/api';
import type { Gender } from '@/api/types';
import { uploadMediaFile } from '@/lib/upload';
import { ROUTES } from '@/lib/routes';

export const MAX_BIO = 150;
export const BIO_WARN = Math.floor(MAX_BIO * 0.85);

export const TAG_OPTIONS = ['摄影', '旅行', '音乐', '美食', '运动', '阅读', '艺术', '科技', '游戏', '户外', '时尚', '电影'];

export const GENDER_OPTIONS: { id: Gender; label: string }[] = [
	{ id: 'male', label: '男' },
	{ id: 'female', label: '女' },
	{ id: 'other', label: '其他' }
];

type Baseline = { nickname: string; bio: string; tags: string[]; gender: Gender | null };

export function useEditProfileData() {
	let nickname = $state('');
	let bio = $state('');
	let tags = $state<Set<string>>(new Set());
	const extraTags = $derived([...tags].filter((t) => !TAG_OPTIONS.includes(t)));
	let gender = $state<Gender | null>(null);

	let saved = $state(false);
	let savedTimer: number | undefined;
	$effect(() => () => window.clearTimeout(savedTimer));

	let saving = $state(false);
	let uploadingAvatar = $state(false);
	let deleteConfirmOpen = $state(false);
	let deleteConfirmText = $state('');
	let deleting = $state(false);

	let avatar = $state('');
	let userId = $state('');
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let nicknameError = $state<string | null>(null);
	let formError = $state<string | null>(null);
	let needsAuth = $state(false);
	let reloadKey = $state(0);

	let baseline = $state<Baseline | null>(null);

	const dirty = $derived.by(() => {
		if (!baseline) return false;
		const cur = [...tags].sort();
		const base = [...baseline.tags].sort();
		return (
			nickname !== baseline.nickname ||
			bio !== baseline.bio ||
			gender !== baseline.gender ||
			cur.length !== base.length ||
			cur.some((t, i) => t !== base[i])
		);
	});

	const islandMeta = $derived(saving ? '保存中…' : saved ? '已保存' : dirty ? '有未保存的更改' : undefined);

	$effect(() => {
		void reloadKey;
		let cancelled = false;

		async function fetchAccount() {
			loading = true;
			loadError = null;
			needsAuth = false;
			try {
				const res = await getAccount().catch(() => netError(null));
				if (cancelled) return;
				if (res.code === 10100) {
					needsAuth = true;
					return;
				}
				const data = res.code === 0 ? res.data : null;
				if (data) {
					nickname = data.nickname;
					bio = data.bio ?? '';
					avatar = data.avatar_url ?? '';
					userId = data.user_id;
					if (data.tags) tags = new Set(data.tags);
					gender = data.gender ?? null;
					baseline = {
						nickname: data.nickname,
						bio: data.bio ?? '',
						tags: data.tags ? [...data.tags] : [],
						gender: data.gender ?? null
					};
				} else {
					loadError = res.message ?? '加载资料失败';
				}
			} catch (err) {
				if (!cancelled) loadError = err instanceof Error ? err.message : '加载资料失败';
			} finally {
				if (!cancelled) loading = false;
			}
		}

		void fetchAccount();
		return () => {
			cancelled = true;
		};
	});

	const toggleTag = (t: string) => {
		const n = new Set(tags);
		if (n.has(t)) n.delete(t);
		else n.add(t);
		tags = n;
	};

	const handleSave = async () => {
		if (saving) return;
		const trimmedNickname = nickname.trim();
		if (!trimmedNickname) {
			nicknameError = '昵称不能为空';
			document.getElementById('edit-nickname')?.focus();
			return;
		}
		nicknameError = null;
		formError = null;
		saving = true;
		try {
			const res = await updateAccount({
				nickname: trimmedNickname,
				bio: bio.trim(),
				tags: Array.from(tags),
				...(gender ? { gender } : {})
			});
			if (res.code !== 0) throw new Error(res.message || '保存失败');
			baseline = { nickname: trimmedNickname, bio: bio.trim(), tags: [...tags], gender };
			saved = true;
			showToast('已保存', 'success');
			savedTimer = window.setTimeout(() => {
				saved = false;
				void goto(ROUTES.me);
			}, 800);
		} catch (err) {
			formError = err instanceof Error && err.message ? err.message : '保存失败，请重试';
		} finally {
			saving = false;
		}
	};

	const handleAvatarUpload = async (file: File) => {
		if (uploadingAvatar) return;
		uploadingAvatar = true;
		try {
			const fileId = await uploadMediaFile(file, 'avatar');
			await updateAccount({ avatar_file_id: fileId });
			const res = await getAccount().catch(() => null);
			if (res?.code === 0 && res.data?.avatar_url) avatar = res.data.avatar_url;
			showToast('头像已更新', 'success');
		} catch {
			showToast('头像上传失败，请重试', 'danger');
		} finally {
			uploadingAvatar = false;
		}
	};

	const confirmDelete = async () => {
		if (deleteConfirmText !== 'DELETE_MY_ACCOUNT' || deleting) return;
		deleting = true;
		try {
			await deleteAccount({ confirmation: deleteConfirmText });
			await logout();
			void goto(ROUTES.home, { replaceState: true });
		} catch {
			deleting = false;
			showToast('注销失败，请重试', 'danger');
		}
	};

	const refresh = () => {
		reloadKey++;
	};

	return {
		get nickname() { return nickname; },
		set nickname(v: string) { nickname = v; },
		get bio() { return bio; },
		set bio(v: string) { bio = v; },
		get tags() { return tags; },
		get extraTags() { return extraTags; },
		get gender() { return gender; },
		set gender(v: Gender | null) { gender = v; },
		get saved() { return saved; },
		get saving() { return saving; },
		get uploadingAvatar() { return uploadingAvatar; },
		get deleteConfirmOpen() { return deleteConfirmOpen; },
		set deleteConfirmOpen(v: boolean) { deleteConfirmOpen = v; },
		get deleteConfirmText() { return deleteConfirmText; },
		set deleteConfirmText(v: string) { deleteConfirmText = v; },
		get deleting() { return deleting; },
		get avatar() { return avatar; },
		get userId() { return userId; },
		get loading() { return loading; },
		get loadError() { return loadError; },
		get nicknameError() { return nicknameError; },
		set nicknameError(v: string | null) { nicknameError = v; },
		get formError() { return formError; },
		set formError(v: string | null) { formError = v; },
		get needsAuth() { return needsAuth; },
		get dirty() { return dirty; },
		get islandMeta() { return islandMeta; },
		toggleTag,
		handleSave,
		handleAvatarUpload,
		confirmDelete,
		refresh
	};
}

export type EditProfileData = ReturnType<typeof useEditProfileData>;

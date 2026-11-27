import { goto } from '$app/navigation';
import { auth, logout, requireAuth } from '@/stores/auth.svelte';
import { showToast } from '@/stores/toast.svelte';
import { getAccount, getSettings, updateSettings, netError } from '@/api';
import { ROUTES } from '@/lib/routes';

const NOTIF_KEY = 'hotspot:notif';

const readLocal = (key: string, fallback: boolean) => {
	try {
		const v = localStorage.getItem(key);
		return v === null ? fallback : v === '1';
	} catch {
		return fallback;
	}
};

const writeLocal = (key: string, value: boolean) => {
	try {
		localStorage.setItem(key, value ? '1' : '0');
	} catch {}
};

export function useSettingsData() {
	let notif = $state(readLocal(NOTIF_KEY, true));
	let privateProfile = $state(false);
	let locationSharing = $state(true);
	let socialListsVisible = $state(true);
	let groupsVisible = $state(true);
	let reactedMomentsVisible = $state(true);
	let allowFollow = $state(true);
	let nearbyMomentNotify = $state(false);
	let accountName = $state('');
	let accountEmail = $state('');
	let accountAvatar = $state('');
	let loading = $state(false);
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);
	let confirmLogoutOpen = $state(false);
	let loggingOut = $state(false);

	const toggleNearbyMomentNotify = () => {
		if (!auth.isAuthenticated) {
			requireAuth({ action: '修改附近新瞬间通知设置', run: toggleNearbyMomentNotify });
			return;
		}
		const next = !nearbyMomentNotify;
		nearbyMomentNotify = next;
		// 与另两个开关同口径：业务码非 0 也要回滚（只 catch 传输异常会留下错误状态）
		void updateSettings({ settings: { nearby_moment_notify: next } })
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message ?? '设置失败');
			})
			.catch(() => {
				nearbyMomentNotify = !next;
				showToast('设置失败，请稍后重试', 'danger');
			});
	};

	const toggleLocationSharing = () => {
		if (!auth.isAuthenticated) {
			requireAuth({ action: '修改位置共享设置', run: toggleLocationSharing });
			return;
		}
		const next = !locationSharing;
		locationSharing = next;
		void updateSettings({ settings: { location_visible: next } })
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message ?? '设置失败');
			})
			.catch(() => {
				locationSharing = !next;
				showToast('设置失败，请稍后重试', 'danger');
			});
	};

	const togglePrivateProfile = () => {
		if (!auth.isAuthenticated) {
			requireAuth({ action: '修改私密账号设置', run: togglePrivateProfile });
			return;
		}
		const next = !privateProfile;
		privateProfile = next;
		void updateSettings({ settings: { privacy_level: next ? 'private' : 'public' } })
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message ?? '设置失败');
			})
			.catch(() => {
				privateProfile = !next;
				showToast('设置失败，请稍后重试', 'danger');
			});
	};

	const toggleSocialListsVisible = () => {
		if (!auth.isAuthenticated) {
			requireAuth({ action: '修改社交列表可见性设置', run: toggleSocialListsVisible });
			return;
		}
		const next = !socialListsVisible;
		socialListsVisible = next;
		void updateSettings({ settings: { social_lists_visible: next } })
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message ?? '设置失败');
			})
			.catch(() => {
				socialListsVisible = !next;
				showToast('设置失败，请稍后重试', 'danger');
			});
	};

	const toggleGroupsVisible = () => {
		if (!auth.isAuthenticated) {
			requireAuth({ action: '修改群组列表可见性设置', run: toggleGroupsVisible });
			return;
		}
		const next = !groupsVisible;
		groupsVisible = next;
		void updateSettings({ settings: { groups_visible: next } })
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message ?? '设置失败');
			})
			.catch(() => {
				groupsVisible = !next;
				showToast('设置失败，请稍后重试', 'danger');
			});
	};

	const toggleReactedMomentsVisible = () => {
		if (!auth.isAuthenticated) {
			requireAuth({ action: '修改反应瞬间可见性设置', run: toggleReactedMomentsVisible });
			return;
		}
		const next = !reactedMomentsVisible;
		reactedMomentsVisible = next;
		void updateSettings({ settings: { reacted_moments_visible: next } })
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message ?? '设置失败');
			})
			.catch(() => {
				reactedMomentsVisible = !next;
				showToast('设置失败，请稍后重试', 'danger');
			});
	};

	const toggleAllowFollow = () => {
		if (!auth.isAuthenticated) {
			requireAuth({ action: '修改允许关注设置', run: toggleAllowFollow });
			return;
		}
		const next = !allowFollow;
		allowFollow = next;
		void updateSettings({ settings: { allow_follow: next } })
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message ?? '设置失败');
			})
			.catch(() => {
				allowFollow = !next;
				showToast('设置失败，请稍后重试', 'danger');
			});
	};

	const toggleNotif = () => {
		notif = !notif;
		writeLocal(NOTIF_KEY, notif);
	};

	$effect(() => {
		void reloadKey;
		let cancelled = false;

		async function fetchData() {
			if (!auth.isAuthenticated) {
				accountName = '';
				accountEmail = '';
				accountAvatar = '';
				loading = false;
				loadError = null;
				return;
			}
			try {
				loading = true;
				loadError = null;
				const [accountRes, settingsRes] = await Promise.all([
					getAccount().catch(() => netError(null)),
					getSettings().catch(() => netError(null))
				]);
				if (cancelled) return;

				const accountData = accountRes.code === 0 ? accountRes.data : null;
				const settingsData = settingsRes.code === 0 ? settingsRes.data : null;

				if (accountData) {
					accountName = accountData.nickname ?? '';
					accountEmail = accountData.email ?? '';
					accountAvatar = accountData.avatar_url ?? '';
				}

				if (settingsData) {
					if (settingsData.location_visible !== undefined) locationSharing = settingsData.location_visible;
					if (settingsData.nearby_moment_notify !== undefined) nearbyMomentNotify = settingsData.nearby_moment_notify;
					if (settingsData.privacy_level !== undefined) privateProfile = settingsData.privacy_level === 'private';
					if (settingsData.social_lists_visible !== undefined) socialListsVisible = settingsData.social_lists_visible;
					if (settingsData.groups_visible !== undefined) groupsVisible = settingsData.groups_visible;
					if (settingsData.reacted_moments_visible !== undefined) reactedMomentsVisible = settingsData.reacted_moments_visible;
					if (settingsData.allow_follow !== undefined) allowFollow = settingsData.allow_follow;
				}

				if (accountRes.code !== 0 && settingsRes.code !== 0 && accountRes.code !== 401) {
					loadError = accountRes.message ?? '加载失败，请稍后重试';
				}
			} catch {
				if (!cancelled) loadError = '加载失败，请稍后重试';
			} finally {
				if (!cancelled) loading = false;
			}
		}

		void fetchData();
		return () => {
			cancelled = true;
		};
	});

	const handleLogout = async () => {
		if (loggingOut) return;
		loggingOut = true;
		try {
			await logout();
			confirmLogoutOpen = false;
			void goto(ROUTES.home);
		} catch {
			showToast('退出登录失败，请重试', 'danger');
		} finally {
			loggingOut = false;
		}
	};

	const refresh = () => {
		reloadKey++;
	};

	return {
		get notif() { return notif; },
		get privateProfile() { return privateProfile; },
		get locationSharing() { return locationSharing; },
		get socialListsVisible() { return socialListsVisible; },
		get groupsVisible() { return groupsVisible; },
		get reactedMomentsVisible() { return reactedMomentsVisible; },
		get allowFollow() { return allowFollow; },
		get nearbyMomentNotify() { return nearbyMomentNotify; },
		get accountName() { return accountName; },
		get accountEmail() { return accountEmail; },
		get accountAvatar() { return accountAvatar; },
		get loading() { return loading; },
		get loadError() { return loadError; },
		get confirmLogoutOpen() { return confirmLogoutOpen; },
		set confirmLogoutOpen(v: boolean) { confirmLogoutOpen = v; },
		get loggingOut() { return loggingOut; },
		toggleNearbyMomentNotify,
		toggleLocationSharing,
		togglePrivateProfile,
		toggleSocialListsVisible,
		toggleGroupsVisible,
		toggleReactedMomentsVisible,
		toggleAllowFollow,
		toggleNotif,
		handleLogout,
		refresh
	};
}

export type SettingsData = ReturnType<typeof useSettingsData>;

// 群详情页 tab 的 URL 契约：?tab=overview|members|announcements&memberTab=members|applications|invites|banned
// 刷新/返回保持当前位置，也支持直达「成员→申请」。非法值一律回落到默认档。
export type GroupTab = 'overview' | 'members' | 'announcements';
export type MemberSubTab = 'members' | 'applications' | 'invites' | 'banned';

const GROUP_TABS: GroupTab[] = ['overview', 'members', 'announcements'];
const MEMBER_TABS: MemberSubTab[] = ['members', 'applications', 'invites', 'banned'];

export function parseGroupTab(raw: string | null | undefined): GroupTab {
	const value = raw ?? '';
	return (GROUP_TABS as string[]).includes(value) ? (value as GroupTab) : 'overview';
}

export function parseMemberTab(raw: string | null | undefined): MemberSubTab {
	const value = raw ?? '';
	return (MEMBER_TABS as string[]).includes(value) ? (value as MemberSubTab) : 'members';
}

/** 计算需要写入 URL 的 query 片段；值为 null 表示删除该参数。 */
export function groupTabQuery(
	tab: GroupTab,
	memberTab: MemberSubTab
): { tab: string | null; memberTab: string | null } {
	return {
		tab: tab === 'overview' ? null : tab,
		// 成员子档只在「成员」主档下有意义，切走时清理，避免残留成孤儿参数
		memberTab: tab === 'members' && memberTab !== 'members' ? memberTab : null
	};
}

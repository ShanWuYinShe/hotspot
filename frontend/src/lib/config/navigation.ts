/**
 * 全站导航事实源（单一来源）
 *
 * 桌面侧边栏、移动/平板导航抽屉、底部主标签全部从这里派生，确保
 * 「同一设备场景 + 主题」下所有页面（含管理后台）路由集合与风格一致。
 *
 * - 标签用 i18n key（`labelKey`），组件内用 `t(item.labelKey)` 解析，保持响应式。
 * - 图标统一 `lucide-svelte`，类型对齐既有写法（`typeof Home`）。
 */
import Home from 'lucide-svelte/icons/home';
import Compass from 'lucide-svelte/icons/compass';
import Image from 'lucide-svelte/icons/image';
import UsersRound from 'lucide-svelte/icons/users-round';
import MessageCircle from 'lucide-svelte/icons/message-circle';
import Bell from 'lucide-svelte/icons/bell';
import Heart from 'lucide-svelte/icons/heart';
import UserCheck from 'lucide-svelte/icons/user-check';
import UserPlus from 'lucide-svelte/icons/user-plus';
import MapPin from 'lucide-svelte/icons/map-pin';
import Menu from 'lucide-svelte/icons/menu';
import LayoutDashboard from 'lucide-svelte/icons/layout-dashboard';
import Users from 'lucide-svelte/icons/users';
import Megaphone from 'lucide-svelte/icons/megaphone';
import FileText from 'lucide-svelte/icons/file-text';
import Flag from 'lucide-svelte/icons/flag';
import Shield from 'lucide-svelte/icons/shield';
import Settings from 'lucide-svelte/icons/settings';
import type { TranslationKey } from '$lib/i18n';

export type NavIcon = typeof Home;

/** 角标来源：交由组件结合对应 store 解析数量。 */
export type NavBadge = 'notifications' | 'messages' | 'inbox';

export interface NavItem {
  href: string;
  labelKey: TranslationKey;
  icon: NavIcon;
  /** 需要登录才能进入；未登录时显示「登录」提示而非直接跳转。 */
  requiresAuth?: boolean;
  /** 仅精确匹配该路径才高亮（如 `/`、`/moments`、`/admin`）。 */
  exact?: boolean;
  badge?: NavBadge;
}

/** 底部主标签：可为路由跳转（`href`）或特殊动作（`action`）。 */
export interface BottomTab {
  key: string;
  labelKey: TranslationKey;
  icon: NavIcon;
  href?: string;
  action?: 'menu';
  requiresAuth?: boolean;
  exact?: boolean;
  badge?: NavBadge;
}

/** 浏览 / 发现类顶层目的地。 */
export const discoverNavItems: NavItem[] = [
  { href: '/', labelKey: 'sidebar.home', icon: Home, exact: true },
  { href: '/discover', labelKey: 'nav.discover', icon: Compass },
  { href: '/users/nearby', labelKey: 'sidebar.nearbyPeople', icon: UsersRound },
  { href: '/groups/nearby', labelKey: 'sidebar.nearbyGroups', icon: UsersRound },
  { href: '/moments/nearby', labelKey: 'sidebar.nearbyMoments', icon: Image }
];

/** 个人中心目的地（均需登录）。 */
export const personalNavItems: NavItem[] = [
  {
    href: '/chat',
    labelKey: 'sidebar.messages',
    icon: MessageCircle,
    requiresAuth: true,
    badge: 'messages'
  },
  {
    href: '/notifications',
    labelKey: 'sidebar.notifications',
    icon: Bell,
    requiresAuth: true,
    badge: 'notifications'
  },
  { href: '/moments', labelKey: 'sidebar.myMoments', icon: Image, requiresAuth: true, exact: true },
  {
    href: '/groups',
    labelKey: 'sidebar.myGroups',
    icon: UsersRound,
    requiresAuth: true,
    exact: true
  },
  { href: '/friends', labelKey: 'sidebar.friends', icon: Heart, requiresAuth: true },
  { href: '/follows', labelKey: 'sidebar.following', icon: UserCheck, requiresAuth: true },
  { href: '/followers', labelKey: 'sidebar.followers', icon: UserPlus, requiresAuth: true }
];

/** 管理后台各分区。 */
export const adminNavItems: NavItem[] = [
  { href: '/admin', labelKey: 'admin.overview', icon: LayoutDashboard, exact: true },
  { href: '/admin/users', labelKey: 'admin.users', icon: Users },
  { href: '/admin/groups', labelKey: 'admin.groups', icon: UsersRound },
  { href: '/admin/announcements', labelKey: 'admin.announcements', icon: Megaphone },
  { href: '/admin/audit', labelKey: 'admin.audit', icon: FileText },
  { href: '/admin/reports', labelKey: 'admin.reports', icon: Flag },
  { href: '/admin/rbac', labelKey: 'admin.rbac', icon: Shield },
  { href: '/admin/config', labelKey: 'admin.config', icon: Settings }
];

/**
 * 移动 / 平板底部主标签：3 个高频目的地 + 一个「菜单」入口。
 * 「菜单」打开导航抽屉（NavDrawer），抽屉内暴露与桌面侧边栏一致的全部路由，
 * 从而保证所有设备场景下路由集合一致。
 */
export const bottomTabs: BottomTab[] = [
  { key: 'nearby', href: '/', labelKey: 'nav.nearby', icon: MapPin, exact: true },
  { key: 'discover', href: '/discover', labelKey: 'nav.discover', icon: Compass },
  {
    key: 'messages',
    href: '/chat',
    labelKey: 'nav.messagesTab',
    icon: MessageCircle,
    requiresAuth: true,
    badge: 'inbox'
  },
  { key: 'menu', action: 'menu', labelKey: 'nav.menu', icon: Menu }
];

/** 统一的高亮判定：`exact` 仅精确匹配，否则匹配自身或其子路径。 */
export function isNavActive(
  href: string | undefined,
  exact: boolean | undefined,
  pathname: string
): boolean {
  if (!href) return false;
  if (exact) return pathname === href;
  return pathname === href || pathname.startsWith(`${href}/`);
}

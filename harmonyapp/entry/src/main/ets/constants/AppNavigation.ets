/**
 * 全站导航事实源（对齐前端 navigation.ts）
 *
 * 桌面侧边栏、移动 / 平板导航的导航集合均从此派生，确保「同一设备场景」下
 * 路由集合与图标一致。承前端约定：同一导航面内图标唯一表义。
 *
 * - isTab=true：核心 tab，在 DesktopLayout main content 区切换 currentRoute（tabRoute）
 * - isTab=false：钻取页，点击 push 全屏页（pageName），带返回
 * - openNotificationCenter=true：通知项，点击打开通知中心 overlay（而非 push）
 *
 * 图标方案（确保桌面侧边栏内唯一）：
 * 首页 house / 发现 compass / 附近的人 person / 附近的群组 person_2(group) / 消息 message /
 * 通知 bell / 我的瞬间 picture / 我的群组 person_2 / 关系 heart / 我的 person。
 */
import { AppIcon } from '../components/SystemIcon';
import { PageNames } from './AppConstants';

export interface AppNavItem {
  label: string;
  icon: AppIcon;
  /** 核心 tab 的 currentRoute 值（isTab=true 时用） */
  tabRoute: string;
  /** push 目标 PageNames（isTab=false 且 openNotificationCenter=false 时用） */
  pageName: string;
  isTab: boolean;
  requiresAuth: boolean;
  badge: 'none' | 'messages' | 'notifications';
  /** 点击打开通知中心 overlay */
  openNotificationCenter: boolean;
}

/** 桌面侧边栏发现组（顶层目的地） */
export const DESKTOP_DISCOVER_NAV: AppNavItem[] = [
  {
    label: '首页', icon: AppIcon.HOME, tabRoute: 'nearby', pageName: '',
    isTab: true, requiresAuth: false, badge: 'none', openNotificationCenter: false
  },
  {
    label: '发现', icon: AppIcon.COMPASS, tabRoute: 'discover', pageName: '',
    isTab: true, requiresAuth: false, badge: 'none', openNotificationCenter: false
  },
  {
    label: '附近的人', icon: AppIcon.PERSON, tabRoute: '', pageName: PageNames.NEARBY_USERS,
    isTab: false, requiresAuth: false, badge: 'none', openNotificationCenter: false
  },
  {
    label: '附近的群组', icon: AppIcon.GROUP, tabRoute: '', pageName: PageNames.NEARBY_GROUPS,
    isTab: false, requiresAuth: false, badge: 'none', openNotificationCenter: false
  }
];

/** 桌面侧边栏个人中心组（均需登录） */
export const DESKTOP_PERSONAL_NAV: AppNavItem[] = [
  {
    label: '消息', icon: AppIcon.MESSAGE, tabRoute: 'messages', pageName: '',
    isTab: true, requiresAuth: true, badge: 'messages', openNotificationCenter: false
  },
  {
    label: '通知', icon: AppIcon.NOTIFICATION, tabRoute: '', pageName: '',
    isTab: false, requiresAuth: true, badge: 'notifications', openNotificationCenter: true
  },
  {
    label: '我的瞬间', icon: AppIcon.PHOTO, tabRoute: '', pageName: PageNames.MOMENT,
    isTab: false, requiresAuth: true, badge: 'none', openNotificationCenter: false
  },
  {
    label: '我的群组', icon: AppIcon.GROUP, tabRoute: '', pageName: PageNames.GROUP,
    isTab: false, requiresAuth: true, badge: 'none', openNotificationCenter: false
  },
  {
    label: '关系', icon: AppIcon.HEART, tabRoute: '', pageName: PageNames.RELATIONSHIP_HUB,
    isTab: false, requiresAuth: true, badge: 'none', openNotificationCenter: false
  }
];

/** 移动 / 平板底栏 4 tab（对齐前端 bottomTabs） */
export interface AppBottomTab {
  label: string;
  icon: AppIcon;
  /** 底栏 tab 对应的 PhoneLayout tab 索引（0=首页 1=发现 3=消息 4=我的） */
  index: number;
  requiresAuth: boolean;
  badge: 'none' | 'inbox';
}

export const BOTTOM_TABS: AppBottomTab[] = [
  { label: '首页', icon: AppIcon.HOME, index: 0, requiresAuth: false, badge: 'none' },
  { label: '发现', icon: AppIcon.COMPASS, index: 1, requiresAuth: false, badge: 'none' },
  { label: '消息', icon: AppIcon.MESSAGE, index: 3, requiresAuth: true, badge: 'inbox' },
  { label: '我的', icon: AppIcon.PERSON, index: 4, requiresAuth: true, badge: 'none' }
];

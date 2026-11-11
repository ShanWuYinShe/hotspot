/**
 * 全局路由管理器
 * 封装 NavPathStack 操作，提供统一的导航能力。
 * 任何组件（包括非 NavDestination 的 Tab 内嵌组件）均可通过静态方法导航。
 */

import { hilog } from '@kit.PerformanceAnalysisKit';
import { PageNames } from '../../constants/AppConstants';

/** 登录页参数 */
export class LoginRouteParam {
  reason: string = '';
  action: string = '';
  redirectUrl: string = '';
  restoreAction: string = '';
  restoreTargetId: string = '';
  restoreTitle: string = '';
}

/** 聊天页参数 */
export class ChatRouteParam {
  chatId: string = '';
  chatType: string = 'private_chat';
  title: string = '聊天';
  targetId: string = '';
  /** 对端在线状态文本（在线/离线；空串不显示，来自会话列表 presence） */
  presenceText: string = '';
}

/** 群组详情页参数 */
export class GroupDetailRouteParam {
  groupId: string = '';
}

/** 瞬间详情页参数 */
export class MomentDetailRouteParam {
  momentId: string = '';
}

/** 用户资料页参数 */
export class ProfileRouteParam {
  userId: string = '';
}

/** 地图选点页参数 */
export class LocationPickerRouteParam {
  /** 'anchor' = 设置浏览锚点（默认），'location_pick' = 为创建群组/动态选取位置 */
  mode: string = 'anchor';
}

/** 图片预览页参数 */
export class PhotoPreviewRouteParam {
  momentId: string = '';
  images: string[] = [];
  index: number = 0;
}

/** 搜索页参数 */
export class SearchRouteParam {
  keyword: string = '';
  /** 初始 tab 索引：0=全部 1=用户 2=动态 3=话题 4=群组（对齐前端搜索 Tab） */
  initialTab: number = 0;
}

/** 公告页参数 */
export class AnnouncementsRouteParam {
  items: Object[] = [];
  focusAnnouncementId: string = '';
}

/** 编辑瞬间页参数 */
export class EditMomentRouteParam {
  moment_id: string = '';
  description: string = '';
  tags: string[] = [];
  existingImageUrls: string[] = [];
  privacy_level: string = 'public';
  location_name: string = '';
  latitude: number = 0;
  longitude: number = 0;
}

/** 分享解析页参数 */
export class ShareResolutionRouteParam {
  share_code: string = '';
}

const DOMAIN_NUMBER: number = 0xFF20;
const TAG: string = 'AppRouter';

export class AppRouter {
  private static pathStack: NavPathStack | null = null;

  /**
   * 推入页中仍保留底部 Dock 的页面（对齐 web：附近域叶子页底部导航不变，
   * 搜索/详情/设置等全屏叶子页隐藏底部导航）。
   */
  private static readonly DOCK_VISIBLE_PAGES: string[] = [
    PageNames.NEARBY_USERS,
    PageNames.NEARBY_MOMENTS,
    PageNames.NEARBY_GROUPS
  ];

  /** 初始化路由（在 Index.ets aboutToAppear 中调用） */
  static init(stack: NavPathStack): void {
    AppRouter.pathStack = stack;
    hilog.info(DOMAIN_NUMBER, TAG, 'AppRouter 初始化完成');
  }

  /**
   * 推入页均为全屏叶子页（web 移动端叶子页不展示底部导航），隐藏悬浮 Dock，
   * 避免 Dock 被叶子页透明底透出后仍显示但点击被页面拦截。
   */
  /** 公开刷新 Dock 可见性：系统返回（手势/返回键）绕过 AppRouter.pop 时由 Navigation 拦截回调调用。 */
  static refreshDockVisibility(): void {
    AppRouter.syncDockVisibility();
  }

  private static syncDockVisibility(): void {
    const stack = AppRouter.pathStack;
    if (!stack) {
      return;
    }
    const names: string[] = stack.getAllPathName();
    const top: string = names.length > 0 ? names[names.length - 1] : '';
    const keepDock: boolean = top === '' || AppRouter.DOCK_VISIBLE_PAGES.indexOf(top) >= 0;
    AppStorage.setOrCreate('hideMainTabBar', !keepDock);
  }

  /** 压入新页面 */
  /** 读取当前栈顶路由参数（推入页在 build 前同步判定自身模式用） */
  static getTopParam(): Object | undefined {
    if (!AppRouter.pathStack) {
      return undefined;
    }
    const names = AppRouter.pathStack.getAllPathName();
    if (names.length === 0) {
      return undefined;
    }
    return AppRouter.pathStack.getParamByIndex(names.length - 1) as Object | undefined;
  }

  static push(name: string, param?: Object): void {
    if (!AppRouter.pathStack) {
      hilog.error(DOMAIN_NUMBER, TAG, `push 失败：NavPathStack 未初始化（目标: ${name}）`);
      return;
    }
    AppRouter.pathStack.pushPathByName(name, param);
    AppRouter.syncDockVisibility();
  }

  /** 返回上一页，可附带返回结果 */
  static pop(result?: Object): void {
    if (!AppRouter.pathStack) {
      hilog.error(DOMAIN_NUMBER, TAG, 'pop 失败：NavPathStack 未初始化');
      return;
    }
    AppRouter.pathStack.pop(result);
    AppRouter.syncDockVisibility();
  }

  /** 返回到指定页面 */
  static popToName(name: string): void {
    if (!AppRouter.pathStack) {
      hilog.error(DOMAIN_NUMBER, TAG, `popToName 失败：NavPathStack 未初始化（目标: ${name}）`);
      return;
    }
    AppRouter.pathStack.popToName(name);
    AppRouter.syncDockVisibility();
  }

  /** 替换当前页面 */
  static replacePath(name: string, param?: Object): void {
    if (!AppRouter.pathStack) {
      hilog.error(DOMAIN_NUMBER, TAG, `replacePath 失败：NavPathStack 未初始化（目标: ${name}）`);
      return;
    }
    AppRouter.pathStack.replacePath({ name: name, param: param });
    AppRouter.syncDockVisibility();
  }

  /** 清空导航栈（回到根视图，即 Tabs） */
  static clear(): void {
    if (!AppRouter.pathStack) {
      hilog.error(DOMAIN_NUMBER, TAG, 'clear 失败：NavPathStack 未初始化');
      return;
    }
    AppRouter.pathStack.clear();
  }

  /** 获取 NavPathStack 实例（高级用途） */
  static getPathStack(): NavPathStack | null {
    return AppRouter.pathStack;
  }
}

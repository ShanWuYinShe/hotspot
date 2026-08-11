/**
 * 全局路由管理器
 * 封装 NavPathStack 操作，提供统一的导航能力。
 * 任何组件（包括非 NavDestination 的 Tab 内嵌组件）均可通过静态方法导航。
 */

import { hilog } from '@kit.PerformanceAnalysisKit';

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
}

/** 群组详情页参数 */
export class GroupDetailRouteParam {
  groupId: string = '';
}

/** 动态详情页参数 */
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

  /** 初始化路由（在 Index.ets aboutToAppear 中调用） */
  static init(stack: NavPathStack): void {
    AppRouter.pathStack = stack;
    hilog.info(DOMAIN_NUMBER, TAG, 'AppRouter 初始化完成');
  }

  /** 压入新页面 */
  static push(name: string, param?: Object): void {
    if (!AppRouter.pathStack) {
      hilog.error(DOMAIN_NUMBER, TAG, `push 失败：NavPathStack 未初始化（目标: ${name}）`);
      return;
    }
    AppRouter.pathStack.pushPathByName(name, param);
  }

  /** 返回上一页，可附带返回结果 */
  static pop(result?: Object): void {
    if (!AppRouter.pathStack) {
      hilog.error(DOMAIN_NUMBER, TAG, 'pop 失败：NavPathStack 未初始化');
      return;
    }
    AppRouter.pathStack.pop(result);
  }

  /** 返回到指定页面 */
  static popToName(name: string): void {
    if (!AppRouter.pathStack) {
      hilog.error(DOMAIN_NUMBER, TAG, `popToName 失败：NavPathStack 未初始化（目标: ${name}）`);
      return;
    }
    AppRouter.pathStack.popToName(name);
  }

  /** 替换当前页面 */
  static replacePath(name: string, param?: Object): void {
    if (!AppRouter.pathStack) {
      hilog.error(DOMAIN_NUMBER, TAG, `replacePath 失败：NavPathStack 未初始化（目标: ${name}）`);
      return;
    }
    AppRouter.pathStack.replacePath({ name: name, param: param });
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

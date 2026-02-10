/**
 * 附近页面 - 主标签页
 * 包含动态、用户、群组三个子标签，顶部位置栏和搜索/地图入口
 * 类似抖音风格的子标签切换
 */

import { KEY_IS_LOGGED_IN, KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames } from '../../../constants/AppConstants';
import {
  MomentItem, UserItem, GroupItem, GeoPoint,
  GetFeedMomentsRequest, GetFeedUsersRequest, SearchGroupsRequest,
  MomentImage, FollowRequest, JoinGroupRequest
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { LoginPromptView } from '../../../components/LoginPromptView';
import { LocationBar } from '../../../components/LocationBar';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'NearbyPage';

class GroupDetailParams {
  groupId: string = '';
}

@Component
export struct NearbyPage {
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;

  @State currentSubTab: number = 0;
  private subTabController: TabsController = new TabsController();

  // 动态数据
  @State moments: MomentItem[] = [];
  @State momentsLoading: boolean = false;
  @State momentsError: string = '';

  // 用户数据
  @State users: UserItem[] = [];
  @State usersLoading: boolean = false;
  @State usersError: string = '';

  // 群组数据
  @State groups: GroupItem[] = [];
  @State groupsLoading: boolean = false;
  @State groupsError: string = '';

  async aboutToAppear() {
    await this.loadCurrentTabData();
  }

  private getReferencePoint(): GeoPoint | null {
    const loc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (!loc) return null;
    const point: GeoPoint = { latitude: loc.latitude, longitude: loc.longitude };
    return point;
  }

  private async loadCurrentTabData(): Promise<void> {
    switch (this.currentSubTab) {
      case 0:
        await this.loadNearbyMoments();
        break;
      case 1:
        await this.loadNearbyUsers();
        break;
      case 2:
        await this.loadNearbyGroups();
        break;
    }
  }

  // ==================== 加载附近动态 ====================
  private async loadNearbyMoments(): Promise<void> {
    if (!this.isLoggedIn) return;
    const ref = this.getReferencePoint();
    if (!ref) {
      this.momentsError = '请先获取位置信息';
      return;
    }
    this.momentsLoading = true;
    this.momentsError = '';
    try {
      const request: GetFeedMomentsRequest = {
        source: 'nearby',
        reference_point: ref,
        radius: 5000,
        pagination: { limit: 30 }
      };
      const response = await ApiClient.getFeedMoments(request);
      if (response.code === 0 && response.data) {
        this.moments = response.data.items;
      } else {
        this.momentsError = response.message || '加载失败';
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载附近动态失败: ${JSON.stringify(error)}`);
      this.momentsError = '加载附近动态失败';
    } finally {
      this.momentsLoading = false;
    }
  }

  // ==================== 加载附近用户 ====================
  private async loadNearbyUsers(): Promise<void> {
    if (!this.isLoggedIn) return;
    const ref = this.getReferencePoint();
    if (!ref) {
      this.usersError = '请先获取位置信息';
      return;
    }
    this.usersLoading = true;
    this.usersError = '';
    try {
      const request: GetFeedUsersRequest = {
        reference_point: ref,
        radius: 5000,
        pagination: { limit: 30 }
      };
      const response = await ApiClient.getFeedUsers(request);
      if (response.code === 0 && response.data) {
        this.users = response.data.items;
      } else {
        this.usersError = response.message || '加载失败';
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载附近用户失败: ${JSON.stringify(error)}`);
      this.usersError = '加载附近用户失败';
    } finally {
      this.usersLoading = false;
    }
  }

  // ==================== 加载附近群组 ====================
  private async loadNearbyGroups(): Promise<void> {
    const ref = this.getReferencePoint();
    if (!ref) {
      this.groupsError = '请先获取位置信息';
      return;
    }
    this.groupsLoading = true;
    this.groupsError = '';
    try {
      const request: SearchGroupsRequest = {
        reference_point: ref,
        radius: 5000,
        sort_by: 'distance',
        pagination: { limit: 30 }
      };
      const response = await ApiClient.searchGroups(request);
      if (response.code === 0 && response.data) {
        this.groups = response.data.items;
      } else {
        this.groupsError = response.message || '加载失败';
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载附近群组失败: ${JSON.stringify(error)}`);
      this.groupsError = '加载附近群组失败';
    } finally {
      this.groupsLoading = false;
    }
  }

  build() {
    Column() {
      // 顶部位置栏
      LocationBar({ showMapButton: true, showSearchButton: true })

      // 子标签栏 + 内容
      Tabs({ barPosition: BarPosition.Start, controller: this.subTabController }) {
        // 子标签 0: 动态
        TabContent() {
          if (this.isLoggedIn) {
            this.buildMomentsContent()
          } else {
            LoginPromptView({
              title: '查看附近动态需要登录',
              subtitle: '登录后可发现身边有趣的人和事',
              hint: '附近群组无需登录即可浏览',
              icon: AppIcon.CAMERA
            })
          }
        }
        .tabBar(this.SubTabBuilder('动态', 0))

        // 子标签 1: 用户
        TabContent() {
          if (this.isLoggedIn) {
            this.buildUsersContent()
          } else {
            LoginPromptView({
              title: '查看附近用户需要登录',
              subtitle: '登录后可发现附近的人并建立联系',
              hint: '附近群组无需登录即可浏览',
              icon: AppIcon.PERSON
            })
          }
        }
        .tabBar(this.SubTabBuilder('用户', 1))

        // 子标签 2: 群组
        TabContent() {
          this.buildGroupsContent()
        }
        .tabBar(this.SubTabBuilder('群组', 2))
      }
      .barMode(BarMode.Fixed)
      .barHeight(44)
      .onChange((index: number) => {
        this.currentSubTab = index;
        this.loadCurrentTabData();
      })
      .layoutWeight(1)
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  // ==================== 子标签构建器 ====================
  @Builder
  SubTabBuilder(title: string, index: number) {
    Column() {
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontWeight(this.currentSubTab === index ? FontWeight.Bold : FontWeight.Normal)
        .fontColor(this.currentSubTab === index
          ? $r('sys.color.ohos_id_color_text_primary')
          : $r('sys.color.ohos_id_color_text_secondary'))

      // 下划线指示器
      Divider()
        .width(24)
        .strokeWidth(3)
        .color(this.currentSubTab === index
          ? $r('sys.color.ohos_id_color_emphasize')
          : Color.Transparent)
        .margin({ top: 6 })
        .lineCap(LineCapStyle.Round)
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
  }

  // ==================== 动态内容 ====================
  @Builder
  buildMomentsContent() {
    if (this.momentsLoading && this.moments.length === 0) {
      SystemLoadingView({ message: '正在加载附近动态...' })
    } else if (this.momentsError.length > 0 && this.moments.length === 0) {
      this.buildEmptyState(AppIcon.WARNING, this.momentsError, '重试', () => { this.loadNearbyMoments(); })
    } else if (this.moments.length === 0) {
      this.buildEmptyState(AppIcon.CAMERA, '附近暂无动态\n去发布第一条吧！', undefined, undefined)
    } else {
      Refresh({ refreshing: $$this.momentsLoading }) {
        List({ space: DesignConstants.SPACING.SM }) {
          ForEach(this.moments, (moment: MomentItem) => {
            ListItem() {
              this.buildMomentCard(moment)
            }
          }, (moment: MomentItem) => moment.moment_id)
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
          top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
        .scrollBar(BarState.Auto)
      }
      .onRefreshing(() => { this.loadNearbyMoments(); })
    }
  }

  @Builder
  buildMomentCard(moment: MomentItem) {
    Column({ space: DesignConstants.SPACING.MD }) {
      // 用户信息
      Row({ space: DesignConstants.SPACING.SM }) {
        SystemAvatar({
          avatarUrl: moment.user.avatar_url,
          name: moment.user.nickname,
          avatarSize: 40
        })
        Column({ space: 2 }) {
          Text(moment.user.nickname)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
          if (moment.location?.location_name) {
            Text(moment.location.location_name)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)
      }
      .width('100%')

      // 文本内容
      Text(moment.description)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .lineHeight(22)
        .width('100%')

      // 图片网格
      if (moment.images && moment.images.length > 0) {
        this.buildImageGrid(moment.images)
      }

      // 互动栏
      Row() {
        this.buildStatItem(AppIcon.HEART, `${moment.likes_count}`)
        this.buildStatItem(AppIcon.MESSAGE, `${moment.comments_count}`)
      }
      .width('100%')
      .justifyContent(FlexAlign.Start)
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(DesignConstants.RADIUS.LG)
  }

  @Builder
  buildImageGrid(images: MomentImage[]) {
    if (images.length === 1) {
      Image(images[0].url)
        .width('100%')
        .constraintSize({ maxHeight: 240 })
        .objectFit(ImageFit.Cover)
        .borderRadius(DesignConstants.RADIUS.MD)
    } else {
      Flex({ wrap: FlexWrap.Wrap }) {
        ForEach(images.slice(0, 9), (img: MomentImage) => {
          Image(img.url)
            .width(images.length <= 4 ? '48%' : '31%')
            .aspectRatio(1)
            .objectFit(ImageFit.Cover)
            .borderRadius(DesignConstants.RADIUS.SM)
            .margin(2)
        }, (img: MomentImage, index: number) => `${img.url}_${index}`)
      }
      .width('100%')
    }
  }

  @Builder
  buildStatItem(icon: AppIcon, text: string) {
    Row({ space: 4 }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(16)
          .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
      }
      Text(text)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
    }
    .margin({ right: 24 })
  }

  // ==================== 用户内容 ====================
  @Builder
  buildUsersContent() {
    if (this.usersLoading && this.users.length === 0) {
      SystemLoadingView({ message: '正在加载附近用户...' })
    } else if (this.usersError.length > 0 && this.users.length === 0) {
      this.buildEmptyState(AppIcon.WARNING, this.usersError, '重试', () => { this.loadNearbyUsers(); })
    } else if (this.users.length === 0) {
      this.buildEmptyState(AppIcon.PERSON, '附近暂无其他用户', undefined, undefined)
    } else {
      Refresh({ refreshing: $$this.usersLoading }) {
        List({ space: DesignConstants.SPACING.SM }) {
          ForEach(this.users, (user: UserItem) => {
            ListItem() {
              this.buildUserCard(user)
            }
          }, (user: UserItem) => user.user.user_id)
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
          top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
        .scrollBar(BarState.Auto)
      }
      .onRefreshing(() => { this.loadNearbyUsers(); })
    }
  }

  @Builder
  buildUserCard(user: UserItem) {
    Row({ space: DesignConstants.SPACING.MD }) {
      SystemAvatar({
        avatarUrl: user.user.avatar_url,
        name: user.user.nickname,
        avatarSize: 52
      })

      Column({ space: 4 }) {
        Text(user.user.nickname)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))

        if (user.bio) {
          Text(user.bio)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(2)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }

        if (user.location?.location_name) {
          Row({ space: 4 }) {
            Text() {
              SymbolSpan($r(AppIcon.LOCATION))
                .fontSize(12)
                .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
            }
            Text(user.location.location_name)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))

            if (user.location.distance !== undefined && user.location.distance !== null) {
              Text(this.formatDistance(user.location.distance))
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor($r('sys.color.ohos_id_color_emphasize'))
            }
          }
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      // 关注按钮
      Button('关注')
        .type(ButtonType.Capsule)
        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
        .fontColor(Color.White)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .height(32)
        .onClick(() => { this.followUser(user); })
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(DesignConstants.RADIUS.LG)
  }

  // ==================== 群组内容 ====================
  @Builder
  buildGroupsContent() {
    if (this.groupsLoading && this.groups.length === 0) {
      SystemLoadingView({ message: '正在加载附近群组...' })
    } else if (this.groupsError.length > 0 && this.groups.length === 0) {
      this.buildEmptyState(AppIcon.WARNING, this.groupsError, '重试', () => { this.loadNearbyGroups(); })
    } else if (this.groups.length === 0) {
      this.buildEmptyState(AppIcon.GROUP, '附近暂无群组', undefined, undefined)
    } else {
      Refresh({ refreshing: $$this.groupsLoading }) {
        List({ space: DesignConstants.SPACING.SM }) {
          ForEach(this.groups, (group: GroupItem) => {
            ListItem() {
              this.buildGroupCard(group)
            }
          }, (group: GroupItem) => group.group.group_id)
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
          top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
        .scrollBar(BarState.Auto)
      }
      .onRefreshing(() => { this.loadNearbyGroups(); })
    }
  }

  @Builder
  buildGroupCard(group: GroupItem) {
    Row({ space: DesignConstants.SPACING.MD }) {
      SystemAvatar({
        avatarUrl: group.group.avatar_url || '',
        name: group.group.name,
        avatarSize: 52,
        shape: 'squircle'
      })

      Column({ space: 4 }) {
        Text(group.group.name)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: DesignConstants.SPACING.SM }) {
          Text(`${group.group.members_count}人`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))

          if (group.location?.location_name) {
            Text(group.location.location_name)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
              .constraintSize({ maxWidth: '50%' })
          }
        }

        if (group.description) {
          Text(group.description)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      Button('加入')
        .type(ButtonType.Capsule)
        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
        .fontColor(Color.White)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .height(32)
        .onClick(() => { this.joinGroup(group); })
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(DesignConstants.RADIUS.LG)
    .onClick(() => { this.openGroupDetail(group); })
  }

  // ==================== 通用空状态 ====================
  @Builder
  buildEmptyState(icon: AppIcon, message: string, actionLabel: string | undefined, action: (() => void) | undefined) {
    Column({ space: DesignConstants.SPACING.LG }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(48)
          .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
      }
      Text(message)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)
        .lineHeight(24)
      if (actionLabel && action) {
        Button(actionLabel)
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .height(40)
          .onClick(() => { action(); })
      }
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
    .padding(DesignConstants.SPACING.XXL)
  }

  // ==================== 操作方法 ====================
  private async followUser(user: UserItem): Promise<void> {
    try {
      const followReq: FollowRequest = { user_id: user.user.user_id };
      const response = await ApiClient.followUser(followReq);
      if (response.code === 0) {
        SystemToast.showSuccess('已关注', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '关注失败', this.getUIContext());
      }
    } catch (_e) {
      SystemToast.showError('关注失败', this.getUIContext());
    }
  }

  private async joinGroup(group: GroupItem): Promise<void> {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再加入群组', this.getUIContext());
      this.navigateTo(`features/auth/pages/${PageNames.LOGIN}`);
      return;
    }
    try {
      const joinReq: JoinGroupRequest = { group_id: group.group.group_id };
      const response = await ApiClient.joinGroup(joinReq);
      if (response.code === 0 && response.data) {
        if (response.data.status === 'joined') {
          SystemToast.showSuccess('加入成功', this.getUIContext());
          await this.loadNearbyGroups();
        } else if (response.data.status === 'pending_approval') {
          SystemToast.show('已提交申请，等待审批', this.getUIContext());
        }
      } else if (response.code === 1305) {
        SystemToast.show('您已是该群成员', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '加入失败', this.getUIContext());
      }
    } catch (_e) {
      SystemToast.showError('加入群组失败', this.getUIContext());
    }
  }

  private openGroupDetail(group: GroupItem): void {
    const params = new GroupDetailParams();
    params.groupId = group.group.group_id;
    this.navigateTo(`features/group/pages/${PageNames.GROUP_DETAIL}`, params);
  }

  private navigateTo(url: string, params?: object): void {
    try {
      const routerInstance = this.getUIContext().getRouter();
      if (params) {
        routerInstance.pushUrl({ url: url, params: params });
      } else {
        routerInstance.pushUrl({ url: url });
      }
    } catch (_e) {
      // ignore
    }
  }

  private formatDistance(meters: number): string {
    if (meters < 1000) {
      return `${Math.round(meters)}m`;
    }
    return `${(meters / 1000).toFixed(1)}km`;
  }
}

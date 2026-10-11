/**
 * 通知订阅页面
 * TODO: Notification subscriptions and device token APIs are not in the backend spec yet.
 * This page uses local AppStorage/PreferencesUtil for client-side preferences only.
 * When backend spec defines subscription APIs, migrate to API calls.
 */
import { StorageKeys, PageNames } from '../../../constants/AppConstants';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemToast } from '../../../components/SystemToast';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemButton } from '../../../components/SystemButton';
import { PreferencesUtil } from '../../../core/storage/PreferencesUtil';
import { TokenAssetStore } from '../../../core/storage/TokenAssetStore';
import { AppRouter } from '../../../core/router/AppRouter';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { LucideIcon } from '../../../components/LucideIcon';

const SUBSCRIPTIONS_STORAGE_KEY = 'notification_subscriptions';

// Local subscription preferences (client-side only; no backend API)
interface LocalSubscriptionPrefs {
  new_follower_enabled?: boolean;
  mutual_follow_enabled?: boolean;
  group_invite_enabled?: boolean;
  group_invite_response_enabled?: boolean;
  group_application_enabled?: boolean;
  group_application_response_enabled?: boolean;
  message_mention_enabled?: boolean;
  moment_like_enabled?: boolean;
  moment_comment_enabled?: boolean;
  comment_reply_enabled?: boolean;
  system_announcement_enabled?: boolean;
}

interface SubscriptionItem {
  key: keyof LocalSubscriptionPrefs;
  icon: AppIcon;
  title: string;
  description: string;
}

@Preview
@Component
export struct SubscriptionsPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @State private loading: boolean = true;
  @State private savingKey: string = '';
  @State private data: LocalSubscriptionPrefs | null = null;
  // 页面级加载失败态：失败时明确告知当前展示的是兜底默认值（否则用户无从分辨）
  @State private loadError: boolean = false;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private isDestroyed: boolean = false;

  private items: SubscriptionItem[] = [
    { key: 'new_follower_enabled', icon: AppIcon.PERSON, title: '新关注者', description: '有人关注你时通知' },
    { key: 'mutual_follow_enabled', icon: AppIcon.HEART, title: '互相关注', description: '成为好友时通知' },
    { key: 'group_invite_enabled', icon: AppIcon.GROUP, title: '群组邀请', description: '收到群组邀请时通知' },
    { key: 'group_invite_response_enabled', icon: AppIcon.GROUP, title: '邀请回应', description: '对方接受/拒绝邀请时通知' },
    { key: 'group_application_enabled', icon: AppIcon.GROUP, title: '入群申请', description: '收到入群申请时通知' },
    { key: 'group_application_response_enabled', icon: AppIcon.GROUP, title: '申请回应', description: '你的入群申请有结果时通知' },
    { key: 'message_mention_enabled', icon: AppIcon.MESSAGE, title: '消息提及', description: '群聊中被 @ 时通知' },
    { key: 'moment_like_enabled', icon: AppIcon.HEART_FILL, title: '动态点赞', description: '动态被点赞时通知' },
    { key: 'moment_comment_enabled', icon: AppIcon.MESSAGE, title: '动态评论', description: '动态被评论时通知' },
    { key: 'comment_reply_enabled', icon: AppIcon.MESSAGE, title: '评论回复', description: '评论被回复时通知' },
    { key: 'system_announcement_enabled', icon: AppIcon.NOTIFICATION, title: '系统公告', description: '接收系统公告通知' }
  ];

  aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    this.ensureAuthAndLoad();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceSubtle(): string {
    return this.getCurrentThemePalette().surfaceSubtle;
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private async ensureAuthAndLoad(): Promise<void> {
    if (this.isDestroyed) return;
    try {
      const token = (await TokenAssetStore.get(TokenAssetStore.ALIAS_CURRENT_ACCESS)) ?? '';
      if (this.isDestroyed) return;
      if (!token) {
        // 未登录时直接返回上一页（登录检测应在跳转源头完成）
        AppRouter.pop();
        return;
      }
      await this.load();
    } catch (e) {
      this.loading = false;
      this.loadError = true;
      SystemToast.showError('加载失败', this.getUIContext());
    }
  }

  private async load(): Promise<void> {
    if (this.isDestroyed) return;
    this.loading = true;
    this.loadError = false;
    try {
      const stored = await PreferencesUtil.getObject<LocalSubscriptionPrefs>(SUBSCRIPTIONS_STORAGE_KEY);
      if (this.isDestroyed) return;
      this.data = stored ?? this.getDefaultPrefs();
    } catch (e) {
      // 本地读取失败：展示默认偏好，但置错误态提示用户当前并非其真实设置
      this.data = this.getDefaultPrefs();
      this.loadError = true;
    } finally {
      this.loading = false;
    }
  }

  private getDefaultPrefs(): LocalSubscriptionPrefs {
    return {
      new_follower_enabled: true,
      mutual_follow_enabled: true,
      group_invite_enabled: true,
      group_invite_response_enabled: true,
      group_application_enabled: true,
      group_application_response_enabled: true,
      message_mention_enabled: true,
      moment_like_enabled: true,
      moment_comment_enabled: true,
      comment_reply_enabled: true,
      system_announcement_enabled: true
    };
  }

  private getValue(key: keyof LocalSubscriptionPrefs): boolean {
    if (!this.data) return false;
    const dataRecord = (this.data as Object) as Record<string, boolean>;
    return dataRecord[key as string] ?? false;
  }

  private setValue(key: keyof LocalSubscriptionPrefs, value: boolean): void {
    if (!this.data) return;
    const dataRecord = (this.data as Object) as Record<string, boolean>;
    dataRecord[key as string] = value;
    this.data = (dataRecord as Object) as LocalSubscriptionPrefs;
  }

  private async update(key: keyof LocalSubscriptionPrefs, value: boolean): Promise<void> {
    if (this.isDestroyed) return;
    if (!this.data) return;
    if (this.savingKey) return;

    const prev = this.getValue(key);
    this.savingKey = String(key);
    this.setValue(key, value);

    try {
      await PreferencesUtil.setObject(SUBSCRIPTIONS_STORAGE_KEY, this.data);
      if (this.isDestroyed) return;
      SystemToast.showSuccess('已保存', this.getUIContext());
    } catch (e) {
      this.setValue(key, prev);
      SystemToast.showError('保存失败', this.getUIContext());
    } finally {
      this.savingKey = '';
    }
  }

  build() {
    NavDestination() {
  PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint}) {
    Column() {
      // 标题栏 —— 统一走 SystemPageHeader（居中标题 + 返回）
      SystemPageHeader({
        title: '通知偏好',
        titleSize: DesignConstants.FONT.SIZE.XXL,
        titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
        showBackButton: true,
        onBack: () => { AppRouter.pop(); },
        titleAlign: TextAlign.Center,
        backgroundVariant: 'transparent'
      })

      if (this.loading) {
        SystemLoadingView({ message: '加载订阅设置…' })
          .layoutWeight(1)
          .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
      } else {
        List() {
          if (this.loadError) {
            ListItem() {
              Column({ space: DesignConstants.SPACING.SM }) {
                Text('加载失败，当前展示默认偏好')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                SystemButton({
                  label: '重试',
                  variant: 'outline',
                  buttonSize: 'small',
                  onPress: () => {
                    if (!this.isDestroyed) void this.ensureAuthAndLoad();
                  }
                })
              }
              .width('100%')
              .alignItems(HorizontalAlign.Center)
              .padding(DesignConstants.SPACING.MD)
            }
          }

          ListItem() {
            this.ListHeader('通知类型')
          }

          ForEach(this.items, (item: SubscriptionItem, index: number) => {
            if (index > 0) {
              ListItem() {
                this.RowDivider()
              }
            }
            ListItem() {
              this.SubscriptionItemRow(item)
            }
          }, (i: SubscriptionItem) =>
            (i.key as string) + '_v' + this.getValue(i.key) + '_s' + (this.savingKey === String(i.key)))

          ListItem() {
            Text('本页开关仅保存在当前设备，不影响实际收到的通知（通知仍以服务端推送为准）。待服务端通知偏好协议落地后，将自动同步到所有设备。')
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM })
          }
        }
        .divider({ strokeWidth: 0 })
        .layoutWeight(1)
        .scrollBar(BarState.Off)
        .width('100%')
        // 分组外框（唯一一层白卡 + 发丝描边 + surfaceShadow）；行间分隔线由
        // SubscriptionItemRow 内的 hairline Divider 承担，行本身不再自带卡片样式
        .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
        .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getCurrentThemePalette().borderColor })
        .shadow(this.getCurrentThemePalette().surfaceShadow)
        .padding({ bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, DesignConstants.SPACING.LG) })
      }
    }
    .width('100%')
    .height('100%')
    }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }

  @Builder
  private ListHeader(title: string) {
    Text(title)
      .fontSize(DesignConstants.FONT.SIZE.MD)
      .fontColor(this.getCurrentThemePalette().textSecondary)
      .margin({ left: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.MD })
      .width('100%')
  }

  /** 行间 hairline 分隔线（分组内行边界；对齐前端 ListContainer divide-y） */
  @Builder
  private RowDivider() {
    Divider()
      .strokeWidth(DesignConstants.BORDER_WIDTH.HAIRLINE)
      .color(this.getCurrentThemePalette().borderColor)
      .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
  }

  @Builder
  private SubscriptionItemRow(item: SubscriptionItem) {
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        LucideIcon({
            icon: item.icon,
            iconSize: DesignConstants.FONT.SIZE.LG_PLUS,
            color: this.getCurrentThemePalette().accentBackground
          })
      }
      .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
      .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
      .borderRadius(DesignConstants.RADIUS.MD)
      .backgroundColor(this.getSurfaceSubtle())
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      Column({ space: DesignConstants.SPACING.XS }) {
        Text(item.title)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
        Text(item.description)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)

      Toggle({ type: ToggleType.Switch, isOn: this.getValue(item.key) })
        .selectedColor(this.getCurrentThemePalette().accentBackground)
        .onChange((isOn: boolean) => {
          this.update(item.key, isOn);
        })
        .enabled(!this.savingKey || this.savingKey === String(item.key))
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    // 行级卡片样式移除：外框由订阅列表容器承担，行间分隔线走 hairline
  }
}

@Builder
export function SubscriptionsPageBuilder() {
  SubscriptionsPage()
}
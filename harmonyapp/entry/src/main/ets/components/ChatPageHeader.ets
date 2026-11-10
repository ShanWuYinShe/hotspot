/**
 * ChatPageHeader
 * 聊天页浮动页头（对齐前端 ChatView / GroupChatView 移动端头部）
 * - 浮动返回胶囊：返回钮 36 + hairline 描边 + surface 85% 底 + 轻投影（web：
 *   rounded-full border border-default/60 bg-surface-primary/85 shadow-raised-sm）
 * - 头像 40 + 标题 20/600 + 状态 11（私聊 = 在线/离线，群聊 = 正在输入…/N 位成员）；
 *   头像+标题整体可点：私聊进对方主页，群聊进群详情（web avatar-entry button）
 * - 群聊头像右下角类型角标（对齐前端 GroupChatBadge），私聊右下角在线点
 * - 无底部分隔线：头部浮在聊天画布上（web 头部 h-16 透明、仅移动端有返回胶囊）
 * - onTouch 下沉键盘（点头部空白处收起输入法）
 *
 * 与 SystemPageHeader 的差异：不占满整条底栏、无分隔线，且需要浮动返回 + 头像 + 键盘
 * 下沉，故独立组件，仍复用同套主题/DesignConstants 原语。
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../core/theme/AppThemeVisuals';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { AppIcon } from './SystemIcon';
import { SystemAvatar } from './SystemAvatar';
import { KeyboardDismissManager } from '../core/utils/KeyboardDismissManager';
import { SafeAreaUtils } from '../core/utils/SafeAreaUtils';
import { SurfaceIconButton } from './SurfaceIconButton';
import { LucideIcon } from './LucideIcon';

@Component
export struct ChatPageHeader {
  /** 顶部安全区高度（全屏 edge-to-edge 下标题避让状态栏，SystemPageHeader 内聚同款逻辑） */
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  /** 标题（会话名）。 */
  @Prop title: string = '';
  /** 私聊时对方头像 URL；提供则显示 leading 头像。 */
  @Prop targetAvatar: string = '';
  /** 是否展示 leading 头像（私聊 true / 群聊 false）。 */
  @Prop showAvatar: boolean = false;
  /** 在线状态副文本（在线/离线；空串不显示）。 */
  @Prop presenceText: string = '';
  /** 对端是否在线（决定副文本颜色与头像在线点）。 */
  @Prop presenceOnline: boolean = false;
  /**
   * 群聊模式（对齐前端 GroupChatView 头部）：
   * - 头像用群头像（圆角方 shape=squircle，对应 web shape="rounded"）+ 类型角标
   * - 副文本 = 「正在输入…」提示（accent 色）/「N 位成员」/「群聊」
   * - 头像 + 标题整体可点进群详情（web avatar-entry button）
   */
  @Prop isGroup: boolean = false;
  @Prop groupAvatar: string = '';
  @Prop membersCount: number = 0;
  /** 群聊「正在输入」提示文本；非空时副文本切 accent 色。 */
  @Prop typingHint: string = '';
  /** 私聊：头像 + 标题可点进对方主页（web ROUTES.user）。 */
  onOpenProfile: () => void = () => {};
  /** 群聊：头像 + 标题可点进群详情。 */
  onOpenGroupDetail: () => void = () => {};
  /** 是否显示尾随操作钮（web GroupChatView 无尾随按钮，群聊入口收敛到头部）。 */
  @Prop showTrailing: boolean = true;
  /** 返回回调。 */
  onBack: () => void = () => {};
  /** 尾随操作回调（打开会话设置/操作）。 */
  onTrailingAction: () => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  /** 头像 + 标题整体是否可点（群聊恒可点；私聊需提供回调） */
  private isEntryClickable(): boolean {
    return this.isGroup ? true : this.showAvatar;
  }

  private getEntryAccessibilityText(): string {
    if (this.isGroup) {
      return `查看群「${this.title}」详情`;
    }
    return this.showAvatar ? `查看${this.title}的资料` : '';
  }

  /** 群聊副文本：正在输入（accent）/ N 位成员 / 群聊 */
  private getGroupSubText(): string {
    if (this.typingHint.length > 0) {
      return this.typingHint;
    }
    return this.membersCount > 0 ? `${this.membersCount} 位成员` : '群聊';
  }

  build() {
    Row({ space: DesignConstants.SPACING.SM }) {
      // 浮动返回胶囊（web：p-0.5 圆形容器 + 85% 表面 + 发丝描边 + 轻投影）
      Row() {
        SurfaceIconButton({
          icon: AppIcon.BACK,
          // ghost 底：容器本身已承载「浮动卡」语义，钮内不再叠第二层表面
          kind: 'neutral',
          buttonSize: DesignConstants.LAYOUT.ICON_BUTTON_SIZE,
          accessibilityLabel: '返回',
          onPress: () => {
            this.onBack();
          }
        })
      }
      .padding(DesignConstants.SPACING.HALF)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(AppThemeVisuals.withAlpha(this.getPalette().surfaceBackground, 0.85))
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
      .shadow(this.getPalette().controlShadow)

      // 头像 + 标题/状态（唯一资料入口，可点进主页/群详情）
      Row({ space: DesignConstants.SPACING.MD2 }) {
        Stack({ alignContent: Alignment.BottomEnd }) {
          if (this.isGroup || this.showAvatar) {
            SystemAvatar({
              avatarUrl: this.isGroup ? this.groupAvatar : this.targetAvatar,
              name: this.title,
              avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
              // 群聊头像圆角方（web shape="rounded"）；私聊保持圆形
              shape: this.isGroup ? 'squircle' : 'circle',
              // 私聊头像右下角在线点（web OnlineDot sm）；群聊该位置让给类型角标
              isOnline: !this.isGroup && this.presenceOnline,
              onlineDotSize: DesignConstants.COMPONENT.ONLINE_DOT_MD
            })
          }
          if (this.isGroup) {
            // 群聊类型角标（对齐前端 GroupChatBadge：accent 圆 + 成员图标 + 表面色描边）
            Row() {
              LucideIcon({
                icon: AppIcon.GROUP,
                iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                color: this.getPalette().accentText
              })
            }
            .width(DesignConstants.SPACING.LG2)
            .height(DesignConstants.SPACING.LG2)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor(this.getPalette().accentBackground)
            .border({ width: DesignConstants.BORDER_WIDTH.STRONG, color: this.getPalette().surfaceBackground })
            .justifyContent(FlexAlign.Center)
          }
        }

        // 标题 20/600 + 状态 11（web：text-xl font-semibold + mt-0.5 text-xs）
        Column({ space: DesignConstants.SPACING.HALF }) {
          Text(this.title)
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getPalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          if (this.isGroup) {
            Text(this.getGroupSubText())
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.typingHint.length > 0 ?
                this.getPalette().accentBackground : this.getPalette().textSecondary)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          } else if (this.presenceText.length > 0) {
            Text(this.presenceText)
              // 前端 presence 为 text-xs(11)；原 XS2(10) 为近邻档漂移
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.presenceOnline ? this.getPalette().online : this.getPalette().textSecondary)
              .maxLines(1)
          }
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)
      }
      .layoutWeight(1)
      .onClick(() => {
        if (!this.isEntryClickable()) {
          return;
        }
        if (this.isGroup) {
          this.onOpenGroupDetail();
        } else {
          this.onOpenProfile();
        }
      })
      .accessibilityGroup(true)
      .accessibilityText(this.getEntryAccessibilityText())

      // 尾随：会话操作 ghost 圆钮（私聊保留；群聊 web 无尾随按钮——入口已收敛到头部）
      if (this.showTrailing) {
        SurfaceIconButton({
          icon: AppIcon.MORE,
          kind: 'neutral',
          buttonSize: DesignConstants.LAYOUT.ICON_BUTTON_SIZE,
          accessibilityLabel: '会话操作',
          onPress: () => {
            this.onTrailingAction();
          }
        })
      }
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop, DesignConstants.SPACING.SM, DesignConstants.SPACING.MD),
      bottom: DesignConstants.SPACING.SM
    })
    .zIndex(1)
    .onTouch((event: TouchEvent) => {
      // 点头部空白处收起输入法（与原 ChatPage 行为一致）
      if (event.type === TouchType.Down) {
        KeyboardDismissManager.getInstance().dismiss();
      }
    })
  }
}

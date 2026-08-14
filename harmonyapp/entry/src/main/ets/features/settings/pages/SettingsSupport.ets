import { AppIcon } from '../../../components/SystemIcon';
import { DesignConstants } from '../../../constants/DesignConstants';
import { curves } from '@kit.ArkUI';
import {
  ApiVersionMeta,
  BlockedUserItem,
  ClientConfig,
  SessionItem,
  SystemAnnouncementItem,
  UserSettings,
  ContactInfo,
  RelatedUserItem
} from '../../../core/types/api';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { SystemButton } from '../../../components/SystemButton';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { AppRouter, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';

export interface SettingsEntryItem {
  icon: AppIcon;
  title: string;
  subtitle: string;
  action: () => void;
  accentColor?: string;
}

export interface SettingToggleItem {
  key: keyof UserSettings;
  title: string;
  subtitle: string;
}

export interface SupportLinkItem {
  title: string;
  value: string;
}

function formatDateTime(value?: string): string {
  if (!value) {
    return '--';
  }
  try {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return value;
    }
    const month = (date.getMonth() + 1).toString().padStart(2, '0');
    const day = date.getDate().toString().padStart(2, '0');
    const hour = date.getHours().toString().padStart(2, '0');
    const minute = date.getMinutes().toString().padStart(2, '0');
    return `${month}-${day} ${hour}:${minute}`;
  } catch (_) {
    return value;
  }
}

function formatSessionLabel(item: SessionItem): string {
  const deviceName = item.device_name ?? item.device_type ?? '未知设备';
  const location = item.ip_location ?? item.ip_address ?? '未知地点';
  if (item.is_current) {
    return `${deviceName} · 当前设备`;
  }
  return `${deviceName} · ${location}`;
}

function buildSupportLinks(config: ClientConfig | null): SupportLinkItem[] {
  const contact: ContactInfo | undefined = config?.contact;
  const items: SupportLinkItem[] = [];
  if (!contact) {
    return items;
  }
  if (contact.support_email) {
    items.push({ title: '支持邮箱', value: contact.support_email });
  }
  if (contact.privacy_policy_url) {
    items.push({ title: '隐私政策', value: contact.privacy_policy_url });
  }
  if (contact.terms_of_service_url) {
    items.push({ title: '服务条款', value: contact.terms_of_service_url });
  }
  if (contact.official_website_url) {
    items.push({ title: '官方网站', value: contact.official_website_url });
  }
  return items;
}

@Component
export struct SettingsSectionCard {
  @Prop title: string = '';
  @BuilderParam content: () => void;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getCardStyle() {
    return AppThemeVisuals.getSurfaceCardStyle(this.resolvedThemeId);
  }

  build() {
    Column({ space: DesignConstants.SPACING.SM }) {
      Text(this.title)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getPalette().textSecondary)
        .width('100%')

      Column({ space: 0 }) {
        this.content();
      }
      .width('100%')
      .backgroundColor(this.getCardStyle().backgroundColor)
      .borderRadius(this.getCardStyle().borderRadius)
      .shadow(this.getCardStyle().shadow)
      .border({ width: this.getCardStyle().borderWidth, color: this.getCardStyle().borderColor })
      .clip(true)
    }
    .width('100%')
  }
}

@Component
export struct SettingsEntryRow {
  @Prop item: SettingsEntryItem;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State private isPressed: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getListItemStyle() {
    return AppThemeVisuals.getListItemCardStyle(this.resolvedThemeId);
  }

  build() {
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        Text() {
          SymbolSpan($r(this.item.icon))
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontColor([this.getPalette().textSecondary])
        }
      }
      .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getPalette().surfaceSubtle)
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      Column({ space: DesignConstants.SPACING.XS }) {
        Text(this.item.title)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
        if (this.item.subtitle) {
          Text(this.item.subtitle)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getPalette().textSecondary)
            .maxLines(2)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)

      SurfaceIconButton({
        icon: AppIcon.ARROW_RIGHT,
        kind: 'neutral',
        buttonSize: 28,
        glyphSize: 14,
        onPress: this.item.action
      })
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD3, bottom: DesignConstants.SPACING.MD3 })
    .backgroundColor(this.getListItemStyle().backgroundColor)
    .borderRadius(this.getListItemStyle().borderRadius)
    // 整行按压反馈（对齐 tap-97 语义）
    .scale(this.isPressed ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.isPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.isPressed = false;
      }
    })
    .onClick(this.item.action)
  }
}

@Component
export struct SettingsMetaCard {
  @Prop meta: ApiVersionMeta | null = null;
  @Prop loading: boolean = false;
  @Prop error: string = '';
  @Prop lastChecked: string = '';
  refreshAction: () => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getStatusText(): string {
    if (!this.meta) {
      return '未获取';
    }
    switch (this.meta.status) {
      case 'deprecated':
        return '建议升级';
      case 'sunset':
        return '维护中';
      default:
        return '正常';
    }
  }

  private getStatusBackground(): ResourceColor {
    if (!this.meta) {
      return AppThemeManager.getSoftStatusBackground('neutral');
    }
    switch (this.meta.status) {
      case 'deprecated':
        return AppThemeManager.getSoftStatusBackground('warning');
      case 'sunset':
        return AppThemeManager.getSoftStatusBackground('danger');
      default:
        return AppThemeManager.getSoftStatusBackground('success');
    }
  }

  build() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        Column({ space: DesignConstants.SPACING.XS }) {
          Text('系统版本状态')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().textPrimary)
          Text(this.meta?.status === 'sunset' ? '可能影响正常使用' : '与服务端版本保持同步')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().textSecondary)
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)

        Text(this.getStatusText())
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
          .backgroundColor(this.getStatusBackground())
          .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
          .borderRadius(DesignConstants.SHAPE.CHIP)

        SurfaceIconButton({
          icon: AppIcon.REFRESH,
          kind: 'neutral',
          buttonSize: 36,
          glyphSize: 18,
          disabled: this.loading,
          onPress: this.refreshAction
        })
      }

      if (this.loading) {
        Row({ space: DesignConstants.SPACING.SM }) {
          LoadingProgress().width(DesignConstants.SPACING.LG).height(DesignConstants.SPACING.LG).color(this.getPalette().accentBackground)
          Text('正在获取版本信息...')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().textSecondary)
        }
      } else {
        SettingsInfoLine({ label: '当前版本', value: this.meta?.version ?? '--' })
        SettingsInfoLine({ label: '推荐版本', value: this.meta?.recommended_version ?? '已是最新版本' })
        SettingsInfoLine({ label: '提示', value: this.meta?.message ?? '服务运行正常' })
        SettingsInfoLine({ label: '检查时间', value: formatDateTime(this.lastChecked) })
      }

      if (this.error) {
        Text(this.error)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().accentBackground)
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
  }
}

@Component
export struct SettingsInfoLine {
  @Prop label: string = '';
  @Prop value: string = '';
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row({ space: DesignConstants.SPACING.MD }) {
      Text(this.label)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getPalette().textSecondary)
        .layoutWeight(1)
      Text(this.value)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getPalette().textPrimary)
        .textAlign(TextAlign.End)
    }
    .width('100%')
  }
}

@Component
export struct AnnouncementSummaryCard {
  @Prop items: SystemAnnouncementItem[] = [];
  @Prop maxItems: number = 2;
  @Prop showOpenAll: boolean = true;
  openAll: () => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSeverityColor(severity: string): ResourceColor {
    switch (severity) {
      case 'warning':
        return this.getPalette().warning;
      case 'error':
        return this.getPalette().error;
      default:
        return this.getPalette().accentBackground;
    }
  }

  build() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row() {
        Column({ space: DesignConstants.SPACING.XS }) {
          Text('系统公告')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().textPrimary)
          Text(this.items.length > 0 ? '查看最近平台通知与维护提醒' : '当前暂无系统公告')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().textSecondary)
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)

        if (this.items.length > 0 && this.showOpenAll) {
          SystemButton({
            label: '全部查看',
            variant: 'surface-accent',
            buttonSize: 'large',
            onPress: () => { this.openAll(); }
          })
        }
      }
      .width('100%')

      if (this.items.length > 0) {
        ForEach(this.maxItems > 0 ? this.items.slice(0, this.maxItems) : this.items, (item: SystemAnnouncementItem) => {
          Column({ space: DesignConstants.SPACING.SM2 }) {
            Row({ space: DesignConstants.SPACING.SM }) {
              Text(item.title)
                .fontSize(DesignConstants.FONT.SIZE.MD)
                .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                .fontColor(this.getPalette().textPrimary)
                .layoutWeight(1)
                .maxLines(1)
                .textOverflow({ overflow: TextOverflow.Ellipsis })
              Text(item.severity === 'error' ? '重要' : item.severity === 'warning' ? '提醒' : '通知')
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(this.getSeverityColor(item.severity))
                .backgroundColor(this.getPalette().surfaceSubtle)
                .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
                .borderRadius(DesignConstants.RADIUS.FULL)
            }
            .width('100%')

            Text(item.content)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .lineHeight(DesignConstants.FONT.LINE_HEIGHT.SM)
              .fontColor(this.getPalette().textSecondary)
              .maxLines(2)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
              .width('100%')

            Text(formatDateTime(item.created_at))
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getPalette().textTertiary)
              .width('100%')
          }
          .width('100%')
          .padding(DesignConstants.SPACING.MD)
          .backgroundColor(this.getPalette().surfaceElevated)
          .borderRadius(DesignConstants.RADIUS.LG)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().floatingBorderColor })
        }, (item: SystemAnnouncementItem) => item.announcement_id)
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
  }
}

@Component
export struct SupportConfigCard {
  @Prop config: ClientConfig | null = null;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Text('支持与服务信息')
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getPalette().textPrimary)
        .width('100%')

      if (this.config) {
        SettingsInfoLine({ label: '公开注册', value: this.config.features.enable_public_registration ? '已开启' : '未开启' })
        SettingsInfoLine({ label: '访客体验', value: this.config.features.enable_guest_register ? '已开启' : '未开启' })
        SettingsInfoLine({ label: '瞬间功能', value: this.config.features.enable_moments ? '已开启' : '未开启' })
        SettingsInfoLine({ label: '附近的人', value: this.config.features.enable_nearby_people ? '已开启' : '未开启' })
        SettingsInfoLine({ label: '默认位置', value: this.config.defaults.default_location.location_name })
      } else {
        Text('客户端公开配置尚未加载。')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().textSecondary)
          .width('100%')
      }

      if (buildSupportLinks(this.config).length > 0) {
        Column({ space: DesignConstants.SPACING.SM }) {
          ForEach(buildSupportLinks(this.config), (item: SupportLinkItem) => {
            SettingsInfoLine({ label: item.title, value: item.value })
          }, (item: SupportLinkItem) => `${item.title}-${item.value}`)
        }
        .width('100%')
        .padding(DesignConstants.SPACING.MD)
        .backgroundColor(this.getPalette().surfaceElevated)
        .borderRadius(DesignConstants.RADIUS.LG)
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
  }
}

@Component
export struct ToggleSettingRow {
  @Prop item: SettingToggleItem;
  @Prop isOn: boolean = false;
  onChange: (value: boolean) => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row({ space: DesignConstants.SPACING.MD }) {
      Column({ space: DesignConstants.SPACING.XS }) {
        Text(this.item.title)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
        Text(this.item.subtitle)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().textSecondary)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)

      Toggle({ type: ToggleType.Switch, isOn: this.isOn })
        .selectedColor(this.getPalette().accentBackground)
        .onChange(this.onChange)
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
  }
}

@Component
export struct SessionListSection {
  @Prop sessions: SessionItem[] = [];
  revokeOne: (item: SessionItem) => void = () => {};
  revokeAll: () => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row() {
        Column({ space: DesignConstants.SPACING.XS }) {
          Text('当前会话')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().textPrimary)
          Text('查看已登录设备，并可撤销其他会话。')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().textSecondary)
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)

        if (this.sessions.length > 1) {
          SystemButton({
            label: '全部下线',
            variant: 'surface-accent',
            buttonSize: 'large',
            onPress: () => { this.revokeAll(); }
          })
        }
      }
      .width('100%')

      if (this.sessions.length === 0) {
        Text('暂无会话信息')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().textSecondary)
          .width('100%')
      } else {
        ForEach(this.sessions, (item: SessionItem) => {
          Row({ space: DesignConstants.SPACING.MD }) {
            Column({ space: DesignConstants.SPACING.XS }) {
              Text(formatSessionLabel(item))
                .fontSize(DesignConstants.FONT.SIZE.MD)
                .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                .fontColor(this.getPalette().textPrimary)
              Text(`创建于 ${formatDateTime(item.created_at)}`)
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor(this.getPalette().textSecondary)
              Text(`最近活跃 ${formatDateTime(item.last_active_at)}`)
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor(this.getPalette().textSecondary)
            }
            .layoutWeight(1)
            .alignItems(HorizontalAlign.Start)

            if (!item.is_current) {
              SystemButton({
                label: '撤销',
                variant: 'surface-accent',
                buttonSize: 'large',
                onPress: () => { this.revokeOne(item); }
              })
            } else {
              Text('当前')
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(this.getPalette().accentBackground)
                .backgroundColor(this.getPalette().surfaceSubtle)
                .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
                .borderRadius(DesignConstants.RADIUS.FULL)
            }
          }
          .width('100%')
          .padding(DesignConstants.SPACING.MD)
          .backgroundColor(this.getPalette().surfaceElevated)
          .borderRadius(DesignConstants.RADIUS.LG)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().floatingBorderColor })
        }, (item: SessionItem) => item.session_id)
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
  }
}

@Component
export struct RelationshipListSection {
  @Prop title: string = '';
  @Prop subtitle: string = '';
  @Prop emptyText: string = '';
  @Prop friends: RelatedUserItem[] = [];
  @Prop following: RelatedUserItem[] = [];
  @Prop followers: RelatedUserItem[] = [];
  @Prop blocked: BlockedUserItem[] = [];
  unblockUserAction: (userId: string) => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State private pressedMemberKey: string = '';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private openProfile(userId: string): void {
    const params = new ProfileRouteParam();
    params.userId = userId;
    AppRouter.push(PageNames.PROFILE, params);
  }

  @Builder
  private buildUserLine(userId: string, nickname: string, meta: string, actionLabel?: string, action?: () => void) {
    Row({ space: DesignConstants.SPACING.MD }) {
      SystemAvatar({ name: nickname, avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL })
      Column({ space: DesignConstants.SPACING.XS }) {
        Text(nickname)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
        if (meta) {
          Text(meta)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().textSecondary)
        }
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)
      // 成员信息区按压反馈（对齐 tap-97 语义）
      .scale(this.pressedMemberKey === userId ?
        { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
        { x: 1, y: 1 })
      .animation({ curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      ) })
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          this.pressedMemberKey = userId;
        } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
          this.pressedMemberKey = '';
        }
      })
      .onClick(() => this.openProfile(userId))

      if (actionLabel && action) {
        SystemButton({
          label: actionLabel,
          variant: 'surface-accent',
          buttonSize: 'small',
          onPress: () => { action(); }
        })
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    .backgroundColor(this.getPalette().surfaceSubtle)
    .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
  }

  build() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Text(this.title)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getPalette().textPrimary)
        .width('100%')
      if (this.subtitle) {
        Text(this.subtitle)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().textSecondary)
          .width('100%')
      }

      if (!(this.friends.length > 0 || this.following.length > 0 || this.followers.length > 0 || this.blocked.length > 0)) {
        Text(this.emptyText)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getPalette().textSecondary)
          .width('100%')
      }

      if (this.friends.length > 0) {
        ForEach(this.friends, (item: RelatedUserItem) => {
          this.buildUserLine(item.user.user.user_id, item.user.user.nickname, `成为好友于 ${formatDateTime(item.followed_back_at)}`)
        }, (item: RelatedUserItem) => item.user.user.user_id)
      }

      if (this.following.length > 0) {
        ForEach(this.following, (item: RelatedUserItem) => {
          this.buildUserLine(item.user.user.user_id, item.user.user.nickname, `关注于 ${formatDateTime(item.followed_at)}`)
        }, (item: RelatedUserItem) => item.user.user.user_id)
      }

      if (this.followers.length > 0) {
        ForEach(this.followers, (item: RelatedUserItem) => {
          this.buildUserLine(item.user.user.user_id, item.user.user.nickname, `关注你于 ${formatDateTime(item.followed_at)}`)
        }, (item: RelatedUserItem) => item.user.user.user_id)
      }

      if (this.blocked.length > 0) {
        ForEach(this.blocked, (item: BlockedUserItem) => {
          this.buildUserLine(
            item.user.user_id,
            item.user.nickname,
            item.reason ? `原因：${item.reason}` : `拉黑于 ${formatDateTime(item.blocked_at)}`,
            '解除',
            () => this.unblockUserAction(item.user.user_id)
          )
        }, (item: BlockedUserItem) => item.user.user_id)
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
  }
}

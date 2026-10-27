import { AppIcon } from '../../../components/SystemIcon';
import { DesignConstants, lineHeightFor, textLineHeight } from '../../../constants/DesignConstants';
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
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { SystemButton } from '../../../components/SystemButton';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemToggle } from '../../../components/SystemToggle';
import { AppRouter, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { LucideIcon } from '../../../components/LucideIcon';
import { ListGroup, ListRowDivider } from '../../../components/ListGroup';

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
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column({ space: DesignConstants.SPACING.SM }) {
      if (this.title.length > 0) {
        Text(this.title)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textSecondary)
          .width('100%')
      }

      // r3 软岛：无描边、圆角 20、85% surface 底 + 轻投影（BRIEF A）；
      // 分组感由容器承担、行间用 hairline 分隔线，行本身不得再叠第二层卡片。
      Column({ space: DesignConstants.SPACING.HALF }) {
        this.content();
      }
      .width('100%')
      .padding(DesignConstants.SPACING.SM)
      .backgroundColor(this.getPalette().surfaceIsland)
      .borderRadius(DesignConstants.SHAPE.CARD)
      .shadow(this.getPalette().surfaceShadow)
      .clip(true)
    }
    .width('100%')
  }
}

/** 分组内行间 hairline 分隔线（对齐前端 ListContainer embedded divide-y）。 */
@Component
export struct SettingsRowDivider {
  /** 行内分隔线内缩（iOS inset grouped 语言）：默认 0 全宽；与行 padding 同值时起点对齐文字 */
  @Prop startMargin: number = 0;
  @Prop endMargin: number = 0;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    // r3 软岛行分隔改为 gap（Lead 口径：岛内行用间距，不加 1px 横线）
    Column()
      .width('100%')
      .height(DesignConstants.SPACING.HALF)
      .accessibilityLevel('no')
  }
}

/**
 * 主题三选一分段卡（星际/清野/水墨，对齐前端 settings 主题分段选择器）。
 * 独立 @Component：此前是 SettingsPage 的带参 @Builder 经 SettingsSectionCard 的
 * @BuilderParam 渲染，跨组件作用域下 onClick 不触发，必须拆成组件。
 */
@Component
export struct SettingsThemeSegmentCard {
  @Prop mode: string = 'vibrant';
  @Prop label: string = '';
  @Prop swatch: string = '';
  @Prop selected: boolean = false;
  onSelect: (mode: string) => void = () => {
  };
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row({ space: DesignConstants.SPACING.SM }) {
      Column()
        .width(DesignConstants.SPACING.LG)
        .height(DesignConstants.SPACING.LG)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.swatch)
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })

      Text(this.label)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        // 选中态底色是 accentMuted（淡底 + accent 描边），文字必须用 accent 本色
        // （accentText 是 accent 亮实底上的反色前景，配淡底会不可读）
        .fontColor(this.selected ?
          this.getPalette().textAccentOnSoft :
          this.getPalette().textSecondary)
    }
    .layoutWeight(1)
    .justifyContent(FlexAlign.Center)
    .padding({ top: DesignConstants.SPACING.SM2, bottom: DesignConstants.SPACING.SM2 })
    .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
    .backgroundColor(this.selected ?
      this.getPalette().accentMuted :
      DesignConstants.COLOR.TRANSPARENT)
    .border({
      width: this.selected ? DesignConstants.BORDER_WIDTH.HAIRLINE : 0,
      color: this.getPalette().accentBackground
    })
    .responseRegion({
      x: 0,
      y: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - 32) / 2,
      width: '100%',
      height: DesignConstants.LAYOUT.MIN_HIT_TARGET
    })
    .onClick(() => { this.onSelect(this.mode); })
  }
}

@Component
export struct SettingsEntryRow {
  @Prop item: SettingsEntryItem;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @State private isPressed: boolean = false;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        LucideIcon({
            icon: this.item.icon,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
            color: this.getPalette().textAccentOnSoft
          })
      }
      .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getPalette().accentMuted)
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      Column({ space: DesignConstants.SPACING.XS }) {
        Text(this.item.title)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
        if (this.item.subtitle) {
          // 单行 + 省略：分组列表里同组条目必须等高（若允许 2 行，长副标题会把该行
          // 撑高一截，整组行高参差，并让行的点击热区中心偏移）
          Text(this.item.subtitle)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getPalette().textSecondary)
            // 副标题允许两行：单行 + ellipsis 会把「消息、提及与新动态提醒（仅本机）」截成「（…」，
            // 括号只露一半比直接断行更难看（2026-10-21 端上走查）。
            .maxLines(2)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)

      SurfaceIconButton({
        icon: AppIcon.ARROW_RIGHT,
        accessibilityLabel: this.item.title,
        kind: 'neutral',
        buttonSize: 28,
        glyphSize: 14,
        onPress: this.item.action
      })
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
    // 行只保留按压填充：分组外框与行间分隔线由 SettingsSectionCard（ListGroup 契约）承担。
    // 原为行级白卡背景 + 圆角（getListItemCardStyle），导致「一行一张浮卡」。
    .backgroundColor(this.isPressed ? this.getPalette().surfaceSubtle : DesignConstants.COLOR.TRANSPARENT)
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

/**
 * 设置区块标题（对齐前端 settings/SettingsSection.svelte 左标题栏）：
 *   <h2 class="text-sm font-semibold text-text-primary">标题</h2>
 *   <p class="text-xs text-text-secondary mt-1 leading-relaxed">描述</p>
 * 原实现是 uppercase eyebrow（10/medium/0.12em 字距）且无描述，四项都与前端不同。
 */
@Component
export struct SettingsGroupLabel {
  @Prop label: string = '';
  /** 区块描述（前端 SettingsSection.description，text-xs + mt-1 + leading-relaxed 1.625） */
  @Prop description: string = '';
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column({ space: 0 }) {
      Text(this.label)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getPalette().textPrimary)
        .width('100%')
      if (this.description.length > 0) {
        Text(this.description)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getPalette().textSecondary)
          .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.XS, 1.625))
          .width('100%')
          .margin({ top: DesignConstants.SPACING.XS })
      }
    }
    .alignItems(HorizontalAlign.Start)
    .width('100%')
    // 分组标题与分组卡左边缘严格对齐：横向留白只由容器 gutter 提供
    .padding({
      top: DesignConstants.SPACING.MD,
      bottom: DesignConstants.SPACING.SM
    })
  }
}

/**
 * 设置开关行（对齐前端 SettingRow + Toggle：36vp icon-badge + 标题/描述 + 右侧 50×28 iOS 开关）。
 * 整行与开关均可触发 onToggle；无箭头（前端 Toggle 行无 chevron）。
 */
@Component
export struct SettingsToggleRow {
  @Prop icon: AppIcon = AppIcon.SETTINGS;
  @Prop title: string = '';
  @Prop subtitle: string = '';
  @Prop isOn: boolean = false;
  @Prop pending: boolean = false;
  onToggle: (next: boolean) => void = () => {
  };
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        LucideIcon({
            icon: this.icon,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
            color: this.getPalette().textAccentOnSoft
          })
      }
      .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getPalette().accentMuted)
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      Column({ space: DesignConstants.SPACING.XS }) {
        Text(this.title)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
        if (this.subtitle) {
          // 与 SettingsEntryRow 同契约：分组内条目等高，副标题单行省略
          Text(this.subtitle)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getPalette().textSecondary)
            // 与 SettingsEntryRow 一致：副标题允许两行，避免「（仅本机）」这类括号被截成「（…」
            .maxLines(2)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)

      SystemToggle({
        // 开关名称：无此则屏幕阅读器只播「已开启」，用户不知道开的是什么
        label: this.title,
        isOn: this.isOn,
        disabled: this.pending,
        onToggle: (next: boolean) => { this.onToggle(next); }
      })
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
    // 同 SettingsEntryRow：行不再自带卡片背景/圆角，容器（SettingsSectionCard）承担分组框
    .onClick(() => {
      if (!this.pending) {
        this.onToggle(!this.isOn);
      }
    })
  }
}

@Component
export struct SettingsMetaCard {
  @Prop meta: ApiVersionMeta | null = null;
  @Prop loading: boolean = false;
  @Prop error: string = '';
  @Prop lastChecked: string = '';
  refreshAction: () => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

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

  /** R12：状态胶囊文字用 on-soft 状态色（web text-status-*-on-soft 同口径） */
  private getStatusTextColor(): string {
    if (!this.meta) {
      return this.getPalette().textSecondary;
    }
    switch (this.meta.status) {
      case "deprecated":
        return this.getPalette().textStatusWarningOnSoft;
      case "sunset":
        return this.getPalette().textStatusDangerOnSoft;
      default:
        return this.getPalette().textStatusOnlineOnSoft;
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
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.LG))
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().textPrimary)
          Text(this.meta?.status === 'sunset' ? '可能影响正常使用' : '与服务端版本保持同步')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getPalette().textSecondary)
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)

        Text(this.getStatusText())
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getStatusTextColor())
          .backgroundColor(this.getStatusBackground())
          .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
          .borderRadius(DesignConstants.SHAPE.CHIP)

        SurfaceIconButton({
          icon: AppIcon.REFRESH,
          accessibilityLabel: '刷新',
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
          Text('正在获取版本信息…')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
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
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
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
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row({ space: DesignConstants.SPACING.MD }) {
      Text(this.label)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
        .fontColor(this.getPalette().textSecondary)
        .layoutWeight(1)
      Text(this.value)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
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
  /** true = 被外层 SettingsSectionCard 包裹（自身不再加 padding，避免卡中卡双层留白） */
  @Prop embedded: boolean = false;
  openAll: () => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

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
          // 与外层区块标题（SettingsGroupLabel: text-sm semibold）拉开层级但不再用
          // 16/medium 的「第二套区块标题」——卡内标题统一 text-sm medium + text-xs 描述
          Text('系统公告')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().textPrimary)
          Text(this.items.length > 0 ? '查看最近平台通知与维护提醒' : '当前暂无系统公告')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
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
        ForEach(this.maxItems > 0 ? this.items.slice(0, this.maxItems) : this.items, (item: SystemAnnouncementItem, index: number) => {
          if (index > 0) {
            ListRowDivider({ startMargin: DesignConstants.SPACING.MD, endMargin: DesignConstants.SPACING.MD })
          }
          Column({ space: DesignConstants.SPACING.SM2 }) {
            Row({ space: DesignConstants.SPACING.SM }) {
              Text(item.title)
                .fontSize(DesignConstants.FONT.SIZE.MD)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                .fontColor(this.getPalette().textPrimary)
                .layoutWeight(1)
                .maxLines(1)
                .textOverflow({ overflow: TextOverflow.Ellipsis })
              Text(item.severity === 'error' ? '重要' : item.severity === 'warning' ? '提醒' : '通知')
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
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
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
              .fontColor(this.getPalette().textTertiary)
              .width('100%')
          }
          .width('100%')
          .padding({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
          // 嵌套白卡移除：条目改为「行 + 分隔线」，避免卡中卡
        }, (item: SystemAnnouncementItem) => item.announcement_id)
      }
    }
    .width('100%')
    // embedded：被 SettingsSectionCard 包裹时左右留白由行本身承担，
    // 这里去掉整卡 padding，避免「卡中卡」双层留白
    .padding(this.embedded
      ? { top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD }
      : DesignConstants.SPACING.LG)
  }
}

@Component
export struct SupportConfigCard {
  @Prop config: ClientConfig | null = null;
  /** true = 被外层 SettingsSectionCard 包裹（自身去掉整卡 padding，避免卡中卡） */
  @Prop embedded: boolean = false;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Text('支持与服务信息')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
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
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontColor(this.getPalette().textSecondary)
          .width('100%')
      }

      if (buildSupportLinks(this.config).length > 0) {
        Column({ space: DesignConstants.SPACING.SM }) {
          ForEach(buildSupportLinks(this.config), (item: SupportLinkItem, index: number) => {
            if (index > 0) {
              ListRowDivider({ startMargin: 0, endMargin: 0 })
            }
            SettingsInfoLine({ label: item.title, value: item.value })
          }, (item: SupportLinkItem) => `${item.title}-${item.value}`)
        }
        .width('100%')
        // 嵌套白卡移除：信息行改用分隔线分组
      }
    }
    .width('100%')
    .padding(this.embedded
      ? { top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD }
      : DesignConstants.SPACING.LG)
  }
}

@Component
export struct ToggleSettingRow {
  @Prop item: SettingToggleItem;
  @Prop isOn: boolean = false;
  onChange: (value: boolean) => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row({ space: DesignConstants.SPACING.MD }) {
      Column({ space: DesignConstants.SPACING.XS }) {
        Text(this.item.title)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
        Text(this.item.subtitle)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontColor(this.getPalette().textSecondary)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)

      // 原生 Toggle 迁移为 SystemToggle 原语；label 使读屏播报「{名称}，状态」
      SystemToggle({
        label: this.item.title,
        isOn: this.isOn,
        onToggle: this.onChange
      })
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
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row() {
        Column({ space: DesignConstants.SPACING.XS }) {
          Text('当前会话')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.LG))
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().textPrimary)
          Text('查看已登录设备，并可撤销其他会话。')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
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
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontColor(this.getPalette().textSecondary)
          .width('100%')
      } else {
        ForEach(this.sessions, (item: SessionItem, index: number) => {
          if (index > 0) {
            ListRowDivider({ startMargin: DesignConstants.SPACING.MD, endMargin: DesignConstants.SPACING.MD })
          }
          Row({ space: DesignConstants.SPACING.MD }) {
            Column({ space: DesignConstants.SPACING.XS }) {
              Text(formatSessionLabel(item))
                .fontSize(DesignConstants.FONT.SIZE.MD)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                .fontColor(this.getPalette().textPrimary)
              Text(`创建于 ${formatDateTime(item.created_at)}`)
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                .fontColor(this.getPalette().textSecondary)
              Text(`最近活跃 ${formatDateTime(item.last_active_at)}`)
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
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
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                .fontColor(this.getPalette().accentBackground)
                .backgroundColor(this.getPalette().surfaceSubtle)
                .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
                .borderRadius(DesignConstants.RADIUS.FULL)
            }
          }
          .width('100%')
          .padding({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
          // 嵌套白卡移除
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
  /** 空态引导 CTA（r37：仅文案无下一步路径易流失），可选 */
  @Prop emptyActionLabel: string = '';
  emptyAction?: () => void = undefined;
  /** embedded 模式（对齐前端 ProfileUserSection embedded）：隐藏 section header，仅渲染内容区 */
  @Prop embedded: boolean = false;
  @Prop friends: RelatedUserItem[] = [];
  @Prop following: RelatedUserItem[] = [];
  @Prop followers: RelatedUserItem[] = [];
  @Prop blocked: BlockedUserItem[] = [];
  unblockUserAction: (userId: string) => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
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
  private buildUserLine(userId: string, nickname: string, meta: string, online: boolean, actionLabel?: string, action?: () => void) {
    Row({ space: DesignConstants.SPACING.MD }) {
      Stack({ alignContent: Alignment.BottomEnd }) {
        SystemAvatar({ name: nickname, avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL, decorative: true })
        if (online) {
          Row()
            .width(DesignConstants.SPACING.MD)
            .height(DesignConstants.SPACING.MD)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor(this.getPalette().success)
            .border({ width: DesignConstants.BORDER_WIDTH.STRONG, color: this.getPalette().surfaceBackground })
        }
      }
      .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
      .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
      Column({ space: DesignConstants.SPACING.XS }) {
        Text(nickname)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
        if (meta) {
          Text(meta)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
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
    // 关系列表行同样收敛为「行 + 分隔线」：原为每行 surfaceSubtle 圆角卡 + 描边，
    // 连续条目会形成一叠浮块；分组感交给外层容器（ListGroup 契约）。
    .backgroundColor(this.pressedMemberKey === userId ? this.getPalette().surfaceSubtle : DesignConstants.COLOR.TRANSPARENT)
  }

  build() {
    Column({ space: DesignConstants.SPACING.MD }) {
      // embedded 模式隐藏 header（对齐前端 embedded：section 头不渲染，标题在页面层）
      if (!this.embedded) {
        Text(this.title)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.LG))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
          .width('100%')
        if (this.subtitle) {
          Text(this.subtitle)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getPalette().textSecondary)
            .width('100%')
        }
      }

      if (!(this.friends.length > 0 || this.following.length > 0 || this.followers.length > 0 || this.blocked.length > 0)) {
        // 空态：居中图标 + 文案（对齐前端 EmptyState 居中布局，避免顶部大片空白）
        Column({ space: DesignConstants.SPACING.MD }) {
          Row() {
            LucideIcon({
              icon: AppIcon.GROUP,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_MD,
              paletteRole: 'textSecondary'
            })
          }
          .width(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
          .height(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getPalette().surfaceSubtle)
          .justifyContent(FlexAlign.Center)

          Text(this.emptyText)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getPalette().textSecondary)
            .textAlign(TextAlign.Center)

          if ((this.emptyActionLabel ?? '').length > 0 && this.emptyAction) {
            SystemButton({
              label: this.emptyActionLabel,
              variant: 'primary',
              buttonSize: 'default',
              onPress: () => {
                this.emptyAction?.();
              }
            })
          }
        }
        .width('100%')
        .alignItems(HorizontalAlign.Center)
        .padding({ top: DesignConstants.SPACING.XXL, bottom: DesignConstants.SPACING.XXL })
      }

      // 分组容器承担外框 + 行间分隔线（ListGroup 契约），行本身不再自带卡片样式
      ListGroup({ groupPadding: DesignConstants.SPACING.XS }) {
        if (this.friends.length > 0) {
          ForEach(this.friends, (item: RelatedUserItem, index: number) => {
            this.buildUserLine(item.user.user.user_id, item.user.user.nickname, `成为好友于 ${formatDateTime(item.followed_back_at ?? undefined)}`, item.user.presence_status === 'online')
          }, (item: RelatedUserItem) => item.user.user.user_id)
        }

        if (this.following.length > 0) {
          ForEach(this.following, (item: RelatedUserItem, index: number) => {
            this.buildUserLine(item.user.user.user_id, item.user.user.nickname, `关注于 ${formatDateTime(item.followed_at)}`, item.user.presence_status === 'online')
          }, (item: RelatedUserItem) => item.user.user.user_id)
        }

        if (this.followers.length > 0) {
          ForEach(this.followers, (item: RelatedUserItem, index: number) => {
            this.buildUserLine(item.user.user.user_id, item.user.user.nickname, `关注你于 ${formatDateTime(item.followed_at)}`, item.user.presence_status === 'online')
          }, (item: RelatedUserItem) => item.user.user.user_id)
        }

        if (this.blocked.length > 0) {
          ForEach(this.blocked, (item: BlockedUserItem, index: number) => {
            this.buildUserLine(
              item.user.user_id,
              item.user.nickname,
              item.reason ? `原因：${item.reason}` : `拉黑于 ${formatDateTime(item.blocked_at)}`,
              false,
              '解除',
              () => this.unblockUserAction(item.user.user_id)
            )
          }, (item: BlockedUserItem) => item.user.user_id)
        }
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
  }
}
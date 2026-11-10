/**
 * ExternalProfileView —— 他人资料页视图（加载中 / 错误 / 内容三态）。
 * 对齐 web users/[userId] 移动端：
 *   浮动返回/更多（由 ProfilePage 叠加）→ 20 圆角身份卡（头像 68 + 名字 20/600 +
 *   在线 chip + @handle + bio + 位置/兴趣标签 + 三段统计 + 关注/私信动作行）
 *   → 「最近瞬间」2 列 3:4 图墙 / 空态。
 * 资料字段以扁平 props 注入，主题按 resolvedThemeId 自解析，动作回调上抛；无业务状态。
 */

import { curves } from '@kit.ArkUI';
import { DesignConstants, lineHeightFor, textLineHeight } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemButton } from '../../../components/SystemButton';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState } from '../../../components/EmptyState';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { MomentItem } from '../../../core/types/api';
import { DEFAULT_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { MotionUtil } from '../../../core/accessibility/MotionUtil';
import { ImageUrlUtil } from '../../../core/utils/ImageUrlUtil';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { cleanPlainText } from '../../../core/utils/FormatUtil';
import { AppRouter, MomentDetailRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';

@Component
export struct ExternalProfileView {
  @Prop resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  /** 顶部浮层工具行占位（ProfilePage 传入：安全区 + 36 圆钮 + 间距） */
  @Prop topInset: number = 0;
  @Prop @Watch('onLoadingChanged') loading: boolean = false;
  /** R7 内容进场（加载完成；缓存命中不播） */
  @State private enterOpacity: number = 1;
  @State private enterOffset: number = 0;
  private enterPlayed: boolean = false;
  @State private followPopped: boolean = false;
  @Prop errorMessage: string = '';
  @Prop hasProfile: boolean = false;
  @Prop avatarUrl: string = '';
  @Prop nickname: string = '';
  @Prop handle: string = '';
  @Prop locationName: string = '';
  @Prop bio: string = '';
  @Prop isOnline: boolean = false;
  /** 在线状态是否已知（presence_status !== null）；未知时不渲染在线/离线 chip */
  @Prop presenceKnown: boolean = true;
  @Prop tags: string[] = [];
  @Prop momentsCount: string = '0';
  @Prop followersCount: string = '0';
  @Prop followingCount: string = '0';
  @Prop moments: MomentItem[] = [];
  @Prop isFollowing: boolean = false;
  @Prop followPending: boolean = false;
  @Prop safeAreaBottom: number = 0;
  onRetry: () => void = () => {};
  onPrivateChat: () => void = () => {};
  onToggleFollow: () => void = () => {};
  onOpenRemark: () => void = () => {};

  @State private loadedGalleryUrls: string[] = [];
  @State private failedGalleryUrls: string[] = [];

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private onLoadingChanged(): void {
    if (!this.loading) {
      this.resetContentEnter();
      this.playContentEnter();
    }
  }

  private playContentEnter(): void {
    if (this.enterPlayed) return;
    this.enterPlayed = true;
    if (MotionUtil.reduced()) {
      this.enterOpacity = 1;
      this.enterOffset = 0;
      return;
    }
    this.enterOpacity = 0;
    this.enterOffset = MotionUtil.enterOffset();
    MotionUtil.contentEnter(this.getUIContext(), () => {
      this.enterOpacity = 1;
      this.enterOffset = 0;
    });
  }

  private resetContentEnter(): void {
    this.enterPlayed = false;
    this.enterOpacity = 1;
    this.enterOffset = 0;
  }

  private isGalleryLoaded(url: string): boolean {
    return this.loadedGalleryUrls.indexOf(url) >= 0;
  }

  private markGalleryLoaded(url: string): void {
    if (!url || this.isGalleryLoaded(url)) return;
    const next: string[] = this.loadedGalleryUrls.slice();
    next.push(url);
    this.loadedGalleryUrls = next;
  }

  private chunkGallery(): MomentItem[][] {
    const rows: MomentItem[][] = [];
    for (let i = 0; i < this.moments.length; i += 2) {
      rows.push(this.moments.slice(i, i + 2));
    }
    return rows;
  }

  private openMoment(moment: MomentItem): void {
    if (!moment.moment_id) return;
    const params = new MomentDetailRouteParam();
    params.momentId = moment.moment_id;
    AppRouter.push(PageNames.MOMENT_DETAIL, params);
  }

  @Builder
  private statItem(value: string, label: string) {
    Column({ space: DesignConstants.SPACING.SM2 }) {
      Text(value)
        .fontSize(DesignConstants.FONT.SIZE.XXL)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XXL))
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getPalette().textPrimary)
        .maxLines(1)
      Text(label)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
        .fontColor(this.getPalette().textSecondary)
    }
    .alignItems(HorizontalAlign.Center)
    .layoutWeight(1)
  }

  @Builder
  private statDivider() {
    Column()
      .width(DesignConstants.BORDER_WIDTH.HAIRLINE)
      .height(DesignConstants.SPACING.XXL)
      .backgroundColor(this.getPalette().floatingBorderColor)
  }

  @Builder
  private buildOnlineChip() {
    Row({ space: DesignConstants.SPACING.XS }) {
      Column()
        .width(DesignConstants.COMPONENT.INDICATOR_DOT_SIZE)
        .height(DesignConstants.COMPONENT.INDICATOR_DOT_SIZE)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getPalette().online)
      Text(this.isOnline ? '在线' : '离线')
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.isOnline ? this.getPalette().textStatusOnlineOnSoft : this.getPalette().textSecondary)
    }
    .padding({
      left: DesignConstants.SPACING.SM2,
      right: DesignConstants.SPACING.SM2,
      top: DesignConstants.SPACING.HALF,
      bottom: DesignConstants.SPACING.HALF
    })
    .backgroundColor(this.isOnline
      ? AppThemeManager.getSoftStatusBackground('success')
      : this.getPalette().surfaceSubtle)
    .borderRadius(DesignConstants.RADIUS.FULL)
  }

  @Builder
  private buildGalleryItem(moment: MomentItem) {
    Stack({ alignContent: Alignment.BottomStart }) {
      Column() {
        LucideIcon({
          icon: AppIcon.IMAGES,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_MD,
          color: this.getPalette().textTertiary
        })
      }
      .width('100%')
      .height('100%')
      .justifyContent(FlexAlign.Center)
      .backgroundColor(this.getPalette().surfaceSubtle)

      if (moment.images && moment.images.length > 0 && this.failedGalleryUrls.indexOf(moment.images[0].url) < 0) {
        Image(ImageUrlUtil.optimized(MediaUrlUtil.normalize(moment.images[0].url), 800))
          .width('100%')
          .height('100%')
          .objectFit(ImageFit.Cover)
          .opacity(this.isGalleryLoaded(moment.images[0].url) || MotionUtil.reduced() ? 1 : 0)
          .animation({
            duration: MotionUtil.reduced() ? 0 : DesignConstants.ANIMATION.DURATION_SHORT,
            curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
          })
          .onComplete(() => { this.markGalleryLoaded(moment.images[0].url); })
          .onError(() => {
            const next: string[] = this.failedGalleryUrls.slice();
            next.push(moment.images[0].url);
            this.failedGalleryUrls = next;
          })
      } else if (!moment.images || moment.images.length === 0) {
        Text(cleanPlainText(moment.description))
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontColor(this.getPalette().textSecondary)
          .maxLines(3)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .padding(DesignConstants.SPACING.SM)
      }
    }
    .width('100%')
    .aspectRatio(0.75)
    .borderRadius(DesignConstants.RADIUS.MD)
    .clip(true)
    .backgroundColor(this.getPalette().surfaceSubtle)
    .accessibilityText('查看瞬间图片')
    .onClick(() => { this.openMoment(moment); })
  }

  @Builder
  private buildIdentityCard() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.LG }) {
        SystemAvatar({
          avatarUrl: this.avatarUrl,
          name: this.nickname,
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.PROFILE,
          shape: 'circle',
          isOnline: this.isOnline
        })

        Column({ space: DesignConstants.SPACING.SM2 }) {
          Row({ space: DesignConstants.SPACING.SM }) {
            Text(this.nickname || '未设置昵称')
              .fontSize(DesignConstants.FONT.SIZE.XL)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XL))
              .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
              .fontColor(this.getPalette().textPrimary)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
              .layoutWeight(1)
            if (this.presenceKnown) {
              this.buildOnlineChip()
            }
          }
          .width('100%')
          .alignItems(VerticalAlign.Center)

          if (this.handle.length > 0) {
            Text(`@${this.handle}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
              .fontColor(this.getPalette().textSecondary)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
          }
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      Text(this.bio || '这个人还没有填写简介。')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.SM, DesignConstants.FONT.LINE_RATIO.CARD))
        .fontColor(this.getPalette().textSecondary)
        .width('100%')

      Row({ space: DesignConstants.SPACING.MD3 }) {
        Row({ space: DesignConstants.SPACING.SM2 }) {
          LucideIcon({
            icon: AppIcon.LOCATION,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            color: this.getPalette().accentBackground
          })
          Text(this.locationName || '未设置位置')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getPalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .alignItems(VerticalAlign.Center)

        if (this.tags.length > 0) {
          Flex({ wrap: FlexWrap.Wrap }) {
            ForEach(this.tags.slice(0, 4), (tag: string) => {
              Text(`#${tag}`)
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                .fontColor(this.getPalette().textAccentOnSoft)
                .backgroundColor(this.getPalette().accentMuted)
                .padding({
                  left: DesignConstants.SPACING.SM2,
                  right: DesignConstants.SPACING.SM2,
                  top: DesignConstants.SPACING.HALF,
                  bottom: DesignConstants.SPACING.HALF
                })
                .borderRadius(DesignConstants.RADIUS.FULL)
                .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.XS })
            }, (tag: string) => tag)
          }
          .layoutWeight(1)
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      Column()
        .width('100%')
        .height(DesignConstants.BORDER_WIDTH.HAIRLINE)
        .backgroundColor(this.getPalette().floatingBorderColor)

      Row() {
        this.statItem(this.momentsCount, '瞬间')
        this.statDivider()
        this.statItem(this.followersCount, '粉丝')
        this.statDivider()
        this.statItem(this.followingCount, '关注')
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      // 动作行对齐 web：已关注时「私信」为主操作、「已关注」安静档；未关注时「关注」为主。
      Row({ space: DesignConstants.SPACING.SM }) {
        if (this.isFollowing) {
          SystemButton({
            label: '私信',
            variant: 'primary',
            buttonSize: 'default',
            showIcon: true,
            icon: AppIcon.MESSAGE,
            onPress: () => { this.onPrivateChat(); }
          })
            .layoutWeight(1)

          SystemButton({
            label: '已关注',
            variant: 'surface',
            buttonSize: 'default',
            showIcon: true,
            icon: AppIcon.PERSON_ADD,
            disabled: this.followPending,
            onPress: () => { this.onToggleFollow(); }
          })
            .layoutWeight(1)
        } else {
          SystemButton({
            label: '关注',
            variant: 'primary',
            buttonSize: 'default',
            showIcon: true,
            icon: AppIcon.PERSON_ADD,
            disabled: this.followPending,
            onPress: () => {
              MotionUtil.statePop(this.getUIContext(),
                () => { this.followPopped = true; },
                () => { this.followPopped = false; });
              this.onToggleFollow();
            }
          })
            .layoutWeight(1)
            .scale({ x: MotionUtil.popScale(this.followPopped), y: MotionUtil.popScale(this.followPopped) })

          SystemButton({
            label: '私信',
            variant: 'surface',
            buttonSize: 'default',
            showIcon: true,
            icon: AppIcon.MESSAGE,
            onPress: () => { this.onPrivateChat(); }
          })
            .layoutWeight(1)
        }
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor(this.getPalette().surfaceIsland)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .shadow(this.getPalette().surfaceShadow)
  }

  @Builder
  private buildGallerySection() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.SM2 }) {
        LucideIcon({
          icon: AppIcon.PHOTO,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          color: this.getPalette().accentBackground
        })
        Text('最近瞬间')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
          .fontColor(this.getPalette().textPrimary)
        if (this.moments.length > 0) {
          Text(`${this.moments.length}`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getPalette().textSecondary)
        }
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)

      if (this.moments.length === 0) {
        Column({ space: DesignConstants.SPACING.SM }) {
          Column() {
            LucideIcon({
              icon: AppIcon.PHOTO,
              iconSize: DesignConstants.COMPONENT.STATUS_ICON_SIZE,
              color: this.getPalette().accentBackground
            })
          }
          .width(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
          .height(DesignConstants.COMPONENT.EMPTY_ICON_SIZE)
          .justifyContent(FlexAlign.Center)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getPalette().accentMuted)

          Text('还没有带图瞬间')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getPalette().textPrimary)
          Text('带图瞬间会展示在这里；文字瞬间暂不计入画廊。')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(this.getPalette().textSecondary)
            .textAlign(TextAlign.Center)
            .width('100%')
        }
        .width('100%')
        .alignItems(HorizontalAlign.Center)
        .padding({
          top: DesignConstants.SPACING.XXL,
          bottom: DesignConstants.SPACING.XXL
        })
      } else {
        Column({ space: DesignConstants.SPACING.SM }) {
          ForEach(this.chunkGallery(), (row: MomentItem[], rowIndex: number) => {
            Row({ space: DesignConstants.SPACING.SM }) {
              ForEach(row, (moment: MomentItem) => {
                Column() {
                  this.buildGalleryItem(moment)
                }
                .layoutWeight(1)
              }, (moment: MomentItem) => `gallery-${moment.moment_id}`)
              if (row.length === 1) {
                Blank()
                  .layoutWeight(1)
              }
            }
            .width('100%')
            .alignItems(VerticalAlign.Top)
          }, (row: MomentItem[], rowIndex: number) => `gallery-row-${rowIndex}`)
        }
        .width('100%')
      }
    }
    .width('100%')
    .margin({ top: DesignConstants.PAGE_SHELL.SECTION_GAP })
  }

  build() {
    if (this.loading) {
      SystemLoadingView({ message: '正在加载用户资料…' })
        .width('100%')
        .height('100%')
    } else if (this.errorMessage.length > 0 || !this.hasProfile) {
      EmptyState({
        bare: true,
        icon: AppIcon.WARNING,
        message: this.errorMessage.length > 0 ? this.errorMessage : '未找到该用户',
        actionLabel: '重试',
        onAction: () => { this.onRetry(); }
      })
      .width('100%')
      .height('100%')
    } else {
      Scroll() {
        Column() {
          this.buildIdentityCard()
          this.buildGallerySection()
        }
        .width('100%')
        .alignItems(HorizontalAlign.Start)
        .padding({
          top: this.topInset,
          bottom: this.safeAreaBottom + DesignConstants.SPACING.LG
        })
        .opacity(this.enterOpacity)
        .translate({ y: this.enterOffset })
      }
      .width('100%')
      .height('100%')
      .align(Alignment.TopStart)
      .scrollBar(BarState.Off)
    }
  }
}

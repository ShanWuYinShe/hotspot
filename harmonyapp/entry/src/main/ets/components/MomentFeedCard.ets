import { MomentItem, MomentImage, UserInfo } from '../core/types/api';
import { interactionActions } from '../core/state/InteractionActions';
import { MomentSocialState } from '../core/state/InteractionStateStore';
import { SystemAvatar } from './SystemAvatar';
import { AppIcon, IconButton, IconSize } from './SystemIcon';
import { DesignConstants } from '../constants/DesignConstants';
import { MediaUrlUtil } from '../core/utils/MediaUrlUtil';
import { SystemShareService } from '../core/services/SystemShareService';
import { common } from '@kit.AbilityKit';
import { AppRouter, MomentDetailRouteParam, ProfileRouteParam } from '../core/router/AppRouter';
import { PageNames } from '../constants/AppConstants';
import { SystemCard, CardVariant } from './SystemCard';
import { IntelligentLocationCard } from './IntelligentLocationCard';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

@Component
export struct MomentFeedCard {
  @Prop moment: MomentItem;
  @Prop variant: string = 'default';
  @Prop imageMode: string = 'preview';
  @Prop showTags: boolean = false;
  @Prop showLocationCard: boolean = false;
  @Prop showTime: boolean = false;
  @Prop showFavorite: boolean = true;
  @Prop textMaxLines: number = 3;
  @Prop textFontSize: number = DesignConstants.FONT.SIZE.MD;
  onOpenDetail?: (moment: MomentItem) => void;
  onImagePreview?: (momentId: string, images: string[], index: number) => void;
  @StorageLink('interaction_moment_state') private momentStateMap: Record<string, MomentSocialState> = {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private formatCount(count?: number): string {
    const value = count || 0;
    if (value >= 10000) {
      return `${(value / 10000).toFixed(1)}w`;
    }
    if (value >= 1000) {
      return `${(value / 1000).toFixed(1)}k`;
    }
    return value.toString();
  }

  private formatTime(timeStr: string): string {
    const time = new Date(timeStr);
    const now = new Date();
    if (time.toDateString() === now.toDateString()) {
      return `${time.getHours().toString().padStart(2, '0')}:${time.getMinutes().toString().padStart(2, '0')}`;
    }
    const yesterday = new Date(now);
    yesterday.setDate(yesterday.getDate() - 1);
    if (time.toDateString() === yesterday.toDateString()) {
      return '昨天';
    }
    if (time.getFullYear() === now.getFullYear()) {
      return `${(time.getMonth() + 1).toString().padStart(2, '0')}-${time.getDate().toString().padStart(2, '0')}`;
    }
    return time.toLocaleDateString();
  }

  private onLikeClick(moment: MomentItem): void {
    interactionActions.toggleMomentLike(moment.moment_id);
  }

  private onFavoriteClick(moment: MomentItem): void {
    interactionActions.toggleMomentFavorite(moment.moment_id);
  }

  private openMomentDetail(moment: MomentItem): void {
    if (this.onOpenDetail) {
      this.onOpenDetail(moment);
      return;
    }
    const params = new MomentDetailRouteParam();
    params.momentId = moment.moment_id;
    AppRouter.push(PageNames.MOMENT_DETAIL, params);
  }

  private openUserProfile(user: UserInfo | undefined): void {
    const userId = user?.user_id;
    if (!userId) {
      return;
    }
    const params = new ProfileRouteParam();
    params.userId = userId;
    AppRouter.push(PageNames.PROFILE, params);
  }

  private getMomentState(): MomentSocialState | undefined {
    return this.momentStateMap[this.moment.moment_id];
  }

  private async shareMoment(moment: MomentItem): Promise<void> {
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const content = SystemShareService.buildMomentShareContent(
        moment.description,
        moment.user?.nickname ?? '匿名用户',
        moment.moment_id
      );
      await SystemShareService.getInstance().shareText(context, content);
    } catch (error) {
      // User cancelled or share failed - non-fatal
    }
  }

  build() {
    if (this.variant === 'postCard') {
      this.buildPostCard()
    } else {
      Column() {
        SystemCard({
          variant: CardVariant.DEFAULT,
          cardPadding: DesignConstants.SPACING.LG,
          cardBorderRadius: DesignConstants.RADIUS.XXL,
          cardBorderColor: this.getPalette().borderColor,
          cardBackgroundColor: this.getPalette().surfaceBackground,
          clickable: true
        }) {
          this.cardContent()
        }
      }
      .width('100%')
      .onClick(() => this.openMomentDetail(this.moment))
    }
  }

  @Builder
  private buildPostCard(): void {
    Column() {
      Stack() {
        if (this.moment.images && this.moment.images.length > 0) {
          Image(MediaUrlUtil.normalize(this.moment.images[0].url))
            .width('100%')
            .aspectRatio(16 / 9)
            .objectFit(ImageFit.Cover)
        } else {
          Column()
            .width('100%')
            .aspectRatio(16 / 9)
            .backgroundColor(this.getPalette().surfaceSubtle)
        }

        Column()
          .width('100%')
          .height('100%')
          .linearGradient({
            direction: GradientDirection.Bottom,
            colors: [['rgba(0,0,0,0.04)', 0], ['rgba(0,0,0,0.44)', 1]]
          })

        Column() {
          if (this.moment.tags && this.moment.tags.length > 0) {
            Text(this.moment.tags[0])
              .fontSize(10)
              .fontColor(Color.White)
              .backgroundColor(this.getPalette().accentBackground)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .padding({ left: 8, right: 8, top: 2, bottom: 2 })
              .shadow({
                radius: 8,
                color: this.getPalette().accentShadow.color,
                offsetX: 0,
                offsetY: 2
              })
          }

          Blank()

          Row({ space: 6 }) {
            SystemAvatar({
              avatarUrl: this.moment.user?.avatar_url ?? '',
              name: this.moment.user?.nickname ?? '',
              avatarSize: 40,
              shape: 'circle'
            })

            Text(this.moment.user?.nickname ?? '')
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontWeight(FontWeight.Medium)
              .fontColor(Color.White)
              .layoutWeight(1)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })

            if (this.moment.location?.location_name) {
              Text() {
                SymbolSpan($r(AppIcon.LOCATION))
                  .fontSize(9)
                  .fontColor([Color.White])
              }
              Text(this.moment.location.location_name)
                .fontSize(10)
                .fontColor(Color.White)
                .opacity(0.8)
                .maxLines(1)
                .textOverflow({ overflow: TextOverflow.Ellipsis })
            }
          }
          .width('100%')
          .alignItems(VerticalAlign.Center)
          .padding({ left: 12, right: 12 })
        }
        .width('100%')
        .height('100%')
        .padding({ left: 0, right: 0, top: 10, bottom: 10 })
      }
      .width('100%')
      .borderRadius({ topLeft: DesignConstants.RADIUS.XXL, topRight: DesignConstants.RADIUS.XXL })
      .clip(true)

      if (this.moment.description) {
        Text(this.moment.description)
          .fontSize(13)
          .fontColor(this.getPalette().textSecondary)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .lineHeight(20)
          .padding({ left: 16, right: 16, top: 12, bottom: 4 })
          .width('100%')
      }

      Row() {
        Row({ space: DesignConstants.SPACING.MD }) {
          Row({ space: DesignConstants.SPACING.XS }) {
            Text() {
              SymbolSpan($r(AppIcon.HEART))
                .fontSize(15)
                .fontColor([this.getMomentState()?.my_state?.is_liked
                  ? this.getPalette().accentBackground
                  : this.getPalette().textSecondary])
            }
            .onClick(() => this.onLikeClick(this.moment))
            Text(this.formatCount(this.getMomentState()?.likes_count ?? this.moment.likes_count))
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getPalette().textSecondary)
          }

          Row({ space: DesignConstants.SPACING.XS }) {
            Text() {
              SymbolSpan($r(AppIcon.MESSAGE))
                .fontSize(15)
                .fontColor([this.getPalette().textSecondary])
            }
            .onClick(() => this.openMomentDetail(this.moment))
            Text(this.formatCount(this.getMomentState()?.comments_count ?? this.moment.comments_count))
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getPalette().textSecondary)
          }
        }

        Row({ space: 12 }) {
          Text() {
            SymbolSpan($r(AppIcon.STAR_FILL))
              .fontSize(15)
              .fontColor([this.getMomentState()?.my_state?.is_favorited
                ? this.getPalette().accentBackground
                : this.getPalette().textSecondary])
          }
          .onClick(() => this.onFavoriteClick(this.moment))

          Text() {
            SymbolSpan($r(AppIcon.SHARE))
              .fontSize(15)
              .fontColor([this.getPalette().textSecondary])
          }
          .onClick(() => this.shareMoment(this.moment))
        }
      }
      .width('100%')
      .justifyContent(FlexAlign.SpaceBetween)
      .padding({ left: 16, right: 16, top: 4, bottom: 14 })
    }
    .width('100%')
    .borderRadius(DesignConstants.RADIUS.XXL)
    .backgroundColor(this.getPalette().surfaceBackground)
    .shadow(this.getPalette().elevatedShadow)
    .border({ width: 1, color: this.getPalette().borderColor })
    .onClick(() => this.openMomentDetail(this.moment))
  }

  @Builder
  private cardContent(): void {
    Column({ space: DesignConstants.SPACING.MD }) {
      // Header
      this.headerSection()

      // Content
      this.textSection()

      // Images
      this.imageSection()

      // Tags
      this.tagsSection()

      // Location card
      this.locationSection()

      // Action bar
      this.actionBar()
    }
  }

  @Builder
  private headerSection(): void {
    Row({ space: DesignConstants.SPACING.SM }) {
      SystemAvatar({
        avatarUrl: this.moment.user?.avatar_url ?? '',
        name: this.moment.user?.nickname ?? '',
        avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
        shape: 'circle'
      })
        .onClick(() => this.openUserProfile(this.moment.user))

      Column({ space: 2 }) {
        Text(this.moment.user?.nickname ?? '匿名用户')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getPalette().textPrimary)

        if (this.showTime && this.moment.created_at) {
          Text(this.formatTime(this.moment.created_at))
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().textSecondary)
        } else if (this.moment.location?.location_name) {
          Text(this.moment.location.location_name)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().textSecondary)
        }
      }
      .layoutWeight(1)

      if (this.showFavorite) {
        IconButton({
          icon: AppIcon.STAR_FILL,
          iconSize: IconSize.MEDIUM,
          iconColor: this.getMomentState()?.my_state?.is_favorited
            ? this.getPalette().accentBackground
            : this.getPalette().textSecondary,
          onButtonClick: () => this.onFavoriteClick(this.moment)
        })
      }
    }
  }

  @Builder
  private textSection(): void {
    if (this.textMaxLines === 0) {
      Text(this.moment.description)
        .fontSize(this.textFontSize)
        .fontColor(this.getPalette().textPrimary)
        .lineHeight(24)
    } else {
      Text(this.moment.description)
        .fontSize(this.textFontSize)
        .fontColor(this.getPalette().textPrimary)
        .maxLines(this.textMaxLines)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
    }
  }

  @Builder
  private imageSection(): void {
    if (this.moment.images && this.moment.images.length > 0) {
      if (this.imageMode === 'grid') {
        this.imageGrid()
      } else {
        this.imagePreview()
      }
    }
  }

  @Builder
  private imageGrid(): void {
    Scroll() {
      Row({ space: 8 }) {
        ForEach((this.moment.images ?? []).slice(0, 4), (img: MomentImage, imgIndex: number) => {
          Image(MediaUrlUtil.normalize(img.url))
            .width(80)
            .height(80)
            .objectFit(ImageFit.Cover)
            .borderRadius(DesignConstants.RADIUS.MD)
            .backgroundColor(this.getPalette().surfaceSubtle)
            .accessibilityText(`图片 ${imgIndex + 1}，共 ${(this.moment.images ?? []).length} 张`)
            .geometryTransition('moment_img_' + this.moment.moment_id + '_' + imgIndex)
            .onClick(() => {
              if (this.onImagePreview) {
                const urls = (this.moment.images ?? []).map((m: MomentImage) => m.url);
                this.onImagePreview!(this.moment.moment_id, urls, imgIndex);
              } else {
                this.openMomentDetail(this.moment);
              }
            })
        }, (img: MomentImage) => img.url)

        if ((this.moment.images ?? []).length > 4) {
          Text(`+${(this.moment.images ?? []).length - 4}`)
            .width(80)
            .height(80)
            .textAlign(TextAlign.Center)
            .fontSize(13)
            .fontColor(this.getPalette().textSecondary)
            .borderRadius(DesignConstants.RADIUS.MD)
            .border({ width: 1, color: this.getPalette().borderColor })
            .onClick(() => this.openMomentDetail(this.moment))
        }
      }
    }
    .scrollable(ScrollDirection.Horizontal)
    .scrollBar(BarState.Off)
    .width('100%')
  }

  @Builder
  private imagePreview(): void {
    if (this.moment.images && this.moment.images.length > 0) {
      Stack({ alignContent: Alignment.BottomEnd }) {
        Image(MediaUrlUtil.normalize(this.moment.images[0].url))
          .width('100%')
          .aspectRatio(1.6)
          .objectFit(ImageFit.Cover)
          .borderRadius(DesignConstants.SHAPE.CARD)
          .backgroundColor(this.getPalette().surfaceSubtle)
          .accessibilityText('动态图片')

        if (this.moment.images.length > 1) {
          Text(`+${this.moment.images.length - 1}`)
            .fontSize(11)
            .fontWeight(FontWeight.Bold)
            .fontColor(Color.White)
            .padding({ left: 8, right: 8, top: 4, bottom: 4 })
            .borderRadius(DesignConstants.RADIUS.FULL)
            .border({ width: 1, color: 'rgba(255,255,255,0.24)' })
            .backgroundColor('rgba(15,23,42,0.58)')
            .margin({ right: 10, bottom: 10 })
        }
      }
      .width('100%')
      .onClick(() => this.openMomentDetail(this.moment))
    }
  }

  @Builder
  private tagsSection(): void {
    if (this.showTags && this.moment.tags && this.moment.tags.length > 0) {
      Flex({ wrap: FlexWrap.Wrap }) {
        ForEach(this.moment.tags, (tag: string) => {
          Text(`#${tag}`)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().accentBackground)
            .backgroundColor(this.getPalette().surfaceSubtle)
            .borderRadius(DesignConstants.RADIUS.SM)
            .padding({
              left: DesignConstants.SPACING.SM,
              right: DesignConstants.SPACING.SM,
              top: DesignConstants.SPACING.XS,
              bottom: DesignConstants.SPACING.XS
            })
            .margin({ right: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
        }, (tag: string) => tag)
      }
    }
  }

  @Builder
  private locationSection(): void {
    if (this.showLocationCard && this.moment.location) {
      IntelligentLocationCard({
        latitude: this.moment.location.point.latitude,
        longitude: this.moment.location.point.longitude,
        locationName: this.moment.location.location_name,
        distance: this.moment.location.distance ?? -1
      })
    }
  }

  @Builder
  private actionBar(): void {
    Row() {
      Row({ space: DesignConstants.SPACING.XL }) {
        Row({ space: DesignConstants.SPACING.XS }) {
          IconButton({
            icon: this.getMomentState()?.my_state?.is_liked ? AppIcon.HEART_FILL : AppIcon.HEART,
            iconSize: IconSize.MEDIUM,
            iconColor: this.getMomentState()?.my_state?.is_liked
              ? this.getPalette().accentBackground
              : this.getPalette().textSecondary,
            onButtonClick: () => this.onLikeClick(this.moment)
          })
          Text(this.formatCount(this.getMomentState()?.likes_count ?? this.moment.likes_count))
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().textSecondary)
        }

        Row({ space: DesignConstants.SPACING.XS }) {
          IconButton({
            icon: AppIcon.MESSAGE,
            iconSize: IconSize.MEDIUM,
            iconColor: this.getPalette().textSecondary,
            onButtonClick: () => this.openMomentDetail(this.moment)
          })
          Text(this.formatCount(this.getMomentState()?.comments_count ?? this.moment.comments_count))
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getPalette().textSecondary)
        }
      }

      Blank()

      Row({ space: DesignConstants.SPACING.MD }) {
        IconButton({
          icon: this.getMomentState()?.my_state?.is_favorited ? AppIcon.STAR_FILL : AppIcon.STAR,
          iconSize: IconSize.MEDIUM,
          iconColor: this.getMomentState()?.my_state?.is_favorited
            ? this.getPalette().accentBackground
            : this.getPalette().textSecondary,
          onButtonClick: () => this.onFavoriteClick(this.moment)
        })
        IconButton({
          icon: AppIcon.SHARE,
          iconSize: IconSize.MEDIUM,
          iconColor: this.getPalette().textSecondary,
          onButtonClick: () => this.shareMoment(this.moment)
        })
      }
    }
    .width('100%')
  }
}

/**
 * ExternalProfileView —— 他人资料页视图（加载中 / 错误 / 内容三态）。
 * 从 ProfilePage.buildExternalProfileView + ViewedStatItem 抽取；资料字段以扁平 props 注入
 * （避免跨文件传可空复合对象），主题按 resolvedThemeId 自解析（与 ProfileContentAtoms 同规），
 * 动作回调上抛；无本地状态。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemButton } from '../../../components/SystemButton';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState } from '../../../components/EmptyState';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { MomentItem } from '../../../core/types/api';
import { ViewedMomentThumb } from './ViewedMomentThumb';
import { DEFAULT_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

@Component
export struct ExternalProfileView {
  @Prop resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @Prop loading: boolean = false;
  @Prop errorMessage: string = '';
  @Prop hasProfile: boolean = false;
  @Prop avatarUrl: string = '';
  @Prop nickname: string = '';
  @Prop locationName: string = '';
  @Prop bio: string = '';
  @Prop isOnline: boolean = false;
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

  @Builder
  private statItem(value: string, label: string) {
    Column({ space: DesignConstants.SPACING.HALF }) {
      Text(value)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
      Text(label)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
    }
    .alignItems(HorizontalAlign.Center)
    .layoutWeight(1)
  }

  build() {
    if (this.loading) {
      SystemLoadingView({ message: '正在加载用户资料…' })
        .layoutWeight(1)
    } else if (this.errorMessage.length > 0 || !this.hasProfile) {
      EmptyState({
        bare: true,
        icon: AppIcon.WARNING,
        message: this.errorMessage.length > 0 ? this.errorMessage : '未找到该用户',
        actionLabel: '重试',
        onAction: () => { this.onRetry(); }
      })
      .layoutWeight(1)
    } else {
      // 对齐前端 users/[userId] 移动端：左对齐身份行 + bio + 标签 + 三段统计
      // + 近期瞬间九宫格；底部操作条 私信 + 关注 + 备注
      Column() {
        Scroll() {
          Column({ space: DesignConstants.SPACING.MD }) {
            Row({ space: DesignConstants.SPACING.MD }) {
              Stack({ alignContent: Alignment.BottomEnd }) {
                SystemAvatar({
                  avatarUrl: this.avatarUrl,
                  name: this.nickname,
                  avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.LARGE,
                  shape: 'circle',
                  isOnline: this.isOnline
                })
              }
              .width(DesignConstants.COMPONENT.AVATAR_SIZE.LARGE)
              .height(DesignConstants.COMPONENT.AVATAR_SIZE.LARGE)

              Column({ space: DesignConstants.SPACING.XS }) {
                Text(this.nickname || '未知用户')
                  .fontSize(DesignConstants.FONT.SIZE.XXL)
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
                  .maxLines(1)
                  .textOverflow({ overflow: TextOverflow.Ellipsis })
                Row({ space: DesignConstants.SPACING.XS }) {
                  LucideIcon({
                    icon: AppIcon.LOCATION,
                    iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                    color: AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground
                  })
                  Text(this.locationName || '未设置位置')
                    .fontSize(DesignConstants.FONT.SIZE.XS)
                    .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
                    .maxLines(1)
                    .textOverflow({ overflow: TextOverflow.Ellipsis })
                }
                .alignItems(VerticalAlign.Center)
              }
              .alignItems(HorizontalAlign.Start)
              .layoutWeight(1)
            }
            .width('100%')
            .alignItems(VerticalAlign.Center)

            Text(this.bio || '这个人很神秘，暂时还没有留下简介')
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .lineHeight(DesignConstants.FONT.LINE_HEIGHT.MD)
              .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
              .width('100%')

            if (this.tags.length > 0) {
              Flex({ wrap: FlexWrap.Wrap }) {
                ForEach(this.tags.slice(0, 8), (tag: string) => {
                  Text(`#${tag}`)
                    .fontSize(DesignConstants.FONT.SIZE.XS)
                    .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground)
                    .padding({
                      left: DesignConstants.SPACING.MD,
                      right: DesignConstants.SPACING.MD,
                      top: DesignConstants.SPACING.XS,
                      bottom: DesignConstants.SPACING.XS
                    })
                    .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
                    .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).accentMuted)
                    .borderRadius(DesignConstants.RADIUS.FULL)
                }, (tag: string) => tag)
              }
              .width('100%')
            }

            Row() {
              this.statItem(this.momentsCount, '瞬间')
              this.statItem(this.followersCount, '粉丝')
              this.statItem(this.followingCount, '关注')
            }
            .width('100%')
            .margin({ top: DesignConstants.SPACING.SM })

            // 近期瞬间九宫格（无瞬间时给空态，避免内容区整片空白——2026-10-20 端上走查实测
            // 0 瞬间用户的页面出现大段空白、没有任何说明，看起来像页面坏了）
            if (this.moments.length === 0) {
              Column() {
                Text('TA 还没有发布瞬间')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
              }
              .width('100%')
              .padding({ top: DesignConstants.SPACING.LG2, bottom: DesignConstants.SPACING.LG2 })
              .alignItems(HorizontalAlign.Center)
            } else {
              Column({ space: DesignConstants.SPACING.SM }) {
                Text('近期瞬间')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)

                Grid() {
                  ForEach(this.moments, (moment: MomentItem) => {
                    GridItem() {
                      ViewedMomentThumb({
                        moment: moment,
                        palette: AppThemeManager.getPaletteById(this.resolvedThemeId)
                      })
                    }
                  }, (moment: MomentItem) => moment.moment_id)
                }
                .columnsTemplate('1fr 1fr 1fr')
                .columnsGap(DesignConstants.SPACING.SM)
                .rowsGap(DesignConstants.SPACING.SM)
                .width('100%')
              }
              .width('100%')
              .margin({ top: DesignConstants.SPACING.SM })
            }
          }
          .width('100%')
          .justifyContent(FlexAlign.Start)
          .padding({
            left: DesignConstants.SPACING.LG2,
            right: DesignConstants.SPACING.LG2,
            top: DesignConstants.SPACING.MD,
            bottom: DesignConstants.SPACING.LG
          })
          .alignItems(HorizontalAlign.Start)
        }
        .width('100%')
        .layoutWeight(1)
        .align(Alignment.TopStart)
        .scrollBar(BarState.Off)

        // 底部操作条（对齐前端 footer snippet：私信 + 关注 下沉拇指热区；
        // 备注入口对齐前端「更多操作」sheet 的备注名称，仅已关注可保存）
        Row({ space: DesignConstants.SPACING.MD }) {
          SystemButton({
            label: '私信',
            variant: 'outline',
            buttonSize: 'small',
            onPress: () => { this.onPrivateChat(); }
          })
            .layoutWeight(1)

          SystemButton({
            label: this.isFollowing ? '已关注' : '关注',
            variant: this.isFollowing ? 'outline' : 'primary',
            buttonSize: 'small',
            disabled: this.followPending,
            onPress: () => { this.onToggleFollow(); }
          })
            .layoutWeight(1)

          SystemButton({
            label: '备注',
            variant: 'outline',
            buttonSize: 'small',
            onPress: () => { this.onOpenRemark(); }
          })
            .layoutWeight(1)
        }
        .width('100%')
        .padding({
          left: DesignConstants.SPACING.LG2,
          right: DesignConstants.SPACING.LG2,
          top: DesignConstants.SPACING.MD,
          bottom: DesignConstants.SPACING.MD + this.safeAreaBottom
        })
        .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceBackground)
        .border({
          width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE },
          color: AppThemeManager.getPaletteById(this.resolvedThemeId).borderColor
        })
      }
      .width('100%')
      .height('100%')
    }
  }
}

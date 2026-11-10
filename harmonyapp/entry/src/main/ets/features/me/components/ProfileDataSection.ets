/**
 * ProfileDataSection —— 个人中心「个人数据」分区（对齐 web ProfileMobile.svelte → SectionHead + ListContainer）
 *
 * 结构（R13 移动端个人中心）：
 *   分区头：图标 16 + 标题 14/600 + 副标题 11 次色（标题在卡外，mb-12）
 *   软岛卡：5 行「我的瞬间 / 我的群组 / 好友 / 关注 / 粉丝」
 *     行 = 图标砖（40 圆角 12、accent-soft 底 + 18 accent 图标）
 *          + 标题 14/500 + 描述 11 次色 + 右侧计数 11/500 + chevron 16
 *
 * 与 web 一致：副标题是「用途说明」（记录城市里的光影…），计数独立右对齐（「6 条」），
 * 不再把计数塞进副标题行。行间无分割线（ListGroup 软岛 gap 语言）。
 */

import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { Pressable } from '../../../components/Pressable';
import { ListGroup } from '../../../components/ListGroup';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { formatCount } from '../../../core/utils/FormatUtil';
import { DEFAULT_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

@Component
export struct ProfileDataSection {
  @Prop resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @Prop myMomentsCount: number = 0;
  @Prop myGroupsCount: number = 0;
  @Prop myFriendsCount: number = 0;
  @Prop myFollowingCount: number = 0;
  @Prop myFollowersCount: number = 0;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  @Builder
  private dataRow(icon: AppIcon, label: string, detail: string, count: string, onTap: () => void) {
    Pressable({
      onPress: () => { onTap(); },
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW,
      hoverFill: this.getPalette().surfaceSubtle
    }) {
      Row({ space: DesignConstants.SPACING.MD3 }) {
        // 图标砖（web size-9.5 rounded-sm bg-accent-soft text-accent-on-soft；本端取 40 + RADIUS.SM 等价档）
        Row() {
          LucideIcon({
            icon: icon,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_MEDIA_ACTION,
            color: this.getPalette().textAccentOnSoft
          })
        }
        .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
        .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
        .borderRadius(DesignConstants.RADIUS.SM)
        .backgroundColor(this.getPalette().accentMuted)
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)

        Column({ space: DesignConstants.SPACING.HALF }) {
          Text(label)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          Text(detail)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getPalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        Text(count)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textSecondary)
          .maxLines(1)

        LucideIcon({
          icon: AppIcon.ARROW_RIGHT,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          color: this.getPalette().textSecondary
        })
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({
        left: DesignConstants.SPACING.MD,
        right: DesignConstants.SPACING.MD,
        top: DesignConstants.SPACING.MD,
        bottom: DesignConstants.SPACING.MD
      })
    }
  }

  build() {
    Column() {
      // 分区头（web SectionHead：icon 16 + 标题 14/600 + 副标题 11；标题在卡外）
      Row({ space: DesignConstants.SPACING.SM }) {
        LucideIcon({
          icon: AppIcon.IMAGES,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          color: this.getPalette().textSecondary
        })
        Column({ space: DesignConstants.SPACING.HALF }) {
          Text('个人数据')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getPalette().textPrimary)
          Text('内容与关系资产')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getPalette().textSecondary)
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .margin({ bottom: DesignConstants.SPACING.MD })

      // 5 行内容软岛（ListGroup：surfaceIsland + 20 圆角 + surfaceShadow + p8）
      ListGroup() {
        this.dataRow(AppIcon.PHOTO, '我的瞬间', '记录城市里的光影', `${formatCount(this.myMomentsCount)} 条`, () => {
          AppRouter.push(PageNames.MOMENT);
        })
        this.dataRow(AppIcon.GROUP, '我的群组', '和同好一起探索', `${formatCount(this.myGroupsCount)} 个`, () => {
          AppRouter.push(PageNames.GROUP);
        })
        this.dataRow(AppIcon.HEART, '好友', '可以直接聊天的人', `${formatCount(this.myFriendsCount)} 位`, () => {
          AppRouter.push(PageNames.RELATIONSHIP_HUB);
        })
        this.dataRow(AppIcon.PERSON_ADD, '关注', '你订阅的创作者', `${formatCount(this.myFollowingCount)} 人`, () => {
          AppRouter.push(PageNames.RELATIONSHIP_HUB);
        })
        this.dataRow(AppIcon.PERSON, '粉丝', '关注你的人', `${formatCount(this.myFollowersCount)} 人`, () => {
          AppRouter.push(PageNames.RELATIONSHIP_HUB);
        })
      }
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    // 与上方身份卡（web mb-6=24）同节奏
    .margin({ top: DesignConstants.PAGE_SHELL.SECTION_GAP })
  }
}

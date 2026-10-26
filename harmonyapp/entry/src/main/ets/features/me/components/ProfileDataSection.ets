/**
 * ProfileDataSection —— 个人中心「个人数据」分组（分组小标 + ListGroup 数据行）。
 * 从 ProfilePage.buildProfileDataList/buildProfileDataRow 抽取；计数经 props 注入，
 * 主题按 resolvedThemeId 自解析（与 ProfileContentAtoms 同规），行内导航走 AppRouter；无本地状态。
 * 2026-10-13：手写卡片容器与行内底部 hairline 迁移为 ListGroup 原语
 * 2026-10-27（r3）：ListGroup 契约为 gap 行（gap-y-0.5），删除行分隔线
 * （对齐 HarmonyOS 列表规范：分隔线内缩对齐文字起始处，卡片圆角 COMPACT_CARD=16 列表容器档）。
 * 2026-10-17 移动端对齐：本区块只保留前端 ProfileMobile 的 5 行「个人数据」，
 *   端侧动作行（更新位置/升级账号/管理员工具台）改由 ProfilePage 另起端侧分组承载；
 *   行图标 14→16（前端 size-4）、分组小标左内边距 6→4（前端 px-1）。
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

/** 数据行文字起始缩进：行内边距 LG2(20) + 圆形徽标 XS(32) + 行间距 MD(12) */

@Component
export struct ProfileDataSection {
  @Prop resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @Prop myMomentsCount: number = 0;
  @Prop myGroupsCount: number = 0;
  @Prop myFriendsCount: number = 0;
  @Prop myFollowingCount: number = 0;
  @Prop myFollowersCount: number = 0;

  @Builder
  private dataRow(icon: AppIcon, label: string, detail: string, onTap: () => void) {
    // 对齐前端 ProfileMobile：圆形 icon-badge(32) + 左侧两行（标题/副标题）+ 右箭头；
    // 按压反馈统一走 Pressable（P3-1），键鼠 hover 淡填充由 hoverFill 提供
    Pressable({
      onPress: () => { onTap(); },
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW,
      hoverFill: AppThemeManager.getPaletteById(this.resolvedThemeId).accentMuted
    }) {
      Row({ space: DesignConstants.SPACING.MD }) {
        Row() {
          LucideIcon({
            icon: icon,
            // 前端行内 icon-badge：容器 size-8(32) + 图标 size-4(16)
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
            color: AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground
          })
        }
        .width(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
        .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).accentMuted)
        .justifyContent(FlexAlign.Center)

        Column({ space: DesignConstants.SPACING.HALF }) {
          Text(label)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
            .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          Text(detail)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        LucideIcon({
          icon: AppIcon.ARROW_RIGHT,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
          color: AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary
        })
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: DesignConstants.SPACING.LG,
        bottom: DesignConstants.SPACING.LG
      })
    }
  }

  build() {
    Column() {
      // 分组小标（对齐前端 ProfileMobile 改版：eyebrow + 副文案在列表外，列表去外框）
      Text('个人数据')
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
        .letterSpacing(DesignConstants.FONT.LETTER_SPACING.LABEL)
        .textCase(TextCase.UpperCase)
        .width('100%')
        .padding({ left: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.SM })
      Text('在这里查看你的内容与关系')
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
        .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
        .width('100%')
        .padding({ left: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.SM })

      // 列表容器原语（对齐 HarmonyOS 列表规范：单卡 + 行间内缩分隔线）
      // 副标题统一为「量词 + 数字」（2026-10-19 UI 审查，与 web ProfileMobile/ProfilePage 同口径）：
      // 原先「我的瞬间」行是动词短语、好友行重复标题，五行不成体系。
      ListGroup() {
        this.dataRow(AppIcon.PHOTO, '我的瞬间', `${formatCount(this.myMomentsCount)} 条`, () => {
          AppRouter.push(PageNames.MOMENT);
        })
        this.dataRow(AppIcon.GROUP, '我的群组', `${formatCount(this.myGroupsCount)} 个`, () => {
          AppRouter.push(PageNames.GROUP);
        })
        this.dataRow(AppIcon.PERSON, '好友', `${formatCount(this.myFriendsCount)} 位`, () => {
          AppRouter.push(PageNames.RELATIONSHIP_HUB);
        })
        this.dataRow(AppIcon.STAR, '关注', `${formatCount(this.myFollowingCount)} 人`, () => {
          AppRouter.push(PageNames.RELATIONSHIP_HUB);
        })
        this.dataRow(AppIcon.HEART, '粉丝', `${formatCount(this.myFollowersCount)} 人`, () => {
          AppRouter.push(PageNames.RELATIONSHIP_HUB);
        })
      }
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    // 前端「个人数据」分组 mt-5(20)
    .margin({ top: DesignConstants.SPACING.LG })
  }
}

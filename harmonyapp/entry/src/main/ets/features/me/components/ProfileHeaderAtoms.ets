/**
 * ProfileHeaderAtoms —— 个人中心「注册用户视图」头部原子：
 * ProfileAvatarBlock（大头像 + 相机角钮 + 昵称）/ ProfileStatsRow（三列统计卡）。
 * 从 ProfilePage.buildProfileAvatar/buildFigmaStatsRow 抽取；主题按 resolvedThemeId 自解析
 * （与 ProfileContentAtoms 同规），动作回调上抛；无本地状态。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { formatCount } from '../../../core/utils/FormatUtil';

interface ProfileStatEntry {
  label: string;
  value: string;
}

@Component
export struct ProfileAvatarBlock {
  @Prop resolvedThemeId: string = 'vibrant_light';
  @Prop avatarUrl: string = '';
  @Prop nickname: string = '';
  onChangeAvatar: () => void = () => {};

  build() {
    Column() {
      Stack({ alignContent: Alignment.BottomEnd }) {
        // 头像
        SystemAvatar({
          avatarUrl: this.avatarUrl,
          name: this.nickname,
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.XLARGE,
          shape: 'circle'
        })

        // 相机按钮
        Button() {
          LucideIcon({
            icon: AppIcon.CAMERA,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            paletteRole: 'accentText'
          })
        }
        .type(ButtonType.Normal)
        .width(DesignConstants.COMPONENT.STATUS_ICON_SIZE)
        .height(DesignConstants.COMPONENT.STATUS_ICON_SIZE)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground)
        .border({
          width: DesignConstants.BORDER_WIDTH.HEAVY,
          color: AppThemeManager.getPaletteById(this.resolvedThemeId).pageBackground
        })
        .onClick(() => this.onChangeAvatar())
        .accessibilityText('更换头像')
        .responseRegion({
          x: -DesignConstants.SPACING.SM,
          y: -DesignConstants.SPACING.SM,
          width: DesignConstants.LAYOUT.MIN_HIT_TARGET,
          height: DesignConstants.LAYOUT.MIN_HIT_TARGET
        })
      }
      .margin({ top: DesignConstants.SPACING.LG2, bottom: DesignConstants.SPACING.MD })

      Text(this.nickname)
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
        .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
    }
    .alignItems(HorizontalAlign.Center)
  }
}

@Component
export struct ProfileStatsRow {
  @Prop resolvedThemeId: string = 'vibrant_light';
  @Prop momentsCount: number = 0;
  @Prop followersCount: number = 0;
  @Prop groupsCount: number = 0;

  build() {
    Row() {
      ForEach([
        { label: '瞬间', value: formatCount(this.momentsCount) },
        { label: '关注者', value: formatCount(this.followersCount) },
        { label: '群组', value: formatCount(this.groupsCount) }
      ] as ProfileStatEntry[], (stat: ProfileStatEntry, index: number) => {
        Row() {
          if (index > 0) {
            Column()
              .width(DesignConstants.BORDER_WIDTH.HAIRLINE)
              .height(DesignConstants.COMPONENT.AVATAR_SIZE.XS)
              .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).borderColor)
              .margin({ left: 0, right: 0 })
          }
          Column({ space: DesignConstants.SPACING.XS }) {
            Text(stat.value)
              .fontSize(DesignConstants.FONT.SIZE.XXL)
              .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
              .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
            Text(stat.label)
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground)
              .textCase(TextCase.UpperCase)
              .letterSpacing(DesignConstants.FONT.LETTER_SPACING.FIELD)
              .opacity(DesignConstants.OPACITY.EMPHASIZED)
          }
        }
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
        .padding({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
      }, (stat: ProfileStatEntry) => stat.label)
    }
    .width('100%')
    .borderRadius(DesignConstants.SHAPE.CARD)
    .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceBackground)
    .shadow(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceShadow)
    .border({
      width: DesignConstants.BORDER_WIDTH.HAIRLINE,
      color: AppThemeManager.getPaletteById(this.resolvedThemeId).borderColor
    })
    .margin({ bottom: DesignConstants.SPACING.LG2 })
  }
}

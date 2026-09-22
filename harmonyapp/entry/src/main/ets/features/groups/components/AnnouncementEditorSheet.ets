import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemButton } from '../../../components/SystemButton';
import { SystemInput } from '../../../components/SystemInput';
import { SystemTextArea } from '../../../components/SystemTextArea';
import { SystemToggle } from '../../../components/SystemToggle';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { GroupAnnouncementItem } from '../../../core/types/api';

const ANNOUNCEMENT_TITLE_MAX: number = 50;
const ANNOUNCEMENT_CONTENT_MAX: number = 1000;

/**
 * 群公告编辑面板（从 GroupDetailPage.buildAnnouncementSheet 提取的子域组件）。
 *
 * 状态边界：表单四态（editing/title/content/pinned）经 @Link 与页面双向同步，
 * 保存/取消动作经回调交回页面（submitAnnouncement 承载权限校验与服务调用）。
 * 主题取值组件内自算（@StorageProp 解析主题 id），不依赖页面助手。
 */
@Component
export struct AnnouncementEditorSheet {
  @Link editing: GroupAnnouncementItem | null;
  @Link title: string;
  @Link content: string;
  @Link pinned: boolean;
  @Prop saving: boolean = false;
  onCancel: () => void = () => {};
  onSubmit: () => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @Prop safeAreaBottom: number = 0;

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceSubtle(): string {
    return this.getCurrentThemePalette().surfaceSubtle;
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  build() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        Text(this.editing === null ? '发布群公告' : '编辑群公告')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)

        SystemButton({
          label: '取消',
          variant: 'surface',
          buttonSize: 'large',
          shadowed: true,
          disabled: this.saving,
          onPress: () => { this.onCancel(); }
        })

        SystemButton({
          label: this.editing === null ? '发布' : '保存',
          variant: 'primary',
          buttonSize: 'large',
          loading: this.saving,
          onPress: () => { this.onSubmit(); }
        })
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM })

      Scroll() {
        Column({ space: DesignConstants.SPACING.LG }) {
          Column({ space: DesignConstants.SPACING.SM }) {
            SystemInput({
              value: this.title,
              placeholder: '公告标题',
              fontSize: DesignConstants.FONT.SIZE.MD,
              inputHeight: DesignConstants.COMPONENT.BUTTON_HEIGHT,
              maxLength: ANNOUNCEMENT_TITLE_MAX,
              onChange: (value: string) => { this.title = value; }
            })
              .backgroundColor(this.getSurfaceSubtle())
              .shadow(this.getCurrentThemePalette().controlShadow)
              .borderRadius(DesignConstants.RADIUS.MD)
              .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
              .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

            Text(`${this.title.trim().length}/${ANNOUNCEMENT_TITLE_MAX}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getCurrentThemePalette().textTertiary)
              .width('100%')
              .textAlign(TextAlign.End)
          }
          .width('100%')

          Column({ space: DesignConstants.SPACING.SM }) {
            SystemTextArea({
              value: this.content,
              placeholder: '公告内容…',
              minHeight: DesignConstants.COMPONENT.PLACEHOLDER_AREA_HEIGHT,
              maxLength: ANNOUNCEMENT_CONTENT_MAX,
              fontSize: DesignConstants.FONT.SIZE.MD,
              onChange: (value: string) => { this.content = value; }
            })

            Text(`${this.content.trim().length}/${ANNOUNCEMENT_CONTENT_MAX}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getCurrentThemePalette().textTertiary)
              .width('100%')
              .textAlign(TextAlign.End)
          }
          .width('100%')

          // 置顶开关（对齐前端 isPinned 置顶 pill）
          Row({ space: DesignConstants.SPACING.MD }) {
            Text('置顶这条公告')
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor(this.getCurrentThemePalette().textPrimary)
              .layoutWeight(1)
            SystemToggle({
              isOn: this.pinned,
              onToggle: (next: boolean) => { this.pinned = next; }
            })
          }
          .width('100%')
          .padding(DesignConstants.SPACING.LG)
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.RADIUS.XL)
          .shadow(this.getCurrentThemePalette().controlShadow)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
        }
        .width('100%')
        .padding({
          left: DesignConstants.SPACING.LG,
          right: DesignConstants.SPACING.LG,
          top: DesignConstants.SPACING.SM,
          bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0)
        })
      }
      .width('100%')
      .layoutWeight(1)
      .scrollBar(BarState.Off)
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }
}

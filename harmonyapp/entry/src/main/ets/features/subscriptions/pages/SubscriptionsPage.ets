import { PageNames, StorageKeys } from '../../../constants/AppConstants';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { SystemToast } from '../../../components/SystemToast';
import { PreferencesUtil } from '../../../core/storage/PreferencesUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import type { SubscriptionsData, UpdateSubscriptionsRequest } from '../../../core/types/api';

interface SubscriptionItem {
  key: keyof SubscriptionsData;
  title: string;
  description: string;
}

@Entry
@Component
struct SubscriptionsPage {
  @State private loading: boolean = true;
  @State private savingKey: string = '';
  @State private data: SubscriptionsData | null = null;

  private items: SubscriptionItem[] = [
    { key: 'new_follower_enabled', title: '新关注者', description: '有人关注你时通知' },
    { key: 'mutual_follow_enabled', title: '互相关注', description: '成为好友时通知' },
    { key: 'group_invite_enabled', title: '群组邀请', description: '收到群组邀请时通知' },
    { key: 'group_invite_response_enabled', title: '邀请回应', description: '对方接受/拒绝邀请时通知' },
    { key: 'group_application_enabled', title: '入群申请', description: '收到入群申请时通知' },
    { key: 'group_application_response_enabled', title: '申请回应', description: '你的入群申请有结果时通知' },
    { key: 'message_mention_enabled', title: '消息提及', description: '群聊中被 @ 时通知' },
    { key: 'moment_like_enabled', title: '动态点赞', description: '动态被点赞时通知' },
    { key: 'moment_comment_enabled', title: '动态评论', description: '动态被评论时通知' },
    { key: 'comment_reply_enabled', title: '评论回复', description: '评论被回复时通知' },
    { key: 'system_announcement_enabled', title: '系统公告', description: '接收系统公告通知' }
  ];

  aboutToAppear() {
    this.ensureAuthAndLoad();
  }

  private async ensureAuthAndLoad(): Promise<void> {
    try {
      const token = await PreferencesUtil.get<string>(StorageKeys.ACCESS_TOKEN, '');
      if (!token) {
        SystemToast.showWarning('请先登录', this.getUIContext());
        const routerInstance = this.getUIContext().getRouter();
        await routerInstance.pushUrl({
          url: `features/auth/pages/${PageNames.LOGIN}`,
          params: { reason: 'action_required', action: '管理通知订阅', redirectUrl: `features/subscriptions/pages/${PageNames.SUBSCRIPTIONS}` }
        });
        this.loading = false;
        return;
      }
      await this.load();
    } catch (e) {
      this.loading = false;
      SystemToast.showError('加载失败', this.getUIContext());
    }
  }

  private async load(): Promise<void> {
    this.loading = true;
    try {
      const response = await ApiClient.getSubscriptions();
      if (response.code === 0 && response.data) {
        this.data = response.data;
      } else {
        SystemToast.showError(response.message || '加载失败', this.getUIContext());
      }
    } catch (e) {
      SystemToast.showError('加载失败', this.getUIContext());
    } finally {
      this.loading = false;
    }
  }

  private getValue(key: keyof SubscriptionsData): boolean {
    if (!this.data) return false;
    const dataRecord = (this.data as Object) as Record<string, boolean>;
    return dataRecord[key as string] || false;
  }

  private setValue(key: keyof SubscriptionsData, value: boolean): void {
    if (!this.data) return;
    const dataRecord = (this.data as Object) as Record<string, boolean>;
    dataRecord[key as string] = value;
    this.data = (dataRecord as Object) as SubscriptionsData;
  }

  private async update(key: keyof SubscriptionsData, value: boolean): Promise<void> {
    if (!this.data) return;
    if (this.savingKey) return;

    const prev = this.getValue(key);
    this.savingKey = String(key);
    this.setValue(key, value);

    try {
      const payload: UpdateSubscriptionsRequest = {
        new_follower_enabled: undefined,
        mutual_follow_enabled: undefined,
        group_invite_enabled: undefined,
        group_invite_response_enabled: undefined,
        group_application_enabled: undefined,
        group_application_response_enabled: undefined,
        message_mention_enabled: undefined,
        moment_like_enabled: undefined,
        moment_comment_enabled: undefined,
        comment_reply_enabled: undefined,
        system_announcement_enabled: undefined
      };
      switch (key) {
        case 'new_follower_enabled':
          payload.new_follower_enabled = value;
          break;
        case 'mutual_follow_enabled':
          payload.mutual_follow_enabled = value;
          break;
        case 'group_invite_enabled':
          payload.group_invite_enabled = value;
          break;
        case 'group_invite_response_enabled':
          payload.group_invite_response_enabled = value;
          break;
        case 'group_application_enabled':
          payload.group_application_enabled = value;
          break;
        case 'group_application_response_enabled':
          payload.group_application_response_enabled = value;
          break;
        case 'message_mention_enabled':
          payload.message_mention_enabled = value;
          break;
        case 'moment_like_enabled':
          payload.moment_like_enabled = value;
          break;
        case 'moment_comment_enabled':
          payload.moment_comment_enabled = value;
          break;
        case 'comment_reply_enabled':
          payload.comment_reply_enabled = value;
          break;
        case 'system_announcement_enabled':
          payload.system_announcement_enabled = value;
          break;
      }
      const response = await ApiClient.updateSubscriptions(payload);
      if (response.code !== 0) {
        this.setValue(key, prev);
        SystemToast.showError(response.message || '保存失败', this.getUIContext());
      }
    } catch (e) {
      this.setValue(key, prev);
      SystemToast.showError('保存失败', this.getUIContext());
    } finally {
      this.savingKey = '';
    }
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: DesignConstants.COLOR.TEXT_PRIMARY_LIGHT,
          bgColor: Color.Transparent,
          onButtonClick: () => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.back();
          }
        })

        Text('通知订阅')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        Blank().width(48)
      }
      .width('100%')
      .height(56)
      .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      if (this.loading) {
        Column() {
          Text('加载中...')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
        .width('100%')
        .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
      } else {
        List({ space: 0 }) {
          ListItemGroup({ header: this.ListHeader('通知类型') }) {
            ForEach(this.items, (item: SubscriptionItem) => {
              ListItem() {
                Row() {
                  Column({ space: 4 }) {
                    Text(item.title)
                      .fontSize(DesignConstants.FONT.SIZE.LG)
                      .fontColor($r('sys.color.ohos_id_color_text_primary'))
                    Text(item.description)
                      .fontSize(DesignConstants.FONT.SIZE.MD)
                      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  }
                  .layoutWeight(1)
                  .alignItems(HorizontalAlign.Start)

                  Toggle({ type: ToggleType.Switch, isOn: this.getValue(item.key) })
                    .selectedColor($r('sys.color.ohos_id_color_emphasize'))
                    .onChange((isOn: boolean) => {
                      this.update(item.key, isOn);
                    })
                    .enabled(!this.savingKey || this.savingKey === String(item.key))
                }
                .width('100%')
                .padding(DesignConstants.SPACING.LG)
              }
              .backgroundColor($r('sys.color.ohos_id_color_background'))
            }, (i: SubscriptionItem) => i.key as string)
          }
          .borderRadius(DesignConstants.RADIUS.LG)
          .divider({ strokeWidth: 1, color: $r('sys.color.ohos_id_color_component_normal'), startMargin: 16, endMargin: 16 })
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.XL })
        }
        .layoutWeight(1)
        .scrollBar(BarState.Auto)
        .sticky(StickyStyle.Header)
        .width('100%')
        .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  @Builder
  private ListHeader(title: string) {
    Text(title)
      .fontSize(DesignConstants.FONT.SIZE.MD)
      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
      .margin({ left: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.LG })
      .width('100%')
  }
}


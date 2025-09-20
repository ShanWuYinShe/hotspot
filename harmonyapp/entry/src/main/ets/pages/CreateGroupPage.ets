/**
 * 创建群组页面
 * 用户可以创建新的地理位置群组
 */

import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { ApiClient } from '../common/api/ApiClient';
import { LocationUtil } from '../common/location/LocationUtil';
import { PageNames } from '../constants/AppConstants';
import { SystemLoadingView } from '../components/SystemLoadingView';
import { SystemToast } from '../components/SystemToast';
import { IconButton, AppIcon, IconSize } from '../components/SystemIcon';
import { LocationInfo } from '../common/location/LocationUtil';

@Entry
@Component
struct CreateGroupPage {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  private stateManager = GlobalStateManager.getInstance();
  @State groupName: string = '';
  @State groupDescription: string = '';
  @State groupPassword: string = '';
  @State isPasswordRequired: boolean = false;
  @State maxMembers: number = 100;
  @State isCreating: boolean = false;
  @State currentLocation: LocationInfo | null = null;

  async aboutToAppear() {
    // 订阅状态变化
    this.stateManager.subscribe('create_group_page', (state: GlobalState) => {
      this.globalState = state;
    });

    // 获取当前位置
    await this.getCurrentLocation();
  }

  aboutToDisappear() {
    this.stateManager.unsubscribe('create_group_page');
  }

  /**
   * 获取当前位置
   */
  private async getCurrentLocation(): Promise<void> {
    try {
      if (this.globalState.currentLocation) {
        this.currentLocation = {
          latitude: this.globalState.currentLocation.latitude,
          longitude: this.globalState.currentLocation.longitude,
          locationName: this.globalState.currentLocation.locationName || '南天门'
        };
      } else {
        const location = await LocationUtil.getCurrentLocation();
        if (location) {
          this.currentLocation = {
            latitude: location.latitude,
            longitude: location.longitude,
            locationName: location.locationName || '南天门'
          };
        }
      }
    } catch (error) {
      SystemToast.showError('获取位置信息失败');
    }
  }

  /**
   * 创建群组
   */
  private async createGroup(): Promise<void> {
    // 验证输入
    if (!this.groupName.trim()) {
      SystemToast.show('请输入群组名称');
      return;
    }

    if (!this.currentLocation) {
      SystemToast.show('请先获取位置信息');
      return;
    }

    if (this.isPasswordRequired && !this.groupPassword.trim()) {
      SystemToast.show('请输入群组密码');
      return;
    }

    this.isCreating = true;
    try {
      const response = await ApiClient.createGroup({
        name: this.groupName.trim(),
        description: this.groupDescription.trim() || undefined,
        latitude: this.currentLocation.latitude,
        longitude: this.currentLocation.longitude,
        location_name: this.currentLocation.locationName || "南天门",
        password: this.isPasswordRequired ? this.groupPassword.trim() : undefined
      });

      if (response.code === 0 && response.data) {
        SystemToast.showSuccess('群组创建成功');
        
        // 跳转到群组详情页
        const routerInstance = this.getUIContext().getRouter();
        await routerInstance.replaceUrl({
          url: `pages/${PageNames.GROUP_DETAIL}`,
          params: { groupId: response.data.group_id }
        });
      } else {
        SystemToast.showError(response.msg || '创建群组失败');
      }
    } catch (error) {
      SystemToast.showError('创建群组失败');
    } finally {
      this.isCreating = false;
    }
  }

  /**
   * 刷新位置
   */
  private async refreshLocation(): Promise<void> {
    try {
      const location = await LocationUtil.getCurrentLocation();
      if (location) {
        this.currentLocation = {
          latitude: location.latitude,
          longitude: location.longitude,
          locationName: location.locationName || '南天门'
        };
        SystemToast.show('位置已更新');
      }
    } catch (error) {
      SystemToast.showError('获取位置失败');
    }
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          size: IconSize.LARGE,
          backgroundColor: Color.Transparent,
          onClick: () => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.back();
          }
        })

        Text('创建群组')
          .fontSize(18)
          .fontWeight(FontWeight.Medium)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        Button('创建')
          .type(ButtonType.Normal)
          .backgroundColor(Color.Transparent)
          .fontColor($r('sys.color.ohos_id_color_emphasize'))
          .fontSize(16)
          .enabled(!this.isCreating)
          .onClick(() => this.createGroup())
      }
      .width('100%')
      .height(56)
      .padding({ left: 16, right: 16 })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      Scroll() {
        Column() {
          // 基本信息
          Column() {
            Text('基本信息')
              .fontSize(16)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .width('100%')
              .margin({ bottom: 16 })

            // 群组名称
            Column() {
              Text('群组名称 *')
                .fontSize(14)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .width('100%')
                .margin({ bottom: 8 })

              TextInput({ placeholder: '请输入群组名称' })
                .onChange((value: string) => {
                  this.groupName = value;
                })
                .maxLength(30)
            }
            .width('100%')
            .alignItems(HorizontalAlign.Start)
            .margin({ bottom: 16 })

            // 群组描述
            Column() {
              Text('群组描述')
                .fontSize(14)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .width('100%')
                .margin({ bottom: 8 })

              TextArea({ placeholder: '请输入群组描述（可选）' })
                .onChange((value: string) => {
                  this.groupDescription = value;
                })
                .maxLength(200)
                .height(80)
            }
            .width('100%')
            .alignItems(HorizontalAlign.Start)
          }
          .width('100%')
          .padding(16)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(12)
          .margin({ bottom: 16 })

          // 位置信息
          Column() {
            Row() {
              Text('位置信息')
                .fontSize(16)
                .fontWeight(FontWeight.Medium)
                .fontColor($r('sys.color.ohos_id_color_text_primary'))
                .layoutWeight(1)

              Button('刷新')
                .type(ButtonType.Normal)
                .backgroundColor(Color.Transparent)
                .fontColor($r('sys.color.ohos_id_color_emphasize'))
                .fontSize(14)
                .onClick(() => this.refreshLocation())
            }
            .width('100%')
            .margin({ bottom: 16 })

            if (this.currentLocation) {
              Row() {
                Image($r('sys.media.ohos_ic_public_location'))
                  .width(20)
                  .height(20)
                  .fillColor($r('sys.color.ohos_id_color_emphasize'))

                Column() {
                  Text(this.currentLocation.locationName)
                    .fontSize(14)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))
                    .width('100%')

                  Text(`${this.currentLocation.latitude.toFixed(6)}, ${this.currentLocation.longitude.toFixed(6)}`)
                    .fontSize(12)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .width('100%')
                    .margin({ top: 2 })
                }
                .alignItems(HorizontalAlign.Start)
                .layoutWeight(1)
                .margin({ left: 8 })
              }
              .width('100%')
              .padding(12)
              .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
              .borderRadius(8)
            } else {
              Text('正在获取位置信息...')
                .fontSize(14)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .width('100%')
                .textAlign(TextAlign.Center)
                .padding(20)
            }
          }
          .width('100%')
          .padding(16)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(12)
          .margin({ bottom: 16 })

          // 群组设置
          Column() {
            Text('群组设置')
              .fontSize(16)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .width('100%')
              .margin({ bottom: 16 })

            // 最大成员数
            Row() {
              Text('最大成员数')
                .fontSize(14)
                .fontColor($r('sys.color.ohos_id_color_text_primary'))
                .layoutWeight(1)

              Counter() {
                Text(this.maxMembers.toString())
                  .fontSize(14)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
              }
              .onInc(() => {
                if (this.maxMembers < 500) {
                  this.maxMembers += 10;
                }
              })
              .onDec(() => {
                if (this.maxMembers > 10) {
                  this.maxMembers -= 10;
                }
              })
            }
            .width('100%')
            .margin({ bottom: 16 })

            // 密码保护
            Row() {
              Text('密码保护')
                .fontSize(14)
                .fontColor($r('sys.color.ohos_id_color_text_primary'))
                .layoutWeight(1)

              Toggle({ type: ToggleType.Switch, isOn: this.isPasswordRequired })
                .onChange((isOn: boolean) => {
                  this.isPasswordRequired = isOn;
                  if (!isOn) {
                    this.groupPassword = '';
                  }
                })
            }
            .width('100%')
            .margin({ bottom: this.isPasswordRequired ? 16 : 0 })

            // 密码输入
            if (this.isPasswordRequired) {
              Column() {
                Text('群组密码 *')
                  .fontSize(14)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  .width('100%')
                  .margin({ bottom: 8 })

                TextInput({ placeholder: '请输入群组密码' })
                  .type(InputType.Password)
                  .onChange((value: string) => {
                    this.groupPassword = value;
                  })
                  .maxLength(20)
              }
              .width('100%')
              .alignItems(HorizontalAlign.Start)
            }
          }
          .width('100%')
          .padding(16)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(12)
          .margin({ bottom: 16 })

          // 创建按钮
          Button('创建群组')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .width('100%')
            .height(44)
            .enabled(!this.isCreating)
            .onClick(() => this.createGroup())
            .margin({ bottom: 32 })
        }
        .width('100%')
        .padding(16)
      }
      .layoutWeight(1)
      .scrollBar(BarState.Auto)
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }
}










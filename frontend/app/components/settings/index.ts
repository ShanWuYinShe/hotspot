// Settings 组件统一导出
export { default as AccountSection } from './AccountSection.vue';
export { default as PrivacySection } from './PrivacySection.vue';
export { default as NotificationsSection } from './NotificationsSection.vue';
export { default as AppearanceSection } from './AppearanceSection.vue';
export { default as AboutSection } from './AboutSection.vue';
export { default as DangerSection } from './DangerSection.vue';

// Settings 子组件
export { default as SettingToggleItem } from './SettingToggleItem.vue';
export { default as SettingSelectField } from './SettingSelectField.vue';
export { default as SettingSelectSimple } from './SettingSelectSimple.vue';

// 类型定义
export interface SettingOption {
  label: string;
  value: string | number;
  description?: string;
}

export interface SettingToggleColor {
  type: 'blue' | 'green' | 'purple' | 'yellow' | 'neutral' | 'red' | 'pink';
}
/**
 * 设置页面 (Server Component)
 * 静态结构服务端渲染
 */

import { getTranslations } from 'next-intl/server';
import { Sun, Globe, User } from '@/components/icons';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { ThemeSettings } from './theme-settings';
import { LanguageSettings } from './language-settings';
import { AccountSettings } from './account-settings';

export default async function SettingsPage() {
  const t = await getTranslations('settings');

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>

      {/* Appearance */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Sun className="h-5 w-5" />
            {t('theme')}
          </CardTitle>
          <CardDescription>选择你喜欢的主题</CardDescription>
        </CardHeader>
        <ThemeSettings />
      </Card>

      {/* Language */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5" />
            {t('language')}
          </CardTitle>
          <CardDescription>选择显示语言</CardDescription>
        </CardHeader>
        <LanguageSettings />
      </Card>

      {/* Account */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <User className="h-5 w-5" />
            {t('account')}
          </CardTitle>
        </CardHeader>
        <AccountSettings />
      </Card>
    </div>
  );
}

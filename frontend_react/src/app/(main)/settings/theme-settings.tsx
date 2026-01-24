'use client';

/**
 * 主题设置组件 (Client Component)
 * 需要 useTheme hook
 */

import { useTheme } from 'next-themes';
import { useTranslations } from 'next-intl';
import { Moon, Sun, Monitor } from '@/components/icons';
import { CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import type { ThemeMode } from '@/types/client';

export function ThemeSettings() {
  const t = useTranslations('settings');
  const { theme, setTheme } = useTheme();

  const themeOptions: { value: ThemeMode; label: string; icon: typeof Sun }[] = [
    { value: 'light', label: t('themeLight'), icon: Sun },
    { value: 'dark', label: t('themeDark'), icon: Moon },
    { value: 'system', label: t('themeSystem'), icon: Monitor },
  ];

  return (
    <CardContent>
      <div className="flex gap-2">
        {themeOptions.map((option) => (
          <Button
            key={option.value}
            variant={theme === option.value ? 'default' : 'outline'}
            className="flex-1 gap-2"
            onClick={() => setTheme(option.value)}
          >
            <option.icon className="h-4 w-4" />
            {option.label}
          </Button>
        ))}
      </div>
    </CardContent>
  );
}

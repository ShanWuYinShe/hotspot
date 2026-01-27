'use client';

import { ArrowLeft } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { Button } from '@/components/ui/button';

interface ChatHeaderProps {
  title: string;
  subtitle?: string;
  onBack?: () => void;
}

export function ChatHeader({ title, subtitle, onBack }: ChatHeaderProps) {
  const t = useTranslations('common');

  return (
    <div className="flex items-center gap-3 border-b pb-4">
      {onBack ? (
        <Button variant="ghost" size="icon" onClick={onBack} aria-label={t('back')}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
      ) : null}
      <div className="flex-1">
        <h2 className="text-lg font-semibold">{title}</h2>
        {subtitle ? <p className="text-sm text-muted-foreground">{subtitle}</p> : null}
      </div>
    </div>
  );
}

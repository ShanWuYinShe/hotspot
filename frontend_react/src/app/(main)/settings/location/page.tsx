'use client';

import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { useLocation } from '@/hooks/use-location';
import { UserService } from '@/lib/api/services/user';

export default function LocationSettingsPage() {
  const t = useTranslations('settings');
  const tCommon = useTranslations('common');
  const { currentLocation, requestLocation, isLoading } = useLocation();

  const handleUpdate = async () => {
    if (!currentLocation) {
      toast.error(tCommon('retry'));
      return;
    }
    const res = await UserService.updateLocation(currentLocation);
    if (res.code === 0) {
      toast.success(tCommon('saved'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>{t('sections.location')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            {currentLocation?.location_name || tCommon('loading')}
          </p>
          <div className="flex gap-3">
            <Button variant="outline" onClick={requestLocation} disabled={isLoading}>
              {tCommon('confirm')}
            </Button>
            <Button onClick={handleUpdate} disabled={!currentLocation}>
              {tCommon('save')}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

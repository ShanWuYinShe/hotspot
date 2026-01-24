/**
 * 系统公告页面 (Server Component)
 */

import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { EmptyState } from '@/components/common/empty-state';
import { Megaphone } from '@/components/icons';
import { formatRelativeTime } from '@/lib/utils';
import { PageHeader } from '@/components/layout';
import type { AnnouncementItem, PaginatedData } from '@/types/api';

export default async function AnnouncementsPage() {
  const response = await serverApi.post<PaginatedData<AnnouncementItem>>(
    ENDPOINTS.ANNOUNCEMENTS.LIST,
    { limit: 50 },
    { revalidate: 60, tags: ['announcements'] }
  );

  const items = isServerSuccess(response) ? response.data?.items ?? [] : [];
  const header = (
    <PageHeader title="系统公告" description="平台更新与重要通知" />
  );

  if (items.length === 0) {
    return (
      <div className="max-w-2xl mx-auto space-y-6">
        {header}
        <EmptyState icon={Megaphone} title="暂无公告" description="系统公告将在这里展示" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      {header}
      <div className="space-y-4">
        {items.map((announcement) => (
          <Card key={announcement.announcement_id}>
            <CardHeader>
              <div className="flex items-center justify-between gap-3">
                <CardTitle className="text-lg">{announcement.title}</CardTitle>
                <Badge variant="secondary">{announcement.severity}</Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-2">
              <p className="text-sm text-muted-foreground">{announcement.content}</p>
              <p className="text-xs text-muted-foreground">
                {formatRelativeTime(announcement.created_at)}
              </p>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}

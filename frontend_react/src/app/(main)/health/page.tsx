/**
 * 健康检查页面 (Server Component)
 */

import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import type { HealthCheckData } from '@/types/api';

function getStatusColor(status: HealthCheckData['status']) {
  switch (status) {
    case 'healthy':
      return 'bg-green-500';
    case 'degraded':
      return 'bg-yellow-500';
    case 'unhealthy':
      return 'bg-red-500';
    default:
      return 'bg-gray-500';
  }
}

export default async function HealthPage() {
  const response = await serverApi.get<HealthCheckData>(ENDPOINTS.SYSTEM.HEALTH, undefined, {
    revalidate: 30,
  });

  if (!isServerSuccess(response)) {
    return (
      <div className="max-w-lg mx-auto">
        <Card>
          <CardHeader>
            <CardTitle>系统健康检查</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            当前无法获取系统状态，请稍后再试。
          </CardContent>
        </Card>
      </div>
    );
  }

  const data = response.data;

  return (
    <div className="max-w-lg mx-auto">
      <Card>
        <CardHeader>
          <CardTitle>系统健康检查</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <div className="flex items-center justify-between">
            <span>状态</span>
            <Badge className={getStatusColor(data.status)}>{data.status}</Badge>
          </div>
          <div className="flex items-center justify-between">
            <span>版本</span>
            <span>{data.version}</span>
          </div>
          <div className="flex items-center justify-between">
            <span>时间</span>
            <span>{data.timestamp}</span>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

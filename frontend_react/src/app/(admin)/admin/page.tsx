/**
 * Admin Dashboard (Server Component)
 * 服务端获取管理员信息
 */

import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Users, UsersRound, Flag, FileText } from '@/components/icons';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { AdminPermissions } from './admin-permissions';
import { PageHeader } from '@/components/layout';
import type { AdminMeData } from '@/types/api';

async function DashboardStats() {
  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">用户总数</CardTitle>
          <Users className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">--</div>
          <p className="text-xs text-muted-foreground">需要统计接口</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">群组总数</CardTitle>
          <UsersRound className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">--</div>
          <p className="text-xs text-muted-foreground">需要统计接口</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">待处理举报</CardTitle>
          <Flag className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">--</div>
          <p className="text-xs text-muted-foreground">需要统计接口</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">审计日志</CardTitle>
          <FileText className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">--</div>
          <p className="text-xs text-muted-foreground">最近 24 小时</p>
        </CardContent>
      </Card>
    </div>
  );
}

async function AdminInfo() {
  const t = await getTranslations('admin');
  
  const response = await serverApi.post<AdminMeData>(
    ENDPOINTS.ADMIN.ME,
    {},
    { tags: ['admin-me'], revalidate: 300 }
  );

  const adminMe = isServerSuccess(response) ? response.data : null;
  const isSuperAdmin = adminMe?.permissions.includes('*') ?? false;

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('dashboard')}
        description={`欢迎回来，管理员${isSuperAdmin ? '（超级管理员）' : ''}`}
      />

      <DashboardStats />

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>快速操作</CardTitle>
            <CardDescription>常用管理功能入口</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-2">
            <a href="/admin/users" className="text-primary hover:underline">
              → 用户管理
            </a>
            <a href="/admin/reports" className="text-primary hover:underline">
              → 举报处理
            </a>
            <a href="/admin/announcements" className="text-primary hover:underline">
              → 发布公告
            </a>
          </CardContent>
        </Card>

        <AdminPermissions permissions={adminMe?.permissions ?? []} />
      </div>
    </div>
  );
}

export default function AdminDashboardPage() {
  return (
    <Suspense fallback={<LoadingState text="加载中..." />}>
      <AdminInfo />
    </Suspense>
  );
}

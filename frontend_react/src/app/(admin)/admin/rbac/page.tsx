/**
 * 权限管理页面 (Server Component)
 * 服务端获取角色和权限数据
 */

import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { RbacManager } from './rbac-manager';
import type { AdminRoleItem, AdminPermissionItem, PaginatedData } from '@/types/api';

async function RbacContent() {
  const [rolesRes, permissionsRes] = await Promise.all([
    serverApi.get<PaginatedData<AdminRoleItem>>(
      ENDPOINTS.ADMIN.ROLES,
      { limit: 100 },
      { tags: ['admin-roles'], revalidate: 300 }
    ),
    serverApi.get<AdminPermissionItem[]>(
      ENDPOINTS.ADMIN.PERMISSIONS,
      undefined,
      { tags: ['admin-permissions'], revalidate: 300 }
    ),
  ]);

  const roles = isServerSuccess(rolesRes) ? rolesRes.data?.items ?? [] : [];
  const permissions = isServerSuccess(permissionsRes) ? permissionsRes.data ?? [] : [];

  return <RbacManager roles={roles} permissions={permissions} />;
}

export default async function AdminRbacPage() {
  const t = await getTranslations('admin');

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">{t('rbac')}</h1>
      
      <Suspense fallback={<LoadingState text="加载中..." />}>
        <RbacContent />
      </Suspense>
    </div>
  );
}

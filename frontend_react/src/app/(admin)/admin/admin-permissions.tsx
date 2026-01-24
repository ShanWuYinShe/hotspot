/**
 * Admin 权限卡片 (Server Component)
 * 静态展示组件
 */

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

interface AdminPermissionsProps {
  permissions: string[];
}

export function AdminPermissions({ permissions }: AdminPermissionsProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>我的权限</CardTitle>
        <CardDescription>当前账户拥有的权限</CardDescription>
      </CardHeader>
      <CardContent>
        {permissions.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {permissions.map((perm) => (
              <span
                key={perm}
                className="inline-flex items-center rounded-md bg-primary/10 px-2 py-1 text-xs font-medium text-primary"
              >
                {perm === '*' ? '全部权限' : perm}
              </span>
            ))}
          </div>
        ) : (
          <p className="text-muted-foreground">加载中...</p>
        )}
      </CardContent>
    </Card>
  );
}

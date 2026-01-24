'use client';

/**
 * RBAC 管理组件 (Client Component)
 * 处理角色和权限管理
 */

import { useState } from 'react';
import { Plus, Shield, Lock, Pencil, Trash } from '@/components/icons';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { EmptyState } from '@/components/common/empty-state';
import { useToast } from '@/hooks/use-toast';
import type { AdminRoleItem, AdminPermissionItem } from '@/types/api';

interface RbacManagerProps {
  roles: AdminRoleItem[];
  permissions: AdminPermissionItem[];
}

export function RbacManager({ roles, permissions }: RbacManagerProps) {
  const { toast } = useToast();
  const [activeTab, setActiveTab] = useState('roles');

  const handleCreateRole = () => {
    toast.info('功能开发中', '创建角色功能即将上线');
  };

  const handleEditRole = (_roleId: string) => {
    toast.info('功能开发中', '编辑角色功能即将上线');
  };

  const handleDeleteRole = (_roleId: string) => {
    toast.info('功能开发中', '删除角色功能即将上线');
  };

  return (
    <Tabs value={activeTab} onValueChange={setActiveTab}>
      <TabsList className="grid w-full max-w-md grid-cols-2">
        <TabsTrigger value="roles" className="gap-2">
          <Shield className="h-4 w-4" />
          角色管理
        </TabsTrigger>
        <TabsTrigger value="permissions" className="gap-2">
          <Lock className="h-4 w-4" />
          权限列表
        </TabsTrigger>
      </TabsList>

      <TabsContent value="roles" className="space-y-4 mt-6">
        <div className="flex justify-end">
          <Button onClick={handleCreateRole}>
            <Plus className="h-4 w-4 mr-2" />
            创建角色
          </Button>
        </div>

        {roles.length === 0 ? (
          <EmptyState
            icon={Shield}
            title="暂无角色"
            description="点击上方按钮创建第一个角色"
          />
        ) : (
          <div className="grid gap-4">
            {roles.map((role) => (
              <Card key={role.role_id}>
                <CardHeader className="pb-2">
                  <div className="flex items-start justify-between">
                    <div>
                      <CardTitle className="text-lg flex items-center gap-2">
                        {role.name}
                      </CardTitle>
                      <CardDescription>{role.description}</CardDescription>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleEditRole(role.role_id)}
                        aria-label="编辑角色"
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDeleteRole(role.role_id)}
                        aria-label="删除角色"
                      >
                        <Trash className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex flex-wrap gap-2">
                    {role.permissions.map((perm: string) => (
                      <Badge key={perm} variant="outline">
                        {perm === '*' ? '全部权限' : perm}
                      </Badge>
                    ))}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </TabsContent>

      <TabsContent value="permissions" className="space-y-4 mt-6">
        {permissions.length === 0 ? (
          <EmptyState
            icon={Lock}
            title="暂无权限定义"
            description="权限数据将在这里显示"
          />
        ) : (
          <Card>
            <CardHeader>
              <CardTitle>系统权限</CardTitle>
              <CardDescription>所有可分配的权限列表</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3">
                {permissions.map((permission) => (
                  <div
                    key={permission.key}
                    className="flex items-center justify-between p-3 border rounded-lg"
                  >
                    <div>
                      <p className="font-medium font-mono text-sm">{permission.key}</p>
                      <p className="text-sm text-muted-foreground">
                        {permission.description}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}
      </TabsContent>
    </Tabs>
  );
}

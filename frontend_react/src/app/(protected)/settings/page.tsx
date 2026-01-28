'use client';

import * as React from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/stores/auth';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  User,
  Lock,
  Bell,
  Shield,
  HelpCircle,
  FileText,
  LogOut,
  ChevronRight,
  Smartphone,
  Globe,
} from 'lucide-react';

const settingsGroups = [
  {
    title: '账号',
    items: [
      { href: '/settings/profile', label: '编辑资料', icon: User, description: '修改头像、昵称、简介等' },
      { href: '/settings/password', label: '修改密码', icon: Lock, description: '更新登录密码' },
    ],
  },
  {
    title: '通知',
    items: [
      { href: '/settings/notifications', label: '通知设置', icon: Bell, description: '管理推送通知' },
    ],
  },
  {
    title: '隐私',
    items: [
      { href: '/settings/privacy', label: '隐私设置', icon: Shield, description: '谁可以看到我的内容' },
      { href: '/profile/blocked', label: '黑名单', icon: Shield, description: '管理已屏蔽的用户' },
    ],
  },
  {
    title: '其他',
    items: [
      { href: '/settings/devices', label: '设备管理', icon: Smartphone, description: '查看登录设备' },
      { href: '/settings/language', label: '语言', icon: Globe, description: '简体中文' },
    ],
  },
  {
    title: '关于',
    items: [
      { href: '/help', label: '帮助与反馈', icon: HelpCircle, description: '获取帮助或提交反馈' },
      { href: '/terms', label: '服务条款', icon: FileText, description: '用户协议和隐私政策' },
    ],
  },
];

export default function SettingsPage() {
  const router = useRouter();
  const { user, logout } = useAuthStore();

  const handleLogout = () => {
    logout();
    router.push('/');
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold">设置</h1>
        <p className="text-muted-foreground text-sm mt-1">
          管理账号和偏好设置
        </p>
      </div>

      {/* Profile Quick Access */}
      <Card className="shadow-soft">
        <CardContent className="p-4">
          <Link href="/profile" className="flex items-center gap-4 group">
            <Avatar className="h-16 w-16">
              <AvatarImage src={user?.avatar_url} alt={user?.nickname} />
              <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground text-xl font-semibold">
                {user?.nickname?.slice(0, 2).toUpperCase() || 'U'}
              </AvatarFallback>
            </Avatar>
            <div className="flex-1">
              <p className="text-lg font-semibold group-hover:text-primary transition-colors">
                {user?.nickname}
              </p>
              <p className="text-sm text-muted-foreground">查看个人主页</p>
            </div>
            <ChevronRight className="h-5 w-5 text-muted-foreground" />
          </Link>
        </CardContent>
      </Card>

      {/* Settings Groups */}
      {settingsGroups.map((group) => (
        <Card key={group.title} className="shadow-soft">
          <CardContent className="p-4">
            <h2 className="text-sm font-medium text-muted-foreground mb-3">
              {group.title}
            </h2>
            <div className="space-y-1">
              {group.items.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className="flex items-center gap-4 p-3 rounded-lg hover:bg-muted/50 transition-colors group"
                >
                  <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center group-hover:bg-primary/10 transition-colors">
                    <item.icon className="h-5 w-5 text-muted-foreground group-hover:text-primary transition-colors" />
                  </div>
                  <div className="flex-1">
                    <p className="font-medium">{item.label}</p>
                    <p className="text-sm text-muted-foreground">{item.description}</p>
                  </div>
                  <ChevronRight className="h-5 w-5 text-muted-foreground" />
                </Link>
              ))}
            </div>
          </CardContent>
        </Card>
      ))}

      {/* Logout Button */}
      <Card className="shadow-soft">
        <CardContent className="p-4">
          <Button
            variant="ghost"
            onClick={handleLogout}
            className="w-full justify-start gap-4 h-auto p-3 text-destructive hover:text-destructive hover:bg-destructive/10"
          >
            <div className="w-10 h-10 rounded-full bg-destructive/10 flex items-center justify-center">
              <LogOut className="h-5 w-5" />
            </div>
            <div className="text-left">
              <p className="font-medium">退出登录</p>
              <p className="text-sm opacity-70">退出当前账号</p>
            </div>
          </Button>
        </CardContent>
      </Card>

      {/* App Info */}
      <div className="text-center text-xs text-muted-foreground py-4">
        <p>HotSpot v1.0.0</p>
        <p className="mt-1">© 2024 HotSpot Team</p>
      </div>
    </div>
  );
}

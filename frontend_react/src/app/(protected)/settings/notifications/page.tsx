'use client';

import * as React from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Switch } from '@/components/ui/switch';
import { useToast } from '@/hooks/use-toast';
import {
  ArrowLeft,
  Bell,
  Heart,
  MessageCircle,
  UserPlus,
  Megaphone,
  AtSign,
} from 'lucide-react';

interface NotificationSetting {
  key: string;
  label: string;
  description: string;
  icon: React.ElementType;
  enabled: boolean;
}

export default function NotificationSettingsPage() {
  const router = useRouter();
  const { toast } = useToast();
  
  const [settings, setSettings] = React.useState<NotificationSetting[]>([
    {
      key: 'likes',
      label: '点赞通知',
      description: '有人赞了你的瞬间',
      icon: Heart,
      enabled: true,
    },
    {
      key: 'comments',
      label: '评论通知',
      description: '有人评论或回复了你',
      icon: MessageCircle,
      enabled: true,
    },
    {
      key: 'follows',
      label: '关注通知',
      description: '有新的粉丝关注你',
      icon: UserPlus,
      enabled: true,
    },
    {
      key: 'mentions',
      label: '@提醒',
      description: '有人在消息中提到你',
      icon: AtSign,
      enabled: true,
    },
    {
      key: 'system',
      label: '系统通知',
      description: '官方公告和系统消息',
      icon: Megaphone,
      enabled: true,
    },
  ]);

  const [pushEnabled, setPushEnabled] = React.useState(true);

  const handleToggle = (key: string) => {
    setSettings((prev) =>
      prev.map((s) =>
        s.key === key ? { ...s, enabled: !s.enabled } : s
      )
    );
    toast({
      title: '设置已更新',
      description: '通知偏好已保存',
    });
  };

  const handlePushToggle = () => {
    setPushEnabled(!pushEnabled);
    toast({
      title: pushEnabled ? '已关闭推送' : '已开启推送',
      description: pushEnabled
        ? '你将不会收到推送通知'
        : '你将收到实时推送通知',
    });
  };

  return (
    <div className="max-w-xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => router.back()}>
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h1 className="text-2xl font-bold">通知设置</h1>
      </div>

      {/* Push Notification Master Toggle */}
      <Card className="shadow-soft">
        <CardContent className="py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center">
                <Bell className="h-6 w-6 text-primary" />
              </div>
              <div>
                <p className="font-medium">推送通知</p>
                <p className="text-sm text-muted-foreground">
                  接收实时推送消息
                </p>
              </div>
            </div>
            <Switch
              checked={pushEnabled}
              onCheckedChange={handlePushToggle}
            />
          </div>
        </CardContent>
      </Card>

      {/* Individual Settings */}
      <Card className="shadow-soft">
        <CardHeader>
          <CardTitle className="text-lg">通知类型</CardTitle>
          <CardDescription>
            选择你想接收的通知类型
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {settings.map((setting) => (
            <div
              key={setting.key}
              className="flex items-center justify-between py-2"
            >
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center">
                  <setting.icon className="h-5 w-5 text-muted-foreground" />
                </div>
                <div>
                  <p className="font-medium">{setting.label}</p>
                  <p className="text-sm text-muted-foreground">
                    {setting.description}
                  </p>
                </div>
              </div>
              <Switch
                checked={setting.enabled && pushEnabled}
                onCheckedChange={() => handleToggle(setting.key)}
                disabled={!pushEnabled}
              />
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Tips */}
      <Card className="shadow-soft bg-muted/30">
        <CardContent className="py-4">
          <p className="text-sm text-muted-foreground">
            即使关闭推送通知，你仍然可以在 APP 内的通知中心查看所有消息。
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

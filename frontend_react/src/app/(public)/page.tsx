'use client';

import * as React from 'react';
import Link from 'next/link';
import { useAuthStore } from '@/stores/auth';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import {
  Flame,
  ArrowRight,
  MapPin,
  Users,
  Sparkles,
  MessageCircle,
  Heart,
  Compass,
} from 'lucide-react';

const features = [
  {
    icon: MapPin,
    title: '附近发现',
    description: '探索身边的精彩瞬间，发现附近有趣的人和事',
    color: 'from-orange-500 to-red-500',
    bgColor: 'bg-orange-50',
  },
  {
    icon: Sparkles,
    title: '分享生活',
    description: '记录每一个精彩瞬间，让生活更加丰富多彩',
    color: 'from-purple-500 to-pink-500',
    bgColor: 'bg-purple-50',
  },
  {
    icon: Users,
    title: '兴趣社群',
    description: '加入志同道合的群组，找到你的圈子',
    color: 'from-blue-500 to-cyan-500',
    bgColor: 'bg-blue-50',
  },
  {
    icon: MessageCircle,
    title: '即时聊天',
    description: '与朋友实时互动，分享快乐时光',
    color: 'from-green-500 to-emerald-500',
    bgColor: 'bg-green-50',
  },
];

export default function HomePage() {
  const { isAuthenticated, user } = useAuthStore();

  return (
    <div className="py-8 md:py-12">
      {/* Hero Section */}
      <section className="text-center space-y-6 mb-16">
        {/* Logo */}
        <div className="inline-flex items-center justify-center w-20 h-20 rounded-3xl bg-gradient-to-br from-primary to-primary/70 shadow-xl shadow-primary/30 mb-2">
          <Flame className="h-10 w-10 text-primary-foreground" />
        </div>

        <h1 className="text-4xl md:text-5xl lg:text-6xl font-bold tracking-tight">
          欢迎来到{' '}
          <span className="bg-gradient-to-r from-primary to-primary/70 bg-clip-text text-transparent">
            HotSpot
          </span>
        </h1>

        <p className="text-lg md:text-xl text-muted-foreground max-w-2xl mx-auto leading-relaxed">
          发现身边的精彩，与附近的人分享生活。
          <br className="hidden md:block" />
          每一个瞬间都值得被记录，每一份热情都值得被看见。
        </p>

        {/* CTA Buttons */}
        {isAuthenticated ? (
          <div className="flex flex-col sm:flex-row gap-4 justify-center pt-4">
            <p className="text-muted-foreground">
              欢迎回来，
              <span className="font-semibold text-foreground">{user?.nickname}</span>
            </p>
          </div>
        ) : (
          <div className="flex flex-col sm:flex-row gap-4 justify-center pt-4">
            <Button
              asChild
              size="lg"
              className="rounded-full px-8 shadow-lg shadow-primary/30 hover:shadow-xl hover:shadow-primary/40 transition-all"
            >
              <Link href="/register">
                立即加入
                <ArrowRight className="ml-2 h-4 w-4" />
              </Link>
            </Button>
            <Button
              asChild
              variant="outline"
              size="lg"
              className="rounded-full px-8"
            >
              <Link href="/explore">
                <Compass className="mr-2 h-4 w-4" />
                先逛逛
              </Link>
            </Button>
          </div>
        )}
      </section>

      {/* Quick Actions for Authenticated Users */}
      {isAuthenticated && (
        <section className="mb-16">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <Link href="/feed">
              <Card className="group hover:shadow-lg hover:border-primary/30 transition-all cursor-pointer">
                <CardContent className="flex flex-col items-center justify-center py-6">
                  <div className="w-12 h-12 rounded-xl bg-orange-50 flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
                    <Heart className="h-6 w-6 text-orange-500" />
                  </div>
                  <span className="font-medium">我的动态</span>
                </CardContent>
              </Card>
            </Link>
            <Link href="/explore">
              <Card className="group hover:shadow-lg hover:border-primary/30 transition-all cursor-pointer">
                <CardContent className="flex flex-col items-center justify-center py-6">
                  <div className="w-12 h-12 rounded-xl bg-blue-50 flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
                    <Compass className="h-6 w-6 text-blue-500" />
                  </div>
                  <span className="font-medium">探索发现</span>
                </CardContent>
              </Card>
            </Link>
            <Link href="/chat">
              <Card className="group hover:shadow-lg hover:border-primary/30 transition-all cursor-pointer">
                <CardContent className="flex flex-col items-center justify-center py-6">
                  <div className="w-12 h-12 rounded-xl bg-green-50 flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
                    <MessageCircle className="h-6 w-6 text-green-500" />
                  </div>
                  <span className="font-medium">消息</span>
                </CardContent>
              </Card>
            </Link>
            <Link href="/notifications">
              <Card className="group hover:shadow-lg hover:border-primary/30 transition-all cursor-pointer">
                <CardContent className="flex flex-col items-center justify-center py-6">
                  <div className="w-12 h-12 rounded-xl bg-purple-50 flex items-center justify-center mb-3 group-hover:scale-110 transition-transform">
                    <Sparkles className="h-6 w-6 text-purple-500" />
                  </div>
                  <span className="font-medium">通知</span>
                </CardContent>
              </Card>
            </Link>
          </div>
        </section>
      )}

      {/* Features Section */}
      <section className="space-y-8">
        <div className="text-center">
          <h2 className="text-2xl md:text-3xl font-bold mb-2">为什么选择 HotSpot</h2>
          <p className="text-muted-foreground">
            发现更多可能，连接更多精彩
          </p>
        </div>

        <div className="grid md:grid-cols-2 gap-6">
          {features.map((feature, index) => (
            <Card
              key={index}
              className="group overflow-hidden hover:shadow-lg transition-all duration-300 border-0 shadow-soft"
            >
              <CardContent className="p-6">
                <div className="flex items-start gap-4">
                  <div
                    className={`w-14 h-14 rounded-2xl ${feature.bgColor} flex items-center justify-center shrink-0 group-hover:scale-110 transition-transform duration-300`}
                  >
                    <feature.icon
                      className={`h-7 w-7 bg-gradient-to-br ${feature.color} bg-clip-text`}
                      style={{
                        color: feature.color.includes('orange')
                          ? '#f97316'
                          : feature.color.includes('purple')
                          ? '#a855f7'
                          : feature.color.includes('blue')
                          ? '#3b82f6'
                          : '#22c55e',
                      }}
                    />
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold mb-1">{feature.title}</h3>
                    <p className="text-muted-foreground text-sm leading-relaxed">
                      {feature.description}
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* CTA Section for Guests */}
      {!isAuthenticated && (
        <section className="mt-16 text-center">
          <Card className="bg-gradient-to-br from-primary/5 to-primary/10 border-primary/20">
            <CardContent className="py-10">
              <h2 className="text-2xl font-bold mb-3">准备好开始了吗？</h2>
              <p className="text-muted-foreground mb-6 max-w-md mx-auto">
                加入 HotSpot，发现身边的精彩世界
              </p>
              <div className="flex flex-col sm:flex-row gap-3 justify-center">
                <Button
                  asChild
                  size="lg"
                  className="rounded-full px-8"
                >
                  <Link href="/register">
                    免费注册
                    <ArrowRight className="ml-2 h-4 w-4" />
                  </Link>
                </Button>
                <Button
                  asChild
                  variant="outline"
                  size="lg"
                  className="rounded-full px-8"
                >
                  <Link href="/login">已有账号？登录</Link>
                </Button>
              </div>
            </CardContent>
          </Card>
        </section>
      )}
    </div>
  );
}

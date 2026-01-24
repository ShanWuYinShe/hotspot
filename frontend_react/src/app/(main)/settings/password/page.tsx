/**
 * 修改密码页面 (Server Component)
 */


import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ChangePasswordForm } from '@/components/settings';
import { LoginRequiredCard } from '@/components/common/login-required-card';

export default async function ChangePasswordPage() {
  // 检查认证状态
  const authResponse = await serverApi.get<{ user_id: number }>('/auth/me');
  
  if (!isServerSuccess(authResponse)) {
    return (
      <div className="max-w-lg mx-auto">
        <LoginRequiredCard actionName="修改密码" description="登录后才能修改密码。" />
      </div>
    );
  }
  
  return <ChangePasswordForm />;
}

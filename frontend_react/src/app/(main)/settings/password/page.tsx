/**
 * 修改密码页面 (Server Component)
 */


import { serverApi, isServerSuccess, isAuthError, getServerErrorMessage } from '@/lib/api/server';
import { ChangePasswordForm } from '@/components/settings';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { ServerErrorCard } from '@/components/common/server-error-card';

export default async function ChangePasswordPage() {
  // 检查认证状态
  const authResponse = await serverApi.post<{ user_id: number }>('/auth/me', {});
  
  if (!isServerSuccess(authResponse)) {
    if (isAuthError(authResponse)) {
      return (
        <div className="max-w-lg mx-auto">
          <LoginRequiredCard actionName="修改密码" description="登录后才能修改密码。" />
        </div>
      );
    }
    return (
      <div className="max-w-lg mx-auto">
        <ServerErrorCard
          title="加载失败"
          description={getServerErrorMessage(authResponse)}
          requestId={authResponse.request_id}
        />
      </div>
    );
  }
  
  return <ChangePasswordForm />;
}

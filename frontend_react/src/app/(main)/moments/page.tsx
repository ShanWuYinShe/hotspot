/**
 * Moments 页面 (Server Component)
 * 组合服务端和客户端组件
 */

import { MomentsHeader } from './moments-header';
import { MomentsList } from './moments-list';

export default function MomentsPage() {
  return (
    <div className="max-w-2xl mx-auto">
      <MomentsHeader />
      <MomentsList />
    </div>
  );
}

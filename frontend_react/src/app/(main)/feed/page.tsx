/**
 * Feed 页面重定向
 * 保持向后兼容，重定向到首页
 */

import { redirect } from 'next/navigation';

export default function FeedPage() {
  redirect('/');
}

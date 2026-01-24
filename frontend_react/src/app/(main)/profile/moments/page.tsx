/**
 * 我的瞬间页面 (Server Component)
 */

import { MyMomentsList } from '../my-moments-list';

export default function MyMomentsPage() {
  return (
    <div className="max-w-4xl mx-auto">
      <MyMomentsList initialMoments={[]} />
    </div>
  );
}

/**
 * 我的收藏页面 (Server Component)
 */

import { MyFavoritesList } from '../my-favorites-list';

export default function MyFavoritesPage() {
  return (
    <div className="max-w-4xl mx-auto">
      <MyFavoritesList initialMoments={[]} />
    </div>
  );
}

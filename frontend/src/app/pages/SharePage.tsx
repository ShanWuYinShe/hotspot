/**
 * 分享链接页面
 * 解析分享码，展示分享卡片并导航到目标内容
 */

import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router';
import { shareService } from '@/lib/api/services';
import { isSuccess } from '@/lib/api/client';
import { useMapAnchorStore } from '@/lib/stores/map-anchor';
import { toast } from 'sonner';
import type { ShareDetail } from '@/lib/types/api';
import { useDocumentTitle } from '@/lib/hooks/use-document-title';

export function SharePage() {
  const { code } = useParams<{ code: string }>();
  const navigate = useNavigate();
  const [share, setShare] = useState<ShareDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const setAnchor = useMapAnchorStore((s) => s.setAnchor);
  useDocumentTitle('分享');

  useEffect(() => {
    if (!code) {
      setError('无效的分享链接');
      setLoading(false);
      return;
    }

    shareService.get({ share_code: code }).then((res) => {
      if (isSuccess(res) && res.data) {
        setShare(res.data);
      } else {
        setError(res.message || '分享内容不存在或已过期');
      }
      setLoading(false);
    }).catch(() => {
      setError('加载失败');
      setLoading(false);
    });
  }, [code]);

  const handleOpen = () => {
    if (!share) return;
    switch (share.target_type) {
      case 'location':
        if (share.location) {
          setAnchor(share.location.latitude, share.location.longitude, share.location.name);
          navigate('/location');
        }
        break;
      case 'user':
        navigate(`/user/${share.target_id}`);
        break;
      case 'group':
        navigate(`/groups/${share.target_id}`);
        break;
      case 'moment':
        navigate(`/post/${share.target_id}`);
        break;
      default:
        toast.error('不支持的分享类型');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full" style={{ backgroundColor: 'var(--neu-bg)' }}>
        <p style={{ color: 'var(--neu-text-muted)', fontSize: 14 }}>加载中…</p>
      </div>
    );
  }

  if (error || !share) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-3" style={{ backgroundColor: 'var(--neu-bg)' }}>
        <p style={{ color: 'var(--neu-text-muted)', fontSize: 14 }}>{error || '分享内容不存在'}</p>
        <button
          onClick={() => navigate('/')}
          className="px-4 py-2 rounded-full"
          style={{ background: 'var(--neu-accent-primary)', color: 'var(--text-on-accent, white)', fontSize: 13 }}
        >
          返回首页
        </button>
      </div>
    );
  }

  const typeLabels: Record<string, string> = {
    location: '位置',
    user: '用户',
    group: '群组',
    moment: '动态',
  };

  return (
    <div className="flex flex-col items-center justify-center min-h-[80vh] px-6" style={{ backgroundColor: 'var(--neu-bg)' }}>
      <div
        className="w-full max-w-sm rounded-2xl p-6"
        style={{
          background: 'var(--neu-bg)',
          boxShadow: '6px 6px 12px var(--neu-shadow-dark), -6px -6px 12px var(--neu-shadow-light)',
          border: '1px solid var(--neu-border)',
        }}
      >
        <p style={{ fontSize: 12, color: 'var(--neu-text-muted)', marginBottom: 8 }}>
          {typeLabels[share.target_type] || '分享'}
        </p>
        <p style={{ fontSize: 18, fontWeight: 600, color: 'var(--neu-text)', marginBottom: 4 }}>
          {share.title || '分享内容'}
        </p>
        {share.description && (
          <p style={{ fontSize: 13, color: 'var(--neu-text-muted)', marginBottom: 16 }}>
            {share.description}
          </p>
        )}
        <button
          onClick={handleOpen}
          className="w-full py-2.5 rounded-full"
          style={{ background: 'var(--neu-accent-primary)', color: 'var(--text-on-accent, white)', fontSize: 14 }}
        >
          查看详情
        </button>
      </div>
    </div>
  );
}

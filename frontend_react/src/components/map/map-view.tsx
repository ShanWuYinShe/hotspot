'use client';

/**
 * 地图视图组件 (Client Component)
 * 展示附近的用户、群组、瞬间
 */

import { useEffect, useState, useCallback, useRef } from 'react';
import { useRouter } from 'next/navigation';
import Image from 'next/image';
import { Users, Compass, MessageCircle, Filter, ZoomIn, ZoomOut, Locate } from '@/components/icons';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { LoadingState } from '@/components/common/loading-state';
import { LocationRequiredCard } from '@/components/common/location-required-card';
import { useLocation } from '@/hooks/use-location';
import { useToast } from '@/hooks/use-toast';
import { mapService } from '@/lib/api/services/map';
import { isSuccess } from '@/lib/api/client';
import type { MapRenderItem, Bounds } from '@/types/api';

type DataFilter = 'all' | 'user' | 'group' | 'moment';

export function MapView() {
  const router = useRouter();
  const { toast } = useToast();
  const { currentLocation, refreshLocation, isLoading: locationLoading } = useLocation();
  
  const [items, setItems] = useState<MapRenderItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [filter, setFilter] = useState<DataFilter>('all');
  const [zoom, setZoom] = useState(14);
  const [selectedItem, setSelectedItem] = useState<MapRenderItem | null>(null);

  const mapRef = useRef<HTMLDivElement>(null);

  const getBounds = useCallback((): Bounds | null => {
    if (!currentLocation) return null;
    
    const delta = 0.1 / Math.pow(2, zoom - 10);
    return {
      north: currentLocation.latitude + delta,
      south: currentLocation.latitude - delta,
      east: currentLocation.longitude + delta,
      west: currentLocation.longitude - delta,
    };
  }, [currentLocation, zoom]);

  const fetchMapData = useCallback(async () => {
    const bounds = getBounds();
    if (!bounds) return;

    setIsLoading(true);
    try {
      const response = await mapService.render({
        viewport: {
          bounds,
          zoom,
        },
        filter: {
          data_type: filter === 'all' ? undefined : filter,
        },
      });

      if (isSuccess(response)) {
        setItems(response.data?.items ?? []);
        if (response.data?.toast_message) {
          toast.info(response.data.toast_message);
        }
      }
    } catch {
      toast.error('加载失败', '无法获取地图数据');
    } finally {
      setIsLoading(false);
    }
  }, [getBounds, zoom, filter, toast]);

  useEffect(() => {
    if (currentLocation) {
      fetchMapData();
    }
  }, [currentLocation, fetchMapData]);

  useEffect(() => {
    if (!currentLocation && !locationLoading) {
      refreshLocation(false);
    }
  }, [currentLocation, locationLoading, refreshLocation]);

  const handleZoomIn = () => setZoom((z) => Math.min(z + 1, 18));
  const handleZoomOut = () => setZoom((z) => Math.max(z - 1, 10));
  
  const handleLocate = () => {
    refreshLocation();
    toast.info('正在定位...');
  };

  const handleItemClick = (item: MapRenderItem) => {
    setSelectedItem(item);
    
    if (item.type === 'cluster') {
      handleZoomIn();
    }
  };

  const handleNavigate = (item: MapRenderItem) => {
    if (item.type === 'user' && item.entity) {
      router.push(`/users/${item.entity.payload_id}`);
    } else if (item.type === 'group' && item.entity) {
      router.push(`/groups/${item.entity.payload_id}`);
    } else if (item.type === 'moment' && item.entity) {
      router.push(`/moments/${item.entity.payload_id}`);
    }
  };

  if (locationLoading) {
    return (
      <div className="h-[calc(100vh-8rem)] flex items-center justify-center">
        <LoadingState text="正在获取位置..." />
      </div>
    );
  }

  if (!currentLocation) {
    return (
      <div className="h-[calc(100vh-8rem)] flex items-center justify-center">
        <LocationRequiredCard description="请开启位置权限以查看附近地图" />
      </div>
    );
  }

  const bounds = getBounds();
  if (!bounds) {
    return (
      <div className="h-[calc(100vh-8rem)] flex items-center justify-center">
        <LoadingState text="正在加载地图..." />
      </div>
    );
  }

  return (
    <div className="relative h-[calc(100vh-8rem)]">
      {/* 地图容器 */}
      <div
        ref={mapRef}
        className="absolute inset-0 bg-muted"
        style={{
          backgroundImage: `url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23888' fill-opacity='0.05'%3E%3Ccircle cx='30' cy='30' r='2'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E")`,
        }}
      >
        {items.map((item) => (
          <MapMarker
            key={item.id}
            item={item}
            bounds={bounds}
            isSelected={selectedItem?.id === item.id}
            onClick={() => handleItemClick(item)}
          />
        ))}

        {/* 当前位置标记 */}
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2">
          <div className="w-4 h-4 bg-primary rounded-full shadow-lg animate-pulse" />
        </div>
      </div>

      {/* 控制栏 */}
      <div className="absolute top-4 left-4 right-4 flex items-center justify-between">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="secondary" className="shadow-lg">
              <Filter className="h-4 w-4 mr-2" />
              筛选
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuLabel>显示类型</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuRadioGroup value={filter} onValueChange={(v) => setFilter(v as DataFilter)}>
              <DropdownMenuRadioItem value="all">全部</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value="user">用户</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value="group">群组</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value="moment">瞬间</DropdownMenuRadioItem>
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>

        <Badge variant="secondary" className="shadow-lg">
          {items.length} 个结果
        </Badge>
      </div>

      {/* 缩放控制 */}
      <div className="absolute right-4 top-1/2 -translate-y-1/2 flex flex-col gap-2">
        <Button variant="secondary" size="icon" className="shadow-lg" onClick={handleZoomIn} aria-label="放大">
          <ZoomIn className="h-5 w-5" />
        </Button>
        <Button variant="secondary" size="icon" className="shadow-lg" onClick={handleZoomOut} aria-label="缩小">
          <ZoomOut className="h-5 w-5" />
        </Button>
        <Button variant="secondary" size="icon" className="shadow-lg" onClick={handleLocate} aria-label="重新定位">
          <Locate className="h-5 w-5" />
        </Button>
      </div>

      {/* 选中项详情 */}
      {selectedItem && selectedItem.type !== 'cluster' && selectedItem.entity && (
        <Card className="absolute bottom-4 left-4 right-4 shadow-lg">
          <CardContent className="p-4">
            <div className="flex items-center gap-4">
              {selectedItem.entity.cover_url ? (
                <div className="relative h-12 w-12 rounded-lg overflow-hidden bg-muted">
                  <Image
                    src={selectedItem.entity.cover_url}
                    alt=""
                    fill
                    sizes="48px"
                    className="object-cover"
                  />
                </div>
              ) : (
                <div className="h-12 w-12 rounded-lg bg-muted flex items-center justify-center">
                  {selectedItem.type === 'user' && <Users className="h-6 w-6" />}
                  {selectedItem.type === 'group' && <Users className="h-6 w-6" />}
                  {selectedItem.type === 'moment' && <Compass className="h-6 w-6" />}
                </div>
              )}
              <div className="flex-1 min-w-0">
                <p className="font-medium truncate">{selectedItem.entity.title}</p>
                {selectedItem.entity.subtitle && (
                  <p className="text-sm text-muted-foreground truncate">
                    {selectedItem.entity.subtitle}
                  </p>
                )}
              </div>
              <Button onClick={() => handleNavigate(selectedItem)}>
                查看
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* 加载指示器 */}
      {isLoading && (
        <div className="absolute top-20 left-1/2 -translate-x-1/2">
          <LoadingState size="sm" />
        </div>
      )}
    </div>
  );
}

interface MapMarkerProps {
  item: MapRenderItem;
  bounds: Bounds;
  isSelected: boolean;
  onClick: () => void;
}

function MapMarker({ item, bounds, isSelected, onClick }: MapMarkerProps) {
  const latRange = bounds.north - bounds.south;
  const lngRange = bounds.east - bounds.west;
  const rawTop = latRange === 0 ? 50 : ((bounds.north - item.location.latitude) / latRange) * 100;
  const rawLeft = lngRange === 0 ? 50 : ((item.location.longitude - bounds.west) / lngRange) * 100;
  const top = Math.min(100, Math.max(0, rawTop));
  const left = Math.min(100, Math.max(0, rawLeft));

  if (item.type === 'cluster' && item.cluster) {
    return (
      <button
        className={`absolute transform -translate-x-1/2 -translate-y-1/2 transition-transform ${
          isSelected ? 'scale-110' : ''
        }`}
        style={{ top: `${top}%`, left: `${left}%` }}
        onClick={onClick}
      >
        <div className="flex items-center justify-center h-10 w-10 rounded-full bg-primary text-primary-foreground font-bold shadow-lg">
          {item.cluster.count}
        </div>
      </button>
    );
  }

  const IconComponent = item.type === 'user' ? Users :
                        item.type === 'group' ? MessageCircle :
                        Compass;

  return (
    <button
      className={`absolute transform -translate-x-1/2 -translate-y-1/2 transition-transform ${
        isSelected ? 'scale-110' : ''
      }`}
      style={{ top: `${top}%`, left: `${left}%` }}
      onClick={onClick}
    >
      <div className={`flex items-center justify-center h-8 w-8 rounded-full shadow-lg ${
        item.type === 'user' ? 'bg-blue-500' :
        item.type === 'group' ? 'bg-green-500' :
        'bg-orange-500'
      } text-white`}>
        <IconComponent className="h-4 w-4" />
      </div>
    </button>
  );
}

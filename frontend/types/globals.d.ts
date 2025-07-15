/**
 * 全局类型声明文件
 */

// PhotoSwipe库的类型声明
interface PhotoSwipeItem {
  src: string;
  w: number;
  h: number;
  title?: string;
}

interface PhotoSwipeOptions {
  index?: number;
  history?: boolean;
  focus?: boolean;
  showAnimationDuration?: number;
  hideAnimationDuration?: number;
}

interface PhotoSwipeGallery {
  init(): void;
  listen(event: string, callback: (index: number, item: PhotoSwipeItem) => void): void;
  updateSize(force?: boolean): void;
}

interface PhotoSwipeUIDefault {
  new(element: Element | null, ui: any, items: PhotoSwipeItem[], options: PhotoSwipeOptions): PhotoSwipeGallery;
}

// 扩展Window对象以包含PhotoSwipe
declare global {
  interface Window {
    PhotoSwipe?: PhotoSwipeUIDefault;
    PhotoSwipeUI_Default?: any;
  }
}

export {};

// BSP 卡片布局算法不变量测试（旧版无测试，迁移补课）。
import { describe, it, expect } from 'vitest';
import { generateBspCardLayout, deriveBspGridGeometry, appendPlacements, type BspRect } from './bsp-layout';

// 宿主实际用 deriveBspGridGeometry 推算画布高度：canvasHeight = round(rows × unitH)。
// 1200 宽 / 12 张 → unitH≈451，rows≥7 才放得下全部卡片。
const BASE = { canvasWidth: 1200, canvasHeight: 3200, targetCount: 12, seed: 42 };

function overlaps(a: BspRect, b: BspRect): boolean {
  return a.x < b.x + b.width && a.x + a.width > b.x && a.y < b.y + b.height && a.y + a.height > b.y;
}

describe('generateBspCardLayout', () => {
  it('返回卡片数覆盖 targetCount（fillToFull 可能补洞多放）', () => {
    const cards = generateBspCardLayout(BASE);
    expect(cards.length).toBeGreaterThanOrEqual(12);
  });

  it('全部卡片在画布内', () => {
    const cards = generateBspCardLayout(BASE);
    for (const c of cards) {
      expect(c.x).toBeGreaterThanOrEqual(0);
      expect(c.y).toBeGreaterThanOrEqual(0);
      expect(c.x + c.width).toBeLessThanOrEqual(1200);
      expect(c.y + c.height).toBeLessThanOrEqual(3200);
    }
  });

  it('卡片两两不重叠', () => {
    const cards = generateBspCardLayout(BASE);
    for (let i = 0; i < cards.length; i++) {
      for (let j = i + 1; j < cards.length; j++) {
        expect(overlaps(cards[i], cards[j])).toBe(false);
      }
    }
  });

  it('相同 seed 结果确定', () => {
    const a = generateBspCardLayout(BASE);
    const b = generateBspCardLayout(BASE);
    expect(a).toEqual(b);
  });

  it('不同 seed 布局不同', () => {
    const a = generateBspCardLayout(BASE);
    const b = generateBspCardLayout({ ...BASE, seed: 7 });
    expect(a).not.toEqual(b);
  });

  it('无效参数抛错', () => {
    expect(() => generateBspCardLayout({ ...BASE, minRatio: 0 })).toThrow();
    expect(() => generateBspCardLayout({ ...BASE, maxRatio: 0.5, minRatio: 1 })).toThrow();
    expect(() => generateBspCardLayout({ ...BASE, gap: -1 })).toThrow();
  });

  it('空画布或零卡片返回空数组', () => {
    expect(generateBspCardLayout({ ...BASE, canvasWidth: 0 })).toEqual([]);
    expect(generateBspCardLayout({ ...BASE, targetCount: 0 })).toEqual([]);
  });

  it('gap 生效：卡片尺寸 = 单元尺寸减 gap', () => {
    const cards = generateBspCardLayout({ ...BASE, gap: 16 });
    for (const c of cards) {
      expect(c.width).toBeGreaterThanOrEqual(0);
      expect(c.height).toBeGreaterThanOrEqual(0);
    }
  });
});

describe('deriveBspGridGeometry', () => {
  it('导出的单元尺寸可用于宿主推算画布高度', () => {
    const geometry = deriveBspGridGeometry({ canvasWidth: 1200, canvasHeight: 3600, targetCount: 12 });
    expect(geometry.unitW).toBeGreaterThan(0);
    expect(geometry.unitH).toBeGreaterThan(0);
    expect(geometry.cols).toBeGreaterThan(0);
  });
});

describe('appendPlacements', () => {
  it('扩展高度后追加（宿主模式：newCount 传新总数 + 固定 cols）且已有卡片位置不变', () => {
    // 宿主用 pinnedCols 固定列数：追加时 grid 几何必须与初次 solve 一致
    const geo = deriveBspGridGeometry({ canvasWidth: BASE.canvasWidth, canvasHeight: BASE.canvasHeight, targetCount: 12 });
    const existing = generateBspCardLayout({ ...BASE, cols: geo.cols });
    const newTotal = 18;
    // 宿主模式：用 deriveBspGridGeometry 估算扩展高度再追加
    const appendGeo = deriveBspGridGeometry({
      canvasWidth: BASE.canvasWidth, canvasHeight: 3600, targetCount: newTotal, cols: geo.cols,
    });
    const estimatedRows = Math.max(2, Math.ceil((newTotal * 1.6) / appendGeo.cols));
    const expandedHeight = Math.round(estimatedRows * appendGeo.unitH);

    const extended = appendPlacements({
      existing,
      canvasWidth: BASE.canvasWidth,
      canvasHeight: expandedHeight,
      newCount: newTotal,
      cols: geo.cols,
      seed: 42,
    });
    expect(extended.length).toBeGreaterThanOrEqual(newTotal);
    // 前 N 张卡片位置完全一致（冻结）
    for (let i = 0; i < existing.length; i++) {
      expect(extended[i]).toEqual(existing[i]);
    }
    // 全部不重叠、在扩展画布内
    for (let i = 0; i < extended.length; i++) {
      expect(extended[i].y + extended[i].height).toBeLessThanOrEqual(expandedHeight);
      for (let j = i + 1; j < extended.length; j++) {
        expect(overlaps(extended[i], extended[j])).toBe(false);
      }
    }
  });
});

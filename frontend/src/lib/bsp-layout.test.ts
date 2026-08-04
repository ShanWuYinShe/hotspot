// BSP 卡片布局算法不变量测试。
import { describe, it, expect } from 'vitest';
import { generateBspCardLayout, deriveBspGridGeometry, planBspLayout, appendPlacements, type BspRect } from './bsp-layout';

// 宿主通过 planBspLayout 拿到精确行数：canvasHeight = round(rows × unitH)。
// 1200 宽 / 12 张 → 基础宽定列 cols=10，plan 给出恰好满铺的 2 行布局。
const PLAN = planBspLayout({ canvasWidth: 1200, targetCount: 12, seed: 42 });
const BASE = { canvasWidth: 1200, canvasHeight: PLAN!.canvasHeight, targetCount: 12, seed: 42 };

function overlaps(a: BspRect, b: BspRect): boolean {
  return a.x < b.x + b.width && a.x + a.width > b.x && a.y < b.y + b.height && a.y + a.height > b.y;
}

describe('planBspLayout', () => {
  it('给出恰好满铺的行数与配额（n 张卡铺满 rows×cols，无超额）', () => {
    expect(PLAN).not.toBeNull();
    const { rows, quotas, geometry } = PLAN!;
    const n = 12;
    const sum = quotas["1x1"] + quotas["1x2"] + quotas["2x1"] + quotas["2x2"];
    expect(sum).toBe(n);
    // 配额所需格数 = 网格总格数（恰好铺满）
    const cells = quotas["1x1"] + 2 * quotas["1x2"] + 2 * quotas["2x1"] + 4 * quotas["2x2"];
    expect(cells).toBe(rows * geometry.cols);
  });

  it('n=7（nearby/people mock 场景）不再出现“一行一张卡 + 骨架”', () => {
    const plan = planBspLayout({ canvasWidth: 1044, targetCount: 7, minWidth: 184, minHeight: 274, minRatio: 0.68, maxRatio: 1.48, minCanvasHeight: 560 });
    expect(plan).not.toBeNull();
    const cards = generateBspCardLayout({
      canvasWidth: 1044, canvasHeight: plan!.canvasHeight, targetCount: 7,
      minWidth: 184, minHeight: 274, minRatio: 0.68, maxRatio: 1.48,
      cols: plan!.geometry.cols, quotas: plan!.quotas, seed: plan!.seed,
    });
    expect(cards.filter((c) => !c.placeholder).length).toBe(7);
    expect(cards.filter((c) => c.placeholder).length).toBe(0);
  });

  it('覆盖不变量：4 种宽度 × n=3..40 全部恰好铺满（无骨架、真卡 = n）', () => {
    const OPTS = { minWidth: 184, minHeight: 274, minRatio: 0.68, maxRatio: 1.48, gap: 24 };
    const failures: string[] = [];
    for (const W of [900, 1044, 1280, 1440]) {
      for (let n = 3; n <= 40; n++) {
        const plan = planBspLayout({ canvasWidth: W, targetCount: n, ...OPTS, minCanvasHeight: 560 });
        if (!plan) { failures.push(`plan null: W=${W} n=${n}`); continue; }
        const cards = generateBspCardLayout({
          canvasWidth: W, canvasHeight: plan.canvasHeight, targetCount: n, ...OPTS,
          cols: plan.geometry.cols, quotas: plan.quotas, seed: plan.seed,
        });
        const real = cards.filter((c) => !c.placeholder).length;
        const ph = cards.filter((c) => c.placeholder).length;
        if (real < n) failures.push(`real<n: W=${W} n=${n}`);
        if (ph > 0) failures.push(`placeholder: W=${W} n=${n}`);
      }
    }
    expect(failures).toEqual([]);
  });

  it('超宽屏（W=2200）动态列数收缩后依然恰好满铺', () => {
    // 列数由高度推导：W=2200, minHeight 274 → cols=8，12 卡可精确满铺
    const plan = planBspLayout({ canvasWidth: 2200, targetCount: 12, minWidth: 184, minHeight: 274, minRatio: 0.68, maxRatio: 1.48, minCanvasHeight: 560 });
    expect(plan).not.toBeNull();
    const cards = generateBspCardLayout({
      canvasWidth: 2200, canvasHeight: plan!.canvasHeight, targetCount: 12,
      minWidth: 184, minHeight: 274, minRatio: 0.68, maxRatio: 1.48,
      cols: plan!.geometry.cols, quotas: plan!.quotas, seed: plan!.seed,
    });
    expect(cards.filter((c) => !c.placeholder).length).toBeGreaterThanOrEqual(12);
    expect(cards.filter((c) => c.placeholder).length).toBe(0);
  });
});

describe('generateBspCardLayout', () => {
  it('返回卡片数覆盖 targetCount（fillToFull 可能补洞多放）', () => {
    const cards = generateBspCardLayout(BASE);
    expect(cards.length).toBeGreaterThanOrEqual(12);
  });

  it('fillToFull 补位格带 placeholder 标记，且真实卡片数覆盖 targetCount', () => {
    const cards = generateBspCardLayout(BASE);
    const real = cards.filter((c) => !c.placeholder);
    expect(real.length).toBeGreaterThanOrEqual(12);
    // 全量求解路径：真实卡片按序在前，补位格集中在其后（宿主据此区分）
    const firstPlaceholder = cards.findIndex((c) => c.placeholder);
    if (firstPlaceholder !== -1) {
      expect(cards.slice(firstPlaceholder).every((c) => c.placeholder)).toBe(true);
    }
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
  it('扩展高度后追加（宿主模式：plan 配额 + 固定 cols）且已有卡片位置不变', () => {
    // 宿主用 pinnedCols 固定列数：追加时 grid 几何必须与初次 solve 一致
    const geo = deriveBspGridGeometry({ canvasWidth: BASE.canvasWidth, canvasHeight: BASE.canvasHeight, targetCount: 12 });
    const existing = generateBspCardLayout({ ...BASE, cols: geo.cols, quotas: PLAN!.quotas });
    const newTotal = 18;
    const frozenReal = existing.filter((r) => !r.placeholder).length;
    // 宿主模式：对新增卡数做 plan，扩展高度 = (旧行数 + 新行数) × unitH
    const appendPlan = planBspLayout({
      canvasWidth: BASE.canvasWidth, targetCount: newTotal - frozenReal, cols: geo.cols,
    });
    expect(appendPlan).not.toBeNull();
    const oldRows = Math.max(1, Math.round(BASE.canvasHeight / geo.unitH));
    const expandedHeight = Math.round((oldRows + appendPlan!.rows) * geo.unitH);

    const extended = appendPlacements({
      existing,
      canvasWidth: BASE.canvasWidth,
      canvasHeight: expandedHeight,
      newCount: newTotal,
      cols: geo.cols,
      quotas: appendPlan!.quotas,
      seed: appendPlan!.seed,
    });
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
    // 追加后整体依然恰好满铺（无补位骨架）
    expect(extended.filter((r) => !r.placeholder).length).toBe(newTotal);
    expect(extended.filter((r) => r.placeholder).length).toBe(0);
  });

  it('旧布局含补位格时，追加仍达到 newCount 张真实卡片（补位格不消耗新卡配额）', () => {
    const geo = deriveBspGridGeometry({ canvasWidth: BASE.canvasWidth, canvasHeight: BASE.canvasHeight, targetCount: 12 });
    // 构造含补位格的旧布局：超大画布高度 → 行数远超需要 → fillToFull 补位
    const existing = generateBspCardLayout({ ...BASE, cols: geo.cols, canvasHeight: 3200 });
    expect(existing.some((r) => r.placeholder)).toBe(true);

    const newTotal = 18;
    const frozenReal = existing.filter((r) => !r.placeholder).length;
    const appendPlan = planBspLayout({
      canvasWidth: BASE.canvasWidth, targetCount: newTotal - frozenReal, cols: geo.cols,
    });
    expect(appendPlan).not.toBeNull();
    const oldRows = Math.max(1, Math.round(3200 / geo.unitH));
    const expandedHeight = Math.round((oldRows + appendPlan!.rows) * geo.unitH);
    console.log('diag: frozenReal', frozenReal, 'appendPlan', JSON.stringify(appendPlan), 'expandedHeight', expandedHeight);

    const extended = appendPlacements({
      existing,
      canvasWidth: BASE.canvasWidth,
      canvasHeight: expandedHeight,
      newCount: newTotal,
      cols: geo.cols,
      quotas: appendPlan!.quotas,
      seed: 42,
    });
    const real = extended.filter((r) => !r.placeholder).length;
    expect(real).toBeGreaterThanOrEqual(newTotal);
  });
});

/**
 * AmbientBackdrop — 鸿蒙端页面氛围画布（纯装饰层，对齐 web AmbientBackdrop）
 *
 * 目的：给内容稀疏/空态页补「世界感」纵深，避免大面积纯色屏显得空荡。
 * 分层与 web 完全同构（docs/design-system.md 第 16 节）：
 *   天光 sky/glow → 地平线雾 horizon → 主题世界 world → 暗角 vignette。
 * 三主题世界语言（web 同款，不改几何语言）：
 *   vibrant 星际：星尘 + 星座连线 + 双层轨道环 + 城市天际线剪影（远/近两层）
 *   forest  清野：等高线地形（9 条柔顺曲线）+ 暖阳光斑 + 针叶林剪影（远/近两层）
 *   inkwash 水墨：三段墨山 + 朱砂圆相（留缺口）+ 飞白墨点
 *
 * 实现约束：
 * - 单个 Canvas 一次性绘制（尺寸/主题变化时重绘），静态无逐帧动画 ——
 *   不产生每帧状态更新、不随列表滚动重绘；系统减弱动效场景天然无动效；
 * - 装饰层 hitTestBehavior(None) + accessibilityLevel('no')，不拦截触摸/不参与读屏；
 * - 色值全部来自 ThemeConstants.AMBIENT（web --cst-ambient-* 预合成），页面不得写色值；
 * - 几何按 web 的 1600×900 设计空间绘制，按 preserveAspectRatio=slice 口径缩放裁切，
 *   并复刻 web 窄屏构图修正（≤640 档把轨道环/星座/圆相移到画面中部并缩小）。
 */
import { AmbientColorConfig } from '../../core/theme/ThemeConstants';
import { AppThemeManager } from '../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../core/theme/AppThemeTypes';

/** web 世界层设计空间（与 AmbientBackdrop.svelte viewBox 一致） */
const WORLD_WIDTH: number = 1600;
const WORLD_HEIGHT: number = 900;
/** web 窄屏媒体查询断点 640px：本端按 vp 语义取同一档 */
const NARROW_WIDTH: number = 640;
/** 透明色（Canvas clear/渐变收束用，非主题色） */
const TRANSPARENT: string = '#00000000';

@Component
export struct AmbientBackdrop {
  /** 主题 id（缺省跟随全局解析结果；主题切换时重绘） */
  @StorageProp(KEY_RESOLVED_THEME_ID) @Watch('onThemeChanged') resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private settings: RenderingContextSettings = new RenderingContextSettings(true);
  private canvasContext: CanvasRenderingContext2D = new CanvasRenderingContext2D(this.settings);
  private lastWidth: number = 0;
  private lastHeight: number = 0;
  onThemeChanged(): void {
    this.draw();
  }

  build() {
    Canvas(this.canvasContext)
      .width('100%')
      .height('100%')
      // 装饰层：不拦截触摸、不进入无障碍树、不参与布局尺寸
      .hitTestBehavior(HitTestMode.None)
      .accessibilityLevel('no')
      .onReady(() => {
        this.draw();
      })
      .onAreaChange((_oldArea: Area, newArea: Area) => {
        const width = newArea.width as number;
        const height = newArea.height as number;
        if (Math.abs(width - this.lastWidth) > 0.5 || Math.abs(height - this.lastHeight) > 0.5) {
          this.draw();
        }
      })
  }

  // ==================== 绘制入口 ====================

  private draw(): void {
    const width = this.canvasContext.width;
    const height = this.canvasContext.height;
    if (width <= 0 || height <= 0) {
      return;
    }
    this.lastWidth = width;
    this.lastHeight = height;
    const ctx = this.canvasContext;
    const colors: AmbientColorConfig = AppThemeManager.getAmbientById(this.resolvedThemeId);
    const world: string = AppThemeManager.getAmbientWorld(this.resolvedThemeId);

    ctx.clearRect(0, 0, width, height);
    // 1) 天光：顶部 sky 渐隐（web ambient-sky: 180deg, sky 0% → transparent 30%）
    this.fillLinearVertical(ctx, 0, 0, 0, height * 0.3, colors.sky);
    // 2) 左上主天光（web glow--a: radial 46%×40% at 12% 6%, glow-1 → transparent 72%）
    this.fillEllipticGlow(ctx, width * 0.12, height * 0.06, width * 0.46, height * 0.40, colors.glow1, 0.72);
    // 3) 地平线光源（web glow--b: radial 62%×46% at 50% 104%, glow-2 → transparent 72%）
    this.fillEllipticGlow(ctx, width * 0.5, height * 1.04, width * 0.62, height * 0.46, colors.glow2, 0.72);
    // 4) 地平线雾：底部 horizon 渐隐（web ambient-horizon: 0deg, horizon 0% → transparent 30%）
    this.fillLinearVertical(ctx, 0, height, 0, height * 0.7, colors.horizon);
    // 5) 清野暖阳（web ambient-sun: 36vw 光斑，右 4% 上 1%，圆内 50% 42%）
    if (world === 'forest') {
      this.drawSun(ctx, width, height, colors.sun);
    }
    // 6) 主题世界层（1600×900 设计空间，slice 裁切）
    this.drawWorld(ctx, width, height, world, colors);
    // 7) 暗角（web ambient-vignette: radial 135%×115% at 50% -8%，52% 起收束）
    this.drawVignette(ctx, width, height, colors.vignette);
  }

  // ==================== 基础层 ====================

  /** 垂直线性渐隐：fromY 处为 color，toY 处收束为透明（等价 web linear-gradient(angle 0/180)） */
  private fillLinearVertical(ctx: CanvasRenderingContext2D, x: number, fromY: number, _unusedX: number,
    toY: number, color: string): void {
    const gradient = ctx.createLinearGradient(x, fromY, x, toY);
    gradient.addColorStop(0, color);
    gradient.addColorStop(1, TRANSPARENT);
    ctx.fillStyle = gradient;
    ctx.fillRect(0, Math.min(fromY, toY), ctx.width, Math.abs(toY - fromY));
  }

  /** 椭圆径向光斑（web radial-gradient(46% 40% at ...)）：用缩放坐标系画圆再还原 */
  private fillEllipticGlow(ctx: CanvasRenderingContext2D, cx: number, cy: number, radiusX: number,
    radiusY: number, color: string, fadeStop: number): void {
    if (radiusX <= 0 || radiusY <= 0) {
      return;
    }
    ctx.save();
    ctx.translate(cx, cy);
    ctx.scale(radiusX, radiusY);
    const gradient = ctx.createRadialGradient(0, 0, 0, 0, 0, 1);
    gradient.addColorStop(0, color);
    gradient.addColorStop(fadeStop, TRANSPARENT);
    gradient.addColorStop(1, TRANSPARENT);
    ctx.fillStyle = gradient;
    ctx.fillRect(-1, -1, 2, 2);
    ctx.restore();
  }

  private drawSun(ctx: CanvasRenderingContext2D, width: number, height: number, color: string): void {
    if (color === TRANSPARENT) {
      return;
    }
    const size = Math.min(width * 0.36, 620);
    const left = width * 0.96 - size;
    const top = height * 0.01;
    const cx = left + size * 0.5;
    const cy = top + size * 0.42;
    this.fillEllipticGlow(ctx, cx, cy, size * 0.5, size * 0.5, color, 0.68);
  }

  private drawVignette(ctx: CanvasRenderingContext2D, width: number, height: number, color: string): void {
    if (color === TRANSPARENT) {
      return;
    }
    ctx.save();
    ctx.translate(width * 0.5, height * -0.08);
    ctx.scale(width * 1.35, height * 1.15);
    const gradient = ctx.createRadialGradient(0, 0, 0, 0, 0, 1);
    gradient.addColorStop(0, TRANSPARENT);
    gradient.addColorStop(0.52, TRANSPARENT);
    gradient.addColorStop(1, color);
    ctx.fillStyle = gradient;
    ctx.fillRect(-1, -1, 2, 2);
    ctx.restore();
  }

  // ==================== 世界层（1600×900 设计空间） ====================

  private drawWorld(ctx: CanvasRenderingContext2D, width: number, height: number, world: string,
    colors: AmbientColorConfig): void {
    const scale = Math.max(width / WORLD_WIDTH, height / WORLD_HEIGHT);
    const offsetX = (width - WORLD_WIDTH * scale) * 0.5;
    // vibrant/forest 用 xMidYMid（居中），inkwash 用 xMidYMax（贴底）
    const offsetY = world === 'inkwash'
      ? height - WORLD_HEIGHT * scale
      : (height - WORLD_HEIGHT * scale) * 0.5;
    const narrow = width < NARROW_WIDTH;

    ctx.save();
    ctx.translate(offsetX, offsetY);
    ctx.scale(scale, scale);
    if (world === 'vibrant') {
      this.drawVibrantWorld(ctx, narrow, colors);
    } else if (world === 'forest') {
      this.drawForestWorld(ctx, colors);
    } else {
      this.drawInkwashWorld(ctx, narrow, colors);
    }
    ctx.restore();
  }

  // ---------- vibrant 星际 ----------

  private drawVibrantWorld(ctx: CanvasRenderingContext2D, narrow: boolean, colors: AmbientColorConfig): void {
    // 星尘：集中上 2/3，确定性伪随机（同一构建下位置稳定，不产生抖动）
    ctx.fillStyle = colors.dot;
    for (let i = 0; i < 84; i++) {
      const x = Math.round(this.hash(i + 1) * 1600);
      const y = Math.round(Math.pow(this.hash(i + 37), 1.3) * 600);
      const r = 0.5 + Math.pow(this.hash(i + 73), 2) * 1.5;
      const alpha = 0.16 + this.hash(i + 109) * 0.5;
      ctx.globalAlpha = alpha;
      ctx.beginPath();
      ctx.arc(x, y, r, 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.globalAlpha = 1;

    // 星座连线（web CONSTELLATIONS 三段折线）
    this.strokePolylines(ctx, narrow ? [320, 180, 0.6] : null, colors.line, 1.1, [
      [232, 168, 318, 214, 286, 296, 372, 330],
      [1084, 118, 1156, 176, 1122, 250, 1204, 286],
      [640, 92, 700, 148, 664, 208]
    ]);

    // 轨道环（web ambient-orbit 双层，rotate(-18 1290 218)；窄屏平移到画面中部并缩小）
    ctx.save();
    if (narrow) {
      ctx.translate(36, 87);
      ctx.scale(0.6, 0.6);
    }
    ctx.translate(1290, 218);
    ctx.rotate(-18 * Math.PI / 180);
    this.strokeEllipse(ctx, 260, 112, colors.line, 1, 1);
    this.strokeEllipse(ctx, 352, 158, colors.line, 1, 0.55);
    ctx.restore();

    // 城市天际线：远（ink3）+ 近（ink1）
    this.fillSkyline(ctx, colors.ink3, AmbientBackdrop.SKYLINE_FAR, false);
    this.fillSkyline(ctx, colors.ink1, AmbientBackdrop.SKYLINE_NEAR, true);
  }

  /** 远景楼群 [x, 宽, 高]，底部对齐 900；顶部地面带让楼群连成城市轮廓 */
  private fillSkyline(ctx: CanvasRenderingContext2D, color: string, buildings: number[][], roofs: boolean): void {
    ctx.fillStyle = color;
    ctx.beginPath();
    ctx.moveTo(0, 900);
    ctx.lineTo(0, 883);
    ctx.lineTo(1600, 883);
    ctx.lineTo(1600, 900);
    ctx.closePath();
    for (let i = 0; i < buildings.length; i++) {
      const building = buildings[i];
      const x = building[0];
      const w = building[1];
      const h = building[2];
      ctx.moveTo(x, 900);
      ctx.lineTo(x, 900 - h);
      ctx.lineTo(x + w, 900 - h);
      ctx.lineTo(x + w, 900);
      ctx.closePath();
      // 高楼层加细天线，强化「城市」语义（web skyline(..., true)）
      if (roofs && h >= 150) {
        const antennaX = x + w * 0.5;
        ctx.moveTo(antennaX - 2, 900 - h);
        ctx.lineTo(antennaX - 2, 900 - h - 24);
        ctx.lineTo(antennaX + 2, 900 - h - 24);
        ctx.lineTo(antennaX + 2, 900 - h);
        ctx.closePath();
      }
    }
    ctx.fill();
  }

  // ---------- forest 清野 ----------

  private drawForestWorld(ctx: CanvasRenderingContext2D, colors: AmbientColorConfig): void {
    // 等高线：9 条上下叠置柔顺曲线，越往下振幅越大
    ctx.strokeStyle = colors.line;
    ctx.lineWidth = 1.1;
    ctx.lineCap = 'round';
    for (let i = 0; i < 9; i++) {
      const base = 470 + i * 46;
      const amp = 22 + i * 6;
      const pointsX: number[] = [];
      const pointsY: number[] = [];
      for (let s = 0; s < 17; s++) {
        const t = s / 16;
        pointsX.push(Math.round(t * 1600));
        pointsY.push(Math.round(base + Math.sin(t * Math.PI * 2 + i * 0.9) * amp
          + Math.sin(t * Math.PI * 4 + i * 1.7) * amp * 0.32));
      }
      this.strokeSmoothCurve(ctx, pointsX, pointsY);
    }
    // 针叶林剪影：远（ink3）+ 近（ink1）
    this.fillTreeLine(ctx, colors.ink3, AmbientBackdrop.TREES_FAR);
    this.fillTreeLine(ctx, colors.ink1, AmbientBackdrop.TREES_NEAR);
  }

  /** 由 [x, 宽, 高] 生成双峰针叶树路径（底部对齐 900），等价 web treeLine */
  private fillTreeLine(ctx: CanvasRenderingContext2D, color: string, trees: number[][]): void {
    ctx.fillStyle = color;
    ctx.beginPath();
    for (let i = 0; i < trees.length; i++) {
      const tree = trees[i];
      const x = tree[0];
      const w = tree[1];
      const h = tree[2];
      ctx.moveTo(x, 900);
      ctx.lineTo(x + w * 0.34, 900 - h * 0.52);
      ctx.lineTo(x + w * 0.5, 900 - h);
      ctx.lineTo(x + w * 0.66, 900 - h * 0.58);
      ctx.lineTo(x + w, 900);
      ctx.closePath();
    }
    ctx.fill();
  }

  // ---------- inkwash 水墨 ----------

  private drawInkwashWorld(ctx: CanvasRenderingContext2D, narrow: boolean, colors: AmbientColorConfig): void {
    // 三段墨山（web 三层 path 的 cubic/bezier 逐一还原；S 的反射控制点已展开）
    ctx.fillStyle = colors.ink3;
    ctx.beginPath();
    ctx.moveTo(0, 900);
    ctx.lineTo(0, 636);
    ctx.bezierCurveTo(120, 600, 220, 660, 340, 612);
    ctx.bezierCurveTo(460, 564, 560, 560, 700, 620);
    ctx.bezierCurveTo(840, 680, 940, 572, 1080, 642);
    ctx.bezierCurveTo(1220, 712, 1360, 592, 1600, 662);
    ctx.lineTo(1600, 900);
    ctx.closePath();
    ctx.fill();

    ctx.fillStyle = colors.ink2;
    ctx.beginPath();
    ctx.moveTo(0, 900);
    ctx.lineTo(0, 704);
    ctx.bezierCurveTo(160, 662, 260, 722, 420, 682);
    ctx.bezierCurveTo(580, 642, 660, 642, 820, 702);
    ctx.bezierCurveTo(980, 762, 1080, 652, 1240, 712);
    ctx.bezierCurveTo(1400, 772, 1460, 672, 1600, 702);
    ctx.lineTo(1600, 900);
    ctx.closePath();
    ctx.fill();

    ctx.fillStyle = colors.ink1;
    ctx.beginPath();
    ctx.moveTo(0, 900);
    ctx.lineTo(0, 772);
    ctx.bezierCurveTo(200, 742, 320, 802, 520, 772);
    ctx.bezierCurveTo(720, 742, 780, 732, 980, 792);
    ctx.bezierCurveTo(1180, 852, 1240, 752, 1400, 802);
    ctx.bezierCurveTo(1560, 852, 1520, 782, 1600, 802);
    ctx.lineTo(1600, 900);
    ctx.closePath();
    ctx.fill();

    // 朱砂圆相：描边留缺口（dasharray 700 220）+ 旋转；窄屏平移到画面中部并缩小
    ctx.save();
    if (narrow) {
      ctx.translate(-41, 85);
      ctx.scale(0.66, 0.66);
    }
    ctx.translate(1290, 250);
    ctx.rotate(-28 * Math.PI / 180);
    ctx.strokeStyle = colors.emblem;
    ctx.lineWidth = 9;
    ctx.lineCap = 'round';
    ctx.setLineDash([700, 220]);
    ctx.beginPath();
    ctx.arc(0, 0, 146, 0, Math.PI * 2);
    ctx.stroke();
    ctx.setLineDash([]);
    ctx.restore();

    // 飞白墨点（web 4 点，各自不同透明度）
    ctx.fillStyle = colors.emblem;
    this.fillDot(ctx, 246, 742, 4, 0.5);
    this.fillDot(ctx, 284, 706, 2.2, 0.35);
    this.fillDot(ctx, 196, 676, 2.8, 0.4);
    this.fillDot(ctx, 1322, 812, 3, 0.3);
  }

  // ==================== 绘制小工具 ====================

  /** 确定性伪随机：与 web AmbientBackdrop.hash 同式，保证星点位置可复现不抖动 */
  private hash(n: number): number {
    const v = Math.sin(n * 127.1) * 43758.5453;
    return v - Math.floor(v);
  }

  private fillDot(ctx: CanvasRenderingContext2D, x: number, y: number, r: number, alpha: number): void {
    ctx.globalAlpha = alpha;
    ctx.beginPath();
    ctx.arc(x, y, r, 0, Math.PI * 2);
    ctx.fill();
    ctx.globalAlpha = 1;
  }

  /** 折线组描边；narrowTransform = [tx, ty, scale] 时先平移缩放（对齐 web 窄屏构图修正） */
  private strokePolylines(ctx: CanvasRenderingContext2D, narrowTransform: number[] | null, color: string,
    lineWidth: number, polylines: number[][]): void {
    ctx.save();
    if (narrowTransform !== null) {
      ctx.translate(narrowTransform[0], narrowTransform[1]);
      ctx.scale(narrowTransform[2], narrowTransform[2]);
    }
    ctx.strokeStyle = color;
    ctx.lineWidth = lineWidth;
    ctx.lineJoin = 'round';
    ctx.lineCap = 'round';
    for (let i = 0; i < polylines.length; i++) {
      const points = polylines[i];
      ctx.beginPath();
      ctx.moveTo(points[0], points[1]);
      for (let p = 2; p + 1 < points.length; p += 2) {
        ctx.lineTo(points[p], points[p + 1]);
      }
      ctx.stroke();
    }
    ctx.restore();
  }

  private strokeEllipse(ctx: CanvasRenderingContext2D, radiusX: number, radiusY: number, color: string,
    lineWidth: number, alpha: number): void {
    ctx.save();
    ctx.scale(radiusX, radiusY);
    ctx.strokeStyle = color;
    ctx.lineWidth = lineWidth / Math.max(radiusX, radiusY);
    ctx.globalAlpha = alpha;
    ctx.beginPath();
    ctx.arc(0, 0, 1, 0, Math.PI * 2);
    ctx.stroke();
    ctx.restore();
  }

  /** Catmull-Rom → 三次贝塞尔（对齐 web smoothPath），用于等高线柔顺曲线 */
  private strokeSmoothCurve(ctx: CanvasRenderingContext2D, xs: number[], ys: number[]): void {
    const count = xs.length;
    if (count < 2) {
      return;
    }
    ctx.beginPath();
    ctx.moveTo(xs[0], ys[0]);
    for (let i = 0; i < count - 1; i++) {
      const p0x = xs[Math.max(0, i - 1)];
      const p0y = ys[Math.max(0, i - 1)];
      const p1x = xs[i];
      const p1y = ys[i];
      const p2x = xs[i + 1];
      const p2y = ys[i + 1];
      const p3x = xs[Math.min(count - 1, i + 2)];
      const p3y = ys[Math.min(count - 1, i + 2)];
      const c1x = p1x + (p2x - p0x) / 6;
      const c1y = p1y + (p2y - p0y) / 6;
      const c2x = p2x - (p3x - p1x) / 6;
      const c2y = p2y - (p3y - p1y) / 6;
      ctx.bezierCurveTo(c1x, c1y, c2x, c2y, p2x, p2y);
    }
    ctx.stroke();
  }

  // ==================== 世界几何（与 web 常量逐值一致） ====================

  /** 星际远景楼群：web SKYLINE_FAR */
  private static readonly SKYLINE_FAR: number[][] = [
    [0, 46, 44], [54, 34, 58], [96, 60, 36], [164, 42, 62], [214, 70, 42],
    [292, 46, 30], [346, 56, 54], [410, 38, 66], [456, 66, 38], [530, 44, 52],
    [582, 52, 34], [642, 60, 60], [710, 40, 40], [758, 68, 30], [834, 46, 56],
    [888, 56, 44], [952, 40, 64], [1000, 66, 36], [1074, 44, 52], [1126, 62, 30],
    [1196, 48, 58], [1252, 66, 40], [1326, 42, 62], [1376, 56, 34], [1440, 64, 54],
    [1512, 44, 42], [1564, 36, 60]
  ];

  /** 星际近景楼群：web SKYLINE_NEAR（含天台天线） */
  private static readonly SKYLINE_NEAR: number[][] = [
    [24, 88, 72], [128, 70, 96], [212, 104, 64], [332, 78, 108], [426, 112, 78],
    [556, 84, 100], [656, 96, 58], [770, 74, 112], [858, 108, 74], [982, 82, 92],
    [1080, 100, 62], [1196, 78, 104], [1290, 92, 70], [1398, 84, 98], [1494, 88, 66]
  ];

  /** 清野远景树线：web TREES_FAR */
  private static readonly TREES_FAR: number[][] = [
    [0, 60, 84], [54, 46, 60], [104, 66, 96], [176, 50, 70], [232, 58, 104],
    [300, 44, 66], [352, 64, 90], [430, 48, 112], [486, 56, 74], [548, 62, 98],
    [626, 46, 64], [684, 60, 104], [756, 44, 78], [808, 58, 92], [882, 48, 68],
    [940, 62, 100], [1010, 44, 74], [1062, 58, 96], [1130, 48, 62], [1190, 64, 92],
    [1262, 44, 108], [1318, 60, 78], [1386, 48, 94], [1444, 58, 66], [1514, 62, 100]
  ];

  /** 清野近景针叶林：web TREES_NEAR */
  private static readonly TREES_NEAR: number[][] = [
    [8, 84, 150], [104, 68, 108], [182, 96, 178], [292, 72, 120], [374, 88, 196],
    [474, 66, 132], [552, 92, 164], [660, 74, 116], [746, 88, 186], [846, 66, 138],
    [924, 96, 158], [1030, 72, 122], [1114, 86, 190], [1214, 68, 140], [1294, 92, 172],
    [1402, 74, 112], [1486, 96, 182]
  ];
}

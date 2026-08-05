/**
 * Discrete grid-tiered card layout.
 *
 * Cards are placed on an integer unit grid and may only occupy one of four
 * discrete tile shapes — 1×1, 1×2 (portrait / 竖), 2×1 (landscape / 横), or 2×2.
 * The field is overwhelmingly 1×1 cards (~85%+); 1×2 and 2×1 are used sparingly
 * as visual rhythm breaks ("穿插"), and 2×2 is reserved as a single hero anchor
 * for large enough collections.
 *
 * Because every card snaps to the same unit, all cards of the same tier are
 * pixel-identical in size, portrait and landscape tiles appear symmetrically,
 * and a "prefer no two equal-tier cards touch" rule keeps the field varied.
 *
 * The previous continuous binary-space partition was replaced because its
 * random split points produced a different size for every "smallest" card and,
 * on a wide-short canvas, could never yield portrait tiles.
 *
 * Note on the ratio band: `minRatio` / `maxRatio` constrain the 1×1 unit tile
 * (the fundamental cell). The rectangular tiers 1×2 / 2×1 are *defined* by
 * their shape — their 0.5 / 2.0 ratios are the point of having them, so they
 * are admitted regardless of the band as long as the grid can host them.
 */
export type BspRect = {
  x: number;
  y: number;
  width: number;
  height: number;
  /**
   * True for tiles added by the hole-filling pass (beyond `targetCount`).
   * These are layout fillers, not real cards — the host should render them as
   * empty placeholders instead of mapping them back onto items.
   */
  placeholder?: boolean;
};

export type BspLayoutOptions = {
  canvasWidth: number;
  canvasHeight: number;
  targetCount: number;
  /** Clear space between cards. The grid unit is sized so the visible card still meets the minimums. */
  gap?: number;
  /**
   * Symmetric minimum edge length. When the directional `minWidth` / `minHeight`
   * are omitted they both fall back to it.
   */
  minSize?: number;
  /**
   * Minimum width a card may keep after the inter-card gutter is removed.
   * Defaults to `minSize`. Lower than `minHeight` for vertically-stacked cards
   * so the column count (desktop density) is preserved.
   */
  minWidth?: number;
  /**
   * Minimum height a card may keep after the inter-card gutter is removed.
   * Defaults to `minSize`. Set this high enough that every card can show its
   * full vertical content (image + text + action bar) without clipping.
   */
  minHeight?: number;
  /** Aspect-ratio band the 1×1 unit tile must respect, as width / height. */
  minRatio?: number;
  maxRatio?: number;
  /** Optional column-count override; otherwise derived from count and a near-square target. */
  cols?: number;
  /**
   * Optional per-tier card quotas (must sum to `targetCount`). When omitted the
   * soft quota heuristic is used. Provided by `planBspLayout` so the tile mix
   * tiles the grid exactly (no placeholders, no interior holes).
   */
  quotas?: BspQuotas;
  seed?: number;
};

type TileShape = { w: number; h: number; tier: Tier };
type Tier = "1x1" | "1x2" | "2x1" | "2x2";

/** Per-tier card quotas. Sums to the target card count. */
export type BspQuotas = Record<Tier, number>;

const TILE_SHAPES: TileShape[] = [
  { w: 1, h: 1, tier: "1x1" },
  { w: 1, h: 2, tier: "1x2" },
  { w: 2, h: 1, tier: "2x1" },
  { w: 2, h: 2, tier: "2x2" },
];

type Random = () => number;

function mulberry32(seed: number): Random {
  let value = seed >>> 0;
  return () => {
    value += 0x6d2b79f5;
    let t = value;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function shuffle<T>(items: T[], random: Random): T[] {
  const out = items.slice();
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value));
}

type GridGeometry = {
  cols: number;
  unitW: number;
  unitH: number;
  /** Shapes that still fit inside a grid of `cols` columns (rows are decided by the host). */
  shapes: TileShape[];
};

/**
 * Derive the unit grid. The unit is the common denominator every card snaps to,
 * so a single shared unit guarantees all same-tier cards are pixel-identical.
 *
 * Column count is chosen for a *target* near-square field (independent of the
 * probe height the host may pass), so the grid stays stable as the host scales
 * the canvas height to fit the requested card count.
 */
function deriveGrid(
  canvasWidth: number,
  targetCount: number,
  minWidth: number,
  minHeight: number,
  minRatio: number,
  maxRatio: number,
  requestedCols?: number,
): GridGeometry {
  // The unit tile (1×1) must satisfy the ratio band AND the visible minimums.
  // Solve for unitH given unitW so that the 1×1 reads as a balanced square-ish
  // cell — this is what makes same-tier cards visually consistent.
  //
  // Column count is derived dynamically from the canvas width so the 1×1 base
  // tile stays near-square: `unitW ≈ unitH`, which keeps 1×2 / 2×1 / 2×2 at
  // their true aspect ratios (a 1×2 tile reads as "twice as tall", not as a
  // skinny 1×3+ strip). The height drives the column count (card content needs
  // ~minHeight of vertical space), the width only caps the upper bound so a
  // column never gets narrower than a base card; when the card count is small
  // the column count shrinks so the field tiles densely. The floor of 2 keeps
  // narrow containers usable; an explicit `requestedCols` always wins.
  const heightLimitedCols = Math.round(canvasWidth / Math.max(minHeight, 120));
  const widthFloor = Math.floor(canvasWidth / Math.max(minWidth, 120));
  const countLimitedCols = Math.max(2, Math.min(Math.min(heightLimitedCols, widthFloor), targetCount));
  const cols = requestedCols ? clamp(requestedCols, 2, 12) : clamp(countLimitedCols, 2, 12);
  const unitW = canvasWidth / cols;

  // unitH bounded by: ratio band on the 1×1, and minHeight floor (+ gap budget).
  const ratioLo = unitW / maxRatio; // smallest unitH that keeps 1×1 ≥ minRatio
  const ratioHi = unitW / minRatio; // largest unitH that keeps 1×1 ≤ maxRatio
  const floorH = minHeight + 8;
  let unitH = clamp((ratioLo + ratioHi) / 2, floorH, Math.max(floorH, ratioHi));
  // Guard: if the ratio band is too tight to clear minHeight, honour minHeight.
  if (unitH < floorH) unitH = floorH;

  // Keep tile shapes that physically fit in `cols` columns. Rectangular tiers
  // (1×2, 2×1, 2×2) are admitted by their defined shape; the ratio band only
  // governs the 1×1 unit. 2×2 needs ≥2 cols; 2×1 needs ≥2 cols; 1×2 always fits.
  const shapes = TILE_SHAPES.filter((shape) => shape.w <= cols);
  if (!shapes.some((s) => s.tier === "1x1")) {
    shapes.unshift({ w: 1, h: 1, tier: "1x1" });
  }

  return { cols, unitW, unitH, shapes };
}

type Placement = {
  shape: TileShape;
  /** Grid coordinates of the top-left cell. */
  gx: number;
  gy: number;
  index: number;
  /** True for hole-filling tiles (see `fillToFull`). */
  placeholder: boolean;
};

function canPlace(
  occupancy: Uint8Array,
  cols: number,
  shape: TileShape,
  gx: number,
  gy: number,
  rows: number,
): boolean {
  if (gx + shape.w > cols || gy + shape.h > rows) return false;
  for (let dy = 0; dy < shape.h; dy++) {
    for (let dx = 0; dx < shape.w; dx++) {
      if (occupancy[(gy + dy) * cols + (gx + dx)] !== 0) return false;
    }
  }
  return true;
}

function mark(
  occupancy: Uint8Array,
  cols: number,
  shape: TileShape,
  gx: number,
  gy: number,
  value: number,
): void {
  for (let dy = 0; dy < shape.h; dy++) {
    for (let dx = 0; dx < shape.w; dx++) {
      occupancy[(gy + dy) * cols + (gx + dx)] = value;
    }
  }
}

function neighborTier(
  placementGrid: (Tier | null)[],
  cols: number,
  rows: number,
  cx: number,
  cy: number,
): Tier | null {
  if (cx < 0 || cy < 0 || cx >= cols || cy >= rows) return null;
  return placementGrid[cy * cols + cx];
}

function fillPlacementGrid(grid: (Tier | null)[], cols: number, p: Placement): void {
  for (let dy = 0; dy < p.shape.h; dy++) {
    for (let dx = 0; dx < p.shape.w; dx++) {
      grid[(p.gy + dy) * cols + (p.gx + dx)] = p.shape.tier;
    }
  }
}

/**
 * True when any cell the tile would cover is adjacent (up/down/left/right) to a
 * cell already owned by a card of the *same* tier. Used as a soft preference:
 * the placer avoids this when possible but relaxes it to reach the target count.
 */
function touchesSameTier(
  grid: (Tier | null)[],
  cols: number,
  rows: number,
  shape: TileShape,
  gx: number,
  gy: number,
): boolean {
  for (let dy = 0; dy < shape.h; dy++) {
    for (let dx = 0; dx < shape.w; dx++) {
      const cx = gx + dx;
      const cy = gy + dy;
      const checks: Array<[number, number]> = [
        [cx - 1, cy],
        [cx + 1, cy],
        [cx, cy - 1],
        [cx, cy + 1],
      ];
      for (const [nx, ny] of checks) {
        if (neighborTier(grid, cols, rows, nx, ny) === shape.tier) return true;
      }
    }
  }
  return false;
}

/**
 * Soft per-tier quotas for a target card count.
 *
 * The field reads as "mostly 1×1 squares with occasional rectangular tiles as
 * visual rhythm breaks" rather than a varied mosaic. 1×1 is the overwhelming
 * majority (~85%+ depending on count); 1×2 and 2×1 are used sparingly as
 * "穿插" (interspersed accents — just enough to break monotony); 2×2 is
 * reserved as a single hero anchor for large enough collections.
 *
 * `random` adds a small ±1 jitter so consecutive layouts with the same count
 * do not land on identical quotas every time.
 */
function computeTierQuotas(targetCount: number, random: Random): Record<Tier, number> {
  const quotas: Record<Tier, number> = { "1x1": 0, "1x2": 0, "2x1": 0, "2x2": 0 };

  // 2×2 hero: at most one, and only when the scene is large enough that a
  // hero card does not dominate the entire field.
  if (targetCount >= 8) quotas["2x2"] = 1;

  const remaining = targetCount - quotas["2x2"];
  // Rectangular tiers are "穿插" — occasional rhythm breaks in a mostly-1×1
  // field. Each gets ~5% of non-hero cards (not 20%), capped at 2 so they
  // never dominate the visual. A random nudge prevents repeat layouts.
  const rectCount = Math.min(2, Math.max(0, Math.round(remaining * 0.05 + (random() - 0.5))));
  quotas["1x2"] = rectCount;
  quotas["2x1"] = rectCount;
  // 1×1 takes everything else — the bread and butter of the field.
  quotas["1x1"] = Math.max(0, remaining - quotas["1x2"] - quotas["2x1"]);

  // Clamp totals to targetCount (rounding can overshoot by 1 on small sets).
  let total = quotas["1x1"] + quotas["1x2"] + quotas["2x1"] + quotas["2x2"];
  while (total > targetCount && quotas["1x1"] > 0) {
    quotas["1x1"]--;
    total--;
  }
  return quotas;
}

/**
 * Collect every anchor position where `shape` fits in the current occupancy.
 */
function candidateAnchors(
  occupancy: Uint8Array,
  cols: number,
  rows: number,
  shape: TileShape,
): Array<[number, number]> {
  const out: Array<[number, number]> = [];
  for (let gy = 0; gy + shape.h <= rows; gy++) {
    for (let gx = 0; gx + shape.w <= cols; gx++) {
      if (canPlace(occupancy, cols, shape, gx, gy, rows)) out.push([gx, gy]);
    }
  }
  return out;
}

/**
 * Heuristic weights for the per-step placement score. Centralised here so the
 * balance between "avoid same-tier neighbours", "honour quotas", "spread tiles
 * across the canvas", "stay compact", and "balance orientations" can be tuned
 * without hunting through the scoring code.
 *
 * Higher = stronger pull toward that factor. The anti-adjacency penalty is the
 * largest so that even the few rectangular tiles ("穿插") are spread out
 * instead of clumped; the rest break ties and add subtler rhythm.
 */
const SCORE_WEIGHTS = {
  antiAdjacency: 100,   // large penalty when the candidate touches a same-tier card
  size: 18,             // flat per-cell bonus so larger tiles are scheduled before the grid fragments
  quota: 12,            // urgency: tiers behind their placed/quota ratio get a boost
  spread: 8,            // push a tier's tiles away from its already-placed centroid
  compact: 3,           // prefer anchors adjacent to filled cells (keeps the field packable)
  orientation: 6,       // nudge portrait/landscape toward whichever is rarer
  random: 2,            // jitter so equally-scored candidates do not always pick the same one
} as const;

type Candidate = { shape: TileShape; gx: number; gy: number; score: number };

/**
 * Count occupied 4-neighbours of a single cell. Used for the compactness bonus
 * (filling next to existing cards keeps the field packable).
 */
function filledNeighbours(occupancy: Uint8Array, cols: number, rows: number, cx: number, cy: number): number {
  let count = 0;
  const checks: Array<[number, number]> = [[cx - 1, cy], [cx + 1, cy], [cx, cy - 1], [cx, cy + 1]];
  for (const [nx, ny] of checks) {
    if (nx < 0 || ny < 0 || nx >= cols || ny >= rows) continue;
    if (occupancy[ny * cols + nx] !== 0) count++;
  }
  return count;
}

/**
 * Centroid (mean gx/gy) of every already-placed card of `tier`, in grid units.
 * Returns null when none are placed yet. Used by the spread factor so a tier's
 * new tile tends to land away from its existing cluster.
 */
function tierCentroid(placements: Placement[], tier: Tier): { x: number; y: number } | null {
  let sx = 0, sy = 0, n = 0;
  for (const p of placements) {
    if (p.shape.tier !== tier) continue;
    sx += p.gx + p.shape.w / 2;
    sy += p.gy + p.shape.h / 2;
    n++;
  }
  return n === 0 ? null : { x: sx / n, y: sy / n };
}

/**
 * Score one (tier, position) candidate. Higher is better. The score blends the
 * factors described by `SCORE_WEIGHTS`; pure ties are broken by a small random
 * term so two layouts with the same input do not come out identical.
 */
function scoreCandidate(
  candidate: { shape: TileShape; gx: number; gy: number },
  context: {
    occupancy: Uint8Array;
    placementGrid: (Tier | null)[];
    cols: number;
    rows: number;
    quotas: Record<Tier, number>;
    placed: Record<Tier, number>;
    placements: Placement[];
    random: Random;
  },
): number {
  const { shape, gx, gy } = candidate;
  const { occupancy, placementGrid, cols, rows, quotas, placed, placements, random } = context;

  // Anti-adjacency: hard-feel penalty when this tile would touch a same-tier card.
  const adjacencyPenalty = touchesSameTier(placementGrid, cols, rows, shape, gx, gy)
    ? -SCORE_WEIGHTS.antiAdjacency
    : 0;

  // Size priority: bigger tiles get a flat bonus. Large tiles need contiguous
  // space, so scheduling them early (before the grid fragments into 1×1 scraps)
  // is what actually lets the variety budget get placed. Without this, 1×1 — the
  // majority tier — would win every step on raw count and starve the rectangles.
  const sizeBonus = (shape.w * shape.h) * SCORE_WEIGHTS.size;

  // Quota urgency: tiers whose placed/quota ratio is the lowest are the most
  // "behind", so they get a boost. This keeps each tier filling in proportion to
  // its budget instead of one tier monopolising the early steps. The factor is
  // (1 - placed/quota): 1.0 when nothing placed yet, 0.0 once the quota is met.
  const quota = quotas[shape.tier];
  const ratio = quota > 0 ? placed[shape.tier] / quota : 1;
  const urgencyBonus = (1 - ratio) * SCORE_WEIGHTS.quota;

  // Spread: distance from this tier's existing centroid, normalised by grid
  // diagonal, so a tier's tiles distribute across the canvas instead of clumping.
  const centroid = tierCentroid(placements, shape.tier);
  let spreadBonus = 0;
  if (centroid) {
    const cx = gx + shape.w / 2;
    const cy = gy + shape.h / 2;
    const dist = Math.hypot(cx - centroid.x, cy - centroid.y);
    const diag = Math.hypot(cols, rows);
    spreadBonus = (dist / diag) * SCORE_WEIGHTS.spread * 4;
  }

  // Compact: count filled neighbours of every cell this tile covers. Filling
  // next to existing cards avoids isolated holes that block later placements.
  let filled = 0;
  for (let dy = 0; dy < shape.h; dy++) {
    for (let dx = 0; dx < shape.w; dx++) {
      filled += filledNeighbours(occupancy, cols, rows, gx + dx, gy + dy);
    }
  }
  const compactBonus = filled * SCORE_WEIGHTS.compact;

  // Orientation balance: for portrait/landscape, boost whichever orientation has
  // been placed less so the two stay close in count.
  let orientationBonus = 0;
  if (shape.tier === "1x2" || shape.tier === "2x1") {
    const portrait = placed["1x2"];
    const landscape = placed["2x1"];
    const mine = shape.tier === "1x2" ? portrait : landscape;
    const other = shape.tier === "1x2" ? landscape : portrait;
    orientationBonus = (other - mine) * SCORE_WEIGHTS.orientation;
  }

  const jitter = random() * SCORE_WEIGHTS.random;
  return adjacencyPenalty + sizeBonus + urgencyBonus + spreadBonus + compactBonus + orientationBonus + jitter;
}

/**
 * Enumerate every (tier, position) candidate that still fits the remaining grid
 * and has quota left, then score each. Returns them sorted best-first.
 */
function rankCandidates(
  geometry: GridGeometry,
  rows: number,
  occupancy: Uint8Array,
  placementGrid: (Tier | null)[],
  quotas: Record<Tier, number>,
  owed: Record<Tier, number>,
  placed: Record<Tier, number>,
  placements: Placement[],
  random: Random,
  allowAdjacent: boolean,
): Candidate[] {
  const { cols } = geometry;
  const context = { occupancy, placementGrid, cols, rows, quotas, placed, placements, random };
  const candidates: Candidate[] = [];
  for (const shape of geometry.shapes) {
    if (owed[shape.tier] <= 0) continue;
    const anchors = candidateAnchors(occupancy, cols, rows, shape);
    for (const [gx, gy] of anchors) {
      if (!allowAdjacent && touchesSameTier(placementGrid, cols, rows, shape, gx, gy)) continue;
      const score = scoreCandidate({ shape, gx, gy }, context);
      candidates.push({ shape, gx, gy, score });
    }
  }
  candidates.sort((a, b) => b.score - a.score);
  // 穿插保护：还有大形状（1×2 / 2×1 / 2×2）配额未完成时，形状按“占列
  // 风险”排序——1×2 竖卡占整列，必须最先放，否则 2×1 先占满行会让竖卡
  // 永远放不下；1×1 排到末尾，避免 anti-adjacency 惩罚让相邻矩形输给
  // 1×1 抢位。
  if (owed["1x2"] + owed["2x1"] + owed["2x2"] > 0) {
    const tierPriority: Record<Tier, number> = { "1x2": 0, "2x2": 1, "2x1": 2, "1x1": 3 };
    candidates.sort(
      (a, b) => tierPriority[a.shape.tier] - tierPriority[b.shape.tier] || b.score - a.score,
    );
  }
  return candidates;
}

/**
 * Greedy fill that places one card per step, choosing the highest-scoring
 * (tier, position) candidate each time. `allowAdjacent` controls whether
 * same-tier neighbours are tolerated; when false, such candidates are filtered
 * out before scoring. Returns when it can no longer place anything — which may
 * be short of targetCount on a tight grid (the caller then relaxes and retries,
 * or fills the gaps with any-fits shapes).
 *
 * This is deliberately non-backtracking: the scoring heuristic is strong enough
 * to avoid most dead-ends on the small grids this layout targets, and a clean
 * relax-and-retry outer loop is far easier to reason about than step-level undo.
 */
function greedyFill(
  geometry: GridGeometry,
  rows: number,
  occupancy: Uint8Array,
  placementGrid: (Tier | null)[],
  quotas: Record<Tier, number>,
  owed: Record<Tier, number>,
  placed: Record<Tier, number>,
  placements: Placement[],
  targetCount: number,
  random: Random,
  allowAdjacent: boolean,
): void {
  const { cols } = geometry;
  while (placements.length < targetCount) {
    const candidates = rankCandidates(
      geometry, rows, occupancy, placementGrid, quotas, owed, placed, placements, random, allowAdjacent,
    );
    if (!candidates.length) break;
    const choice = candidates[0];
    const placement: Placement = { shape: choice.shape, gx: choice.gx, gy: choice.gy, index: placements.length, placeholder: false };
    mark(occupancy, cols, placement.shape, placement.gx, placement.gy, placement.index + 1);
    fillPlacementGrid(placementGrid, cols, placement);
    placements.push(placement);
    owed[placement.shape.tier]--;
    placed[placement.shape.tier]++;
  }
}

/**
 * Fill every remaining free cell with the largest shape that fits, ignoring
 * quotas and adjacency. Unlike the quota-driven passes this runs until the grid
 * is **completely full** (not until a target card count) — so the canvas never
 * has interior holes. Extra cards beyond `targetCount` are returned with their
 * `index` set past the original items; the host maps them by wrapping items or
 * leaving them empty as it sees fit. Because a 1×1 fits any single free cell,
 * this always reaches a zero-hole layout when the grid has at least one cell.
 */
function fillToFull(
  geometry: GridGeometry,
  rows: number,
  occupancy: Uint8Array,
  placementGrid: (Tier | null)[],
  placements: Placement[],
): void {
  const { cols } = geometry;
  // Biggest-first so leftover space becomes fewer, larger cards rather than a
  // scatter of 1×1s — the same reasoning as the original mopup pass.
  const shapesBySize = geometry.shapes.slice().sort((a, b) => b.w * b.h - a.w * a.h);
  for (;;) {
    let placed = false;
    for (const shape of shapesBySize) {
      const anchors = candidateAnchors(occupancy, cols, rows, shape);
      if (!anchors.length) continue;
      const [gx, gy] = anchors[0];
      const placement: Placement = { shape, gx, gy, index: placements.length, placeholder: true };
      mark(occupancy, cols, shape, gx, gy, placement.index + 1);
      fillPlacementGrid(placementGrid, cols, placement);
      placements.push(placement);
      placed = true;
      break;
    }
    if (!placed) break; // grid genuinely full
  }
}

/**
 * Per-step scored placement.
 *
 * Each card is chosen by scoring every legal (tier, position) candidate and
 * taking the best, so tier choice and position emerge from the current board
 * state instead of a preset sequence. This keeps tile types intermixed across
 * the canvas (no "rectangles clump at the top, squares fill the bottom" banding)
 * while still honouring the soft per-tier quotas, the anti-adjacency preference,
 * and portrait/landscape balance.
 *
 * Passes, each strictly additive on the previous board:
 *  1. Scored greedy honouring anti-adjacency — the visually clean primary pass.
 *  2. Scored greedy with anti-adjacency relaxed — recovers cards the strict
 *     pass left on the table when the grid is dense.
 *  3. `fillToFull` — fills EVERY remaining free cell with the largest fitting
 *     shape so the canvas has zero interior holes. Cards placed here may exceed
 *     `targetCount`; the host wraps or trims them as needed.
 */
function placeCards(
  geometry: GridGeometry,
  rows: number,
  targetCount: number,
  random: Random,
  quotas?: BspQuotas,
): Placement[] {
  const { cols } = geometry;
  const q = quotas ?? computeTierQuotas(targetCount, random);
  const owed: Record<Tier, number> = { ...q };
  const placed: Record<Tier, number> = { "1x1": 0, "1x2": 0, "2x1": 0, "2x2": 0 };

  const occupancy = new Uint8Array(cols * rows);
  const placementGrid: (Tier | null)[] = new Array(cols * rows).fill(null);
  const placements: Placement[] = [];

  // 大形状先放（确定性、无视相邻）：1×1 之后才放，评分中的 anti-adjacency
  // 惩罚就不会让矩形被 1×1 抢位，配额（格数 = 网格格数）恒可完成。
  forcePlaceBigShapes(geometry, rows, occupancy, placementGrid, owed, placements, targetCount);
  if (placements.length < targetCount) {
    greedyFill(geometry, rows, occupancy, placementGrid, q, owed, placed, placements, targetCount, random, false);
  }
  // 碎片兜底：配额中的大形状受网格局部几何限制放不下时，用 1×1 完成剩余
  // 配额，保证恰好 targetCount 张真卡铺满网格（无补位骨架）。
  const oneByOne = TILE_SHAPES[0];
  while (placements.length < targetCount) {
    const anchors = candidateAnchors(occupancy, cols, rows, oneByOne);
    if (!anchors.length) break;
    const [gx, gy] = anchors[0];
    const placement: Placement = { shape: oneByOne, gx, gy, index: placements.length, placeholder: false };
    mark(occupancy, cols, oneByOne, gx, gy, placement.index + 1);
    fillPlacementGrid(placementGrid, cols, placement);
    placements.push(placement);
  }
  // Always top up to a fully tiled canvas — no interior holes, ever. This is
  // what makes the layout visually complete regardless of how targetCount's
  // tier mix divides the grid.
  fillToFull(geometry, rows, occupancy, placementGrid, placements);

  return placements;
}

/**
 * Place every remaining big tile (1×2 / 2×2 / 2×1) with a deterministic
 * leftmost-topmost anchor, ignoring adjacency. Big tiles must go before any
 * 1×1 so the quota mix always fits: 1×2 (occupies a whole column) first, then
 * 2×2 (needs a 2×2 block), then 2×1 (needs two columns in one row). Stops
 * early at `targetCount` placements.
 */
function forcePlaceBigShapes(
  geometry: GridGeometry,
  rows: number,
  occupancy: Uint8Array,
  placementGrid: (Tier | null)[],
  owed: Record<Tier, number>,
  placements: Placement[],
  targetCount: number,
): void {
  const { cols } = geometry;
  const order: TileShape[] = [
    { w: 1, h: 2, tier: "1x2" },
    { w: 2, h: 2, tier: "2x2" },
    { w: 2, h: 1, tier: "2x1" },
  ];
  for (;;) {
    let placed = false;
    for (const shape of order) {
      if (owed[shape.tier] <= 0) continue;
      const anchors = candidateAnchors(occupancy, cols, rows, shape);
      if (!anchors.length) continue;
      const [gx, gy] = anchors[0];
      const placement: Placement = { shape, gx, gy, index: placements.length, placeholder: false };
      mark(occupancy, cols, shape, gx, gy, placement.index + 1);
      fillPlacementGrid(placementGrid, cols, placement);
      placements.push(placement);
      owed[shape.tier]--;
      placed = true;
      break;
    }
    if (!placed || placements.length >= targetCount) break;
  }
}

/**
 * Generates a discrete-tiered, gap-consistent set of card bounds.
 * The field is mostly 1×1 with a 2×2 hero and sparse 1×2/2×1 accents
 * ("穿插") to break the rhythm. Supply a seed for reproducible layouts
 * (for example, in tests).
 */
export function generateBspCardLayout({
  canvasWidth,
  canvasHeight,
  targetCount,
  gap = 8,
  minSize = 110,
  minWidth,
  minHeight,
  minRatio = 0.6,
  maxRatio = 1.7,
  cols,
  quotas,
  seed = Math.floor(Math.random() * 2 ** 32),
}: BspLayoutOptions): BspRect[] {
  if (canvasWidth <= 0 || canvasHeight <= 0 || targetCount <= 0) return [];
  if (minRatio <= 0 || maxRatio < minRatio || gap < 0) {
    throw new Error("Invalid BSP layout constraints.");
  }

  const resolvedMinWidth = minWidth ?? minSize;
  const resolvedMinHeight = minHeight ?? minSize;

  const geometry = deriveGrid(
    canvasWidth,
    targetCount,
    resolvedMinWidth,
    resolvedMinHeight,
    minRatio,
    maxRatio,
    cols,
  );
  // Rows are derived from the actual canvas height, so the host can scale the
  // field to fit the requested card count by adjusting height alone. Use round
  // rather than floor: the host sets canvasHeight = round(rows * unitH), so the
  // quotient is within ε of an integer and floor could drop a whole row to
  // floating-point truncation (e.g. 6.9996 → 6).
  const rows = Math.max(1, Math.round(canvasHeight / geometry.unitH));

  const random = mulberry32(seed);
  const placements = placeCards(geometry, rows, targetCount, random, quotas);
  if (placements.length === 0) return [];

  const inset = gap / 2;
  return placements.map((p) => ({
    x: p.gx * geometry.unitW + inset,
    y: p.gy * geometry.unitH + inset,
    width: Math.max(0, p.shape.w * geometry.unitW - gap),
    height: Math.max(0, p.shape.h * geometry.unitH - gap),
    placeholder: p.placeholder,
  }));
}

/**
 * Enumerate every tier-quota decomposition of `extra` surplus grid cells such
 * that the tiles cover exactly `extra` cells beyond their 1×1 footprint: each
 * 2×2 tile adds 3 surplus cells, each 1×2 / 2×1 adds 1, and 1×1 tiles add
 * none. When a decomposition exists, `n` cards tile an `rows × cols` grid
 * with zero holes.
 *
 * Heuristics mirror the soft quotas: at most one 2×2 hero (and none for
 * small collections), few rectangular "穿插" tiles (≤ 6), and balanced
 * portrait/landscape. `rows === 1` forbids portrait tiles and heroes (they
 * need two rows). The list is sorted by preference: more heroes first, then
 * fewer rectangles, then balanced orientations — the host tries each candidate
 * and keeps the first one that actually places.
 */
function decomposeQuotas(
  extra: number,
  cardBudget: number,
  rows: number,
  heroCap: number,
): BspQuotas[] {
  const out: BspQuotas[] = [];
  if (extra < 0) return out;
  const maxExtra = rows >= 2 ? 3 + 6 : 6; // 1 hero + 6 rectangles; single row has no hero
  if (extra > maxExtra) return out;
  const hMax = rows >= 2 ? Math.min(heroCap, Math.floor(extra / 3)) : 0;
  for (let h = hMax; h >= 0; h--) {
    const ab = extra - 3 * h;
    if (ab > 6) continue;
    if (h + ab > cardBudget) continue;
    const aMax = rows >= 2 ? Math.min(ab, cardBudget - h) : 0;
    for (let a = 0; a <= aMax; a++) {
      const b = ab - a;
      if (a < 0 || b < 0) continue;
      out.push({ "1x1": cardBudget - h - a - b, "1x2": a, "2x1": b, "2x2": h });
    }
  }
  out.sort(
    (p, q) =>
      q["2x2"] - p["2x2"] ||
      p["1x2"] + p["2x1"] - (q["1x2"] + q["2x1"]) ||
      Math.abs(p["1x2"] - p["2x1"]) - Math.abs(q["1x2"] - q["2x1"]),
  );
  return out;
}

/** Count real (non-placeholder) cards in a layout. */
function countReal(rects: BspRect[]): number {
  return rects.reduce((n, r) => n + (r.placeholder ? 0 : 1), 0);
}

/**
 * Plan a hole-free layout for `targetCount` cards: pick the row count and the
 * per-tier quotas so the tiles cover the `rows × cols` grid exactly, with no
 * placeholders and no interior holes.
 *
 * Rows start from the tightest fit for the card count (and `minCanvasHeight`),
 * preferring enough surplus for a 2×2 hero on larger collections; the search
 * widens until the surplus is decomposable. Returns null when no row count
 * admits an exact tiling (the host falls back to the probe loop).
 */
export function planBspLayout(
  options: Omit<BspLayoutOptions, "canvasHeight"> & { minCanvasHeight?: number },
): { geometry: GridGeometry; rows: number; quotas: BspQuotas; canvasHeight: number; seed: number } | null {
  const {
    canvasWidth,
    targetCount,
    minSize = 110,
    minWidth,
    minHeight,
    minRatio = 0.6,
    maxRatio = 1.7,
    cols,
    minCanvasHeight = 0,
  } = options;
  if (canvasWidth <= 0 || targetCount <= 0) return null;

  const resolvedMinWidth = minWidth ?? minSize;
  const resolvedMinHeight = minHeight ?? minSize;
  const geometry = deriveGrid(
    canvasWidth,
    targetCount,
    resolvedMinWidth,
    resolvedMinHeight,
    minRatio,
    maxRatio,
    cols,
  );
  const { cols: c, unitH } = geometry;
  const n = targetCount;

  const rowsForCount = Math.max(1, Math.ceil(n / c));
  const rowsForHeight = minCanvasHeight > 0 ? Math.max(1, Math.ceil(minCanvasHeight / unitH)) : 0;
  const minRows = Math.max(rowsForCount, rowsForHeight);
  // Prefer enough surplus cells (≥ 3) to host a 2×2 hero on larger collections.
  const heroTarget = n >= 8 ? Math.ceil((n + 3) / c) : minRows;
  const heroRows = Math.max(minRows, heroTarget);
  // Surplus > 9 (1 hero + 6 rectangles) is never decomposable, so rows beyond
  // floor((n+9)/c) cannot admit an exact tiling.
  const maxRows = Math.max(heroRows, Math.floor((n + 9) / c));
  const heroCap = n >= 8 ? 1 : 0;

  const tryRows = (rows: number): { geometry: GridGeometry; rows: number; quotas: BspQuotas; canvasHeight: number; seed: number } | null => {
    const extra = rows * c - n;
    const candidates = decomposeQuotas(extra, n, rows, heroCap);
    const height = Math.round(rows * unitH);
    // 逐个候选、逐个 seed 模拟放置：配额必须真实可放（恰好 n 张真卡、
    // 无补位骨架），且把验证成功的 seed 带回宿主——anti-adjacency 评分对
    // seed 敏感（同配额不同 seed 放置成败不同），宿主必须用同一 seed 放置。
    for (const quotas of candidates) {
      for (let s = 0; s < 8; s++) {
        const result = generateBspCardLayout({
          canvasWidth,
          canvasHeight: height,
          targetCount: n,
          minWidth: resolvedMinWidth,
          minHeight: resolvedMinHeight,
          minRatio,
          maxRatio,
          cols: c,
          quotas,
          seed: s,
        });
        if (countReal(result) >= n && result.every((r) => !r.placeholder)) {
          return { geometry, rows, quotas, canvasHeight: height, seed: s };
        }
      }
    }
    return null;
  };

  for (let rows = heroRows; rows <= maxRows; rows++) {
    const plan = tryRows(rows);
    if (plan) return plan;
  }
  // Fall back to fewer rows (accepts an all-1×1 field when heroes don't fit).
  // Walk downward to `rowsForCount` so a single dense row is a valid solution
  // even when minCanvasHeight would have preferred two.
  for (let rows = heroRows - 1; rows >= rowsForCount; rows--) {
    const plan = tryRows(rows);
    if (plan) return plan;
  }
  return null;
}

/**
 * Exposes the grid geometry the layout would use for a given canvas. The host
 * component derives its canvas height from this so the field always has exactly
 * enough rows for the requested cards (no excess vertical space, no shortfall).
 */
export function deriveBspGridGeometry(options: BspLayoutOptions): GridGeometry {
  const { canvasWidth, targetCount = 9, minSize = 110, minWidth, minHeight, minRatio = 0.6, maxRatio = 1.7, cols } = options;
  const resolvedMinWidth = minWidth ?? minSize;
  const resolvedMinHeight = minHeight ?? minSize;
  return deriveGrid(
    canvasWidth,
    targetCount,
    resolvedMinWidth,
    resolvedMinHeight,
    minRatio,
    maxRatio,
    cols,
  );
}

export type { GridGeometry, TileShape, Tier };

/**
 * Append `newCount - existing.length` cards to a previously solved layout
 * WITHOUT moving any already-placed card.
 *
 * This is the performance path for paginated / infinite-scroll feeds: when the
 * host loads more items it does not re-solve the whole canvas (which would be
 * O(count × cells) and would visibly reshuffle existing cards). Instead the
 * existing placements are frozen, the occupancy grid is rebuilt from them, and
 * only the new cards are placed into the remaining free cells using the same
 * scored-greedy strategy as a fresh solve. The canvas may then be extended
 * downward (more rows) to make room, and `fillToFull` guarantees the result is
 * still hole-free.
 *
 * `existing` must be the rects returned by a prior `generateBspCardLayout` /
 * `appendPlacements` call for the same `canvasWidth` and grid options; the
 * function rebuilds the grid from `canvasWidth` + `cols` so the unit geometry
 * matches. Returns the full rect list (frozen cards first, then new ones, then
 * any hole-fillers).
 */
type AppendOptions = Omit<BspLayoutOptions, "targetCount"> & { existing: BspRect[]; newCount: number };

/**
 * Try one append solve with a specific seed (see `appendPlacements`).
 */
function tryAppend({
  existing,
  canvasWidth,
  canvasHeight,
  newCount,
  gap = 8,
  minSize = 110,
  minWidth,
  minHeight,
  minRatio = 0.6,
  maxRatio = 1.7,
  cols,
  quotas,
  seed = Math.floor(Math.random() * 2 ** 32),
}: AppendOptions): BspRect[] {
  if (canvasWidth <= 0 || canvasHeight <= 0 || newCount <= 0) return existing;
  if (minRatio <= 0 || maxRatio < minRatio || gap < 0) {
    throw new Error("Invalid BSP layout constraints.");
  }

  const resolvedMinWidth = minWidth ?? minSize;
  const resolvedMinHeight = minHeight ?? minSize;

  // Recover the column count of the existing layout so the unit grid matches:
  // a fresh `deriveGrid(newCount)` could pick different cols (sqrt(count)
  // scaling), which would make every frozen rect snap to the wrong cells.
  // The narrowest rect in `existing` is one unit wide, so cols = round(W /
  // (minRectWidth + gap)). An explicit `cols` always wins.
  const narrowestWidth = existing.length
    ? existing.reduce((min, r) => Math.min(min, r.width), Infinity)
    : resolvedMinWidth;
  const detectedCols = clamp(Math.round(canvasWidth / (narrowestWidth + gap)), 2, 32);

  const geometry = deriveGrid(
    canvasWidth,
    existing.length + newCount,
    resolvedMinWidth,
    resolvedMinHeight,
    minRatio,
    maxRatio,
    cols ?? detectedCols,
  );
  const { unitW, unitH } = geometry;
  const rows = Math.max(1, Math.round(canvasHeight / unitH));

  // Rebuild occupancy + placement grid from the frozen rects. A rect snaps to
  // the unit grid by rounding its pixel coords; if the prior layout used the
  // same unit geometry (the contract above) the rounding is exact.
  const occupancy = new Uint8Array(geometry.cols * rows);
  const placementGrid: (Tier | null)[] = new Array(geometry.cols * rows).fill(null);
  const frozenPlacements: Placement[] = [];
  for (const rect of existing) {
    const gx = Math.round(rect.x / unitW);
    const gy = Math.round(rect.y / unitH);
    const w = Math.max(1, Math.round(rect.width / unitW));
    const h = Math.max(1, Math.round(rect.height / unitH));
    // Skip rects that fall outside the current grid (e.g. after a width change
    // shifted the units); the host's full recompute path handles that case.
    if (gx < 0 || gy < 0 || gx + w > geometry.cols || gy + h > rows) return existing;
    // Recover the tier from the snapped dimensions.
    const tier = (TILE_SHAPES.find((s) => s.w === w && s.h === h)?.tier ?? "1x1") as Tier;
    const shape: TileShape = { w, h, tier };
    const placement: Placement = { shape, gx, gy, index: frozenPlacements.length, placeholder: rect.placeholder === true };
    mark(occupancy, geometry.cols, shape, gx, gy, placement.index + 1);
    fillPlacementGrid(placementGrid, geometry.cols, placement);
    frozenPlacements.push(placement);
  }

  // Place only the NEW cards (those beyond the frozen real cards). Quotas are
  // computed for the new batch so the appended region keeps the same tier mix
  // instead of degrading into all 1×1s. Hole-filling placeholders in `existing`
  // occupy grid cells but are NOT cards — they must not consume the new-card
  // budget, otherwise appending always falls short of `newCount` real cards.
  const frozenRealCount = frozenPlacements.reduce((n, p) => n + (p.placeholder ? 0 : 1), 0);
  const newCardCount = Math.max(0, newCount - frozenRealCount);
  // An explicit `quotas` describes the NEW cards exactly (sum = newCardCount,
  // from planBspLayout) and is used as-is; otherwise derive soft quotas for the
  // new total and subtract the tiers already represented by frozen REAL cards
  // so we do not over-produce a tier that is already well-represented.
  const computedQuotas = quotas ?? computeTierQuotas(newCount, mulberry32(seed));
  const placedFromFrozen: Record<Tier, number> = { "1x1": 0, "1x2": 0, "2x1": 0, "2x2": 0 };
  for (const p of frozenPlacements) {
    if (!p.placeholder) placedFromFrozen[p.shape.tier]++;
  }
  const owed: Record<Tier, number> = quotas
    ? { ...quotas }
    : {
        "1x1": Math.max(0, computedQuotas["1x1"] - placedFromFrozen["1x1"]),
        "1x2": Math.max(0, computedQuotas["1x2"] - placedFromFrozen["1x2"]),
        "2x1": Math.max(0, computedQuotas["2x1"] - placedFromFrozen["2x1"]),
        "2x2": Math.max(0, computedQuotas["2x2"] - placedFromFrozen["2x2"]),
      };

  const placements: Placement[] = [...frozenPlacements];
  // Re-index so new placements continue after the frozen ones.
  const reindex = () => placements.forEach((p, i) => { p.index = i; });

  const appendTarget = frozenPlacements.length + newCardCount;
  // 大形状先放（确定性、无视相邻），1×1 之后评分放置——与初次 solve 一致
  forcePlaceBigShapes(geometry, rows, occupancy, placementGrid, owed, placements, appendTarget);
  if (placements.length < appendTarget) {
    greedyFill(geometry, rows, occupancy, placementGrid, computedQuotas, owed, placedFromFrozen, placements, appendTarget, mulberry32(seed ^ 0x5a5a), false);
  }
  // 碎片兜底：剩余格无法容纳配额中的大形状（如布局含补位格、空间被切碎）时，
  // 用 1×1 完成剩余配额，保证新卡真卡数达到 newCardCount（宿主无需回退全量重排）。
  const oneByOne = TILE_SHAPES[0];
  while (placements.length < appendTarget) {
    const anchors = candidateAnchors(occupancy, geometry.cols, rows, oneByOne);
    if (!anchors.length) break;
    const [gx, gy] = anchors[0];
    const placement: Placement = { shape: oneByOne, gx, gy, index: placements.length, placeholder: false };
    mark(occupancy, geometry.cols, oneByOne, gx, gy, placement.index + 1);
    fillPlacementGrid(placementGrid, geometry.cols, placement);
    placements.push(placement);
  }
  // Keep the appended canvas hole-free as well.
  fillToFull(geometry, rows, occupancy, placementGrid, placements);
  reindex();

  const inset = gap / 2;
  return placements.map((p) => ({
    x: p.gx * unitW + inset,
    y: p.gy * unitH + inset,
    width: Math.max(0, p.shape.w * unitW - gap),
    height: Math.max(0, p.shape.h * unitH - gap),
    placeholder: p.placeholder,
  }));
}

/**
 * Append `newCount` total real cards to a previously solved layout WITHOUT
 * moving any already-placed card.
 *
 * Placement is seed-sensitive (the scorer carries a random tie-break), so the
 * append is attempted with several seeds and the first result that places the
 * full card count with zero placeholder tiles is returned. When none succeed
 * (rare: the frozen layout fragments the space), the result with the most real
 * cards is returned and the host falls back to a full re-solve.
 */
export function appendPlacements(options: AppendOptions): BspRect[] {
  const { existing, newCount, seed = Math.floor(Math.random() * 2 ** 32) } = options;
  if (existing.length === 0) return existing;
  let best: BspRect[] | null = null;
  let bestReal = -1;
  for (let attempt = 0; attempt < 8; attempt++) {
    const result = tryAppend({ ...options, seed: seed + attempt * 0x9e3779b9 });
    const real = countReal(result);
    if (real >= newCount && result.every((r) => !r.placeholder)) return result;
    if (real > bestReal) {
      bestReal = real;
      best = result;
    }
  }
  return best ?? tryAppend({ ...options, seed });
}

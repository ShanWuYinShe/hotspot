/**
 * Discrete grid-tiered card layout.
 *
 * Cards are placed on an integer unit grid and may only occupy one of four
 * discrete tile shapes — 1×1, 1×2 (portrait / 竖), 2×1 (landscape / 横), or 2×2.
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
};

export type BspLayoutOptions = {
  canvasWidth: number;
  canvasHeight: number;
  targetCount: number;
  /** Clear space between cards. The grid unit is sized so the visible card still meets the minimums. */
  gap?: number;
  /**
   * Symmetric minimum edge length. Kept for backwards compatibility; when the
   * directional `minWidth` / `minHeight` are omitted they both fall back to it.
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
  seed?: number;
};

type TileShape = { w: number; h: number; tier: Tier };
type Tier = "1x1" | "1x2" | "2x1" | "2x2";

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
  // Column count scales with the card count so the grid stays roughly square
  // instead of becoming a single very tall column: a 1000-card feed at cols=3
  // would be 333 rows, which both looks bad and makes the scorer O(rows×cols)
  // per step explode. `sqrt(count)` keeps rows≈cols, but it is clamped by the
  // minimum card width so a very wide feed never produces columns thinner than
  // a card can legibly fill. The floor of 3 preserves the small-grid rhythm;
  // an explicit `requestedCols` always wins.
  const sqrtCols = Math.round(Math.sqrt(Math.max(targetCount, 9)));
  const widthLimitedCols = Math.floor(canvasWidth / Math.max(minWidth, 120));
  const scaledCols = clamp(sqrtCols, 3, Math.max(3, widthLimitedCols));
  const cols = requestedCols ? clamp(requestedCols, 2, 12) : scaledCols;
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
 * The field reads as "mostly uniform squares with rectangular tiles as rhythm"
 * rather than a maximally varied mosaic: 1×1 is the steady majority (~55%),
 * 1×2 and 2×1 each take ~20% and stay balanced against each other, and 2×2 is
 * reserved as a single visual anchor (only when the collection is large enough
 * that one hero card does not dominate it).
 *
 * `random` adds a small ±1 jitter so consecutive layouts with the same count
 * do not land on identical quotas every time.
 */
function computeTierQuotas(targetCount: number, random: Random): Record<Tier, number> {
  const quotas: Record<Tier, number> = { "1x1": 0, "1x2": 0, "2x1": 0, "2x2": 0 };

  // 2×2 anchor: at most one, and only once the collection can absorb it.
  if (targetCount >= 6) quotas["2x2"] = 1;

  const remaining = targetCount - quotas["2x2"];
  // Rectangular tiles each target ~20% of the non-hero cards, rounded with a
  // random nudge so two layouts of the same size are not identical.
  const rectTarget = Math.max(1, Math.round(remaining * 0.2 + (random() - 0.5)));
  quotas["1x2"] = rectTarget;
  quotas["2x1"] = rectTarget;
  // 1×1 takes whatever is left; never negative.
  quotas["1x1"] = Math.max(0, remaining - quotas["1x2"] - quotas["2x1"]);

  // Clamp totals to targetCount (rect rounding can overshoot by 1 on small sets).
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
 * largest because visual repetition ("two identical tiles touching") is the most
 * obvious defect; the rest break ties and add subtler rhythm.
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
    const placement: Placement = { shape: choice.shape, gx: choice.gx, gy: choice.gy, index: placements.length };
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
      const placement: Placement = { shape, gx, gy, index: placements.length };
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
): Placement[] {
  const { cols } = geometry;
  const quotas = computeTierQuotas(targetCount, random);
  const owed: Record<Tier, number> = { ...quotas };
  const placed: Record<Tier, number> = { "1x1": 0, "1x2": 0, "2x1": 0, "2x2": 0 };

  const occupancy = new Uint8Array(cols * rows);
  const placementGrid: (Tier | null)[] = new Array(cols * rows).fill(null);
  const placements: Placement[] = [];

  greedyFill(geometry, rows, occupancy, placementGrid, quotas, owed, placed, placements, targetCount, random, false);
  if (placements.length < targetCount) {
    greedyFill(geometry, rows, occupancy, placementGrid, quotas, owed, placed, placements, targetCount, random, true);
  }
  // Always top up to a fully tiled canvas — no interior holes, ever. This is
  // what makes the layout visually complete regardless of how targetCount's
  // tier mix divides the grid.
  fillToFull(geometry, rows, occupancy, placementGrid, placements);

  return placements;
}

/**
 * Generates a discrete-tiered, gap-consistent set of card bounds.
 * Every same-tier card is pixel-identical; portrait and landscape tiles are
 * balanced; equal-tier adjacency is avoided when the grid allows it. Supply a
 * seed for reproducible layouts (for example, in tests).
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
  const placements = placeCards(geometry, rows, targetCount, random);
  if (placements.length === 0) return [];

  const inset = gap / 2;
  return placements.map((p) => ({
    x: p.gx * geometry.unitW + inset,
    y: p.gy * geometry.unitH + inset,
    width: Math.max(0, p.shape.w * geometry.unitW - gap),
    height: Math.max(0, p.shape.h * geometry.unitH - gap),
  }));
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
export function appendPlacements({
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
  seed = Math.floor(Math.random() * 2 ** 32),
}: Omit<BspLayoutOptions, "targetCount"> & { existing: BspRect[]; newCount: number }): BspRect[] {
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
    const placement: Placement = { shape, gx, gy, index: frozenPlacements.length };
    mark(occupancy, geometry.cols, shape, gx, gy, placement.index + 1);
    fillPlacementGrid(placementGrid, geometry.cols, placement);
    frozenPlacements.push(placement);
  }

  // Place only the NEW cards (those beyond the frozen count). Quotas are
  // computed for the new batch so the appended region keeps the same tier mix
  // instead of degrading into all 1×1s.
  const newCardCount = Math.max(0, newCount - frozenPlacements.length);
  const quotas = computeTierQuotas(newCardCount + frozenPlacements.length, mulberry32(seed));
  // Subtract the tiers already represented by frozen cards so we do not
  // over-produce a tier that is already well-represented.
  const placedFromFrozen: Record<Tier, number> = { "1x1": 0, "1x2": 0, "2x1": 0, "2x2": 0 };
  for (const p of frozenPlacements) placedFromFrozen[p.shape.tier]++;
  const owed: Record<Tier, number> = {
    "1x1": Math.max(0, quotas["1x1"] - placedFromFrozen["1x1"]),
    "1x2": Math.max(0, quotas["1x2"] - placedFromFrozen["1x2"]),
    "2x1": Math.max(0, quotas["2x1"] - placedFromFrozen["2x1"]),
    "2x2": Math.max(0, quotas["2x2"] - placedFromFrozen["2x2"]),
  };

  const placements: Placement[] = [...frozenPlacements];
  // Re-index so new placements continue after the frozen ones.
  const reindex = () => placements.forEach((p, i) => { p.index = i; });

  greedyFill(geometry, rows, occupancy, placementGrid, quotas, owed, placedFromFrozen, placements, frozenPlacements.length + newCardCount, mulberry32(seed ^ 0xa5a5), false);
  if (placements.length < frozenPlacements.length + newCardCount) {
    greedyFill(geometry, rows, occupancy, placementGrid, quotas, owed, placedFromFrozen, placements, frozenPlacements.length + newCardCount, mulberry32(seed ^ 0x5a5a), true);
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
  }));
}

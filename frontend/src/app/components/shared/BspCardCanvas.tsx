import React, { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { generateBspCardLayout, deriveBspGridGeometry, appendPlacements, type BspRect } from "../../lib/bsp-layout";

type BspCardCanvasProps<T> = {
  items: T[];
  renderItem: (item: T, index: number) => React.ReactNode;
  className?: string;
  /** Clear space between cards. 24px protects the shared card hover lift and scale. */
  gap?: number;
  /**
   * Minimum visible edge length after the inter-card gutter has been removed.
   * It prevents the solver from filling a single card-sized region with tiny leaves.
   * Acts as the symmetric fallback for both axes; `minCardWidth` / `minCardHeight`
   * override it per axis when card content has a directional minimum.
   */
  minCardSize?: number;
  /**
   * Minimum visible card width after the gutter is removed. Defaults to
   * `minCardSize`. Lower than `minCardHeight` for vertically-stacked cards so
   * the column count (desktop density) is preserved.
   */
  minCardWidth?: number;
  /**
   * Minimum visible card height after the gutter is removed. Defaults to
   * `minCardSize`. Set this so the tallest card still fits its full vertical
   * content (image + text + action bar) without the action bar being clipped.
   */
  minCardHeight?: number;
  minRatio?: number;
  maxRatio?: number;
  minCanvasHeight?: number;
  ariaLabel?: string;
  /**
   * When provided, a sentinel at the bottom of the canvas fires `onLoadMore`
   * once it scrolls near the viewport — wire this to a paginated fetch or a
   * local "show more" state. `hasMore` must be false once the source is
   * exhausted or the sentinel keeps firing on every scroll.
   */
  onLoadMore?: () => void;
  hasMore?: boolean;
};

/**
 * Responsive BSP renderer shared by the three desktop discovery directories.
 * Cards are mostly 1×1 for easy scanning, with sparse 1×2 / 2×1 accents
 * ("穿插") and an occasional 2×2 hero to break the rhythm without clutter.
 */
export function BspCardCanvas<T>({
  items,
  renderItem,
  className = "",
  gap = 24,
  minCardSize = 160,
  minCardWidth,
  minCardHeight,
  minRatio = 0.68,
  maxRatio = 1.48,
  minCanvasHeight = 460,
  ariaLabel = "Card layout",
  onLoadMore,
  hasMore = false,
}: BspCardCanvasProps<T>) {
  // Directional minimums collapse to the symmetric fallback. When a caller
  // passes only `minCardSize` we keep the legacy symmetric behavior; when it
  // passes the axis-specific props those win for their axis. The width bound
  // protects desktop column density; the height bound protects a card's full
  // vertical content (so its action bar is never clipped).
  const resolvedMinWidth = minCardWidth ?? minCardSize;
  const resolvedMinHeight = minCardHeight ?? minCardSize;
  const containerRef = useRef<HTMLElement | null>(null);
  const [width, setWidth] = useState(0);
  const [seed] = useState(() => Math.floor(Math.random() * 2 ** 32));

  useLayoutEffect(() => {
    const element = containerRef.current;
    if (!element) return;
    const update = () => setWidth(Math.round(element.getBoundingClientRect().width));
    update();
    const observer = new ResizeObserver(update);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  // Canvas height is now driven by the placement grid, not by a fixed ratio.
  // The grid needs enough rows to host `targetCount` cards plus room for the
  // larger tiers (hero / portrait / landscape tiles span 2 cells on an axis).
  // We probe a few candidate row counts and keep the smallest one that the
  // solver can fill to the target, falling back to the legacy floor.
  //
  // The solver now tiles the WHOLE canvas (zero interior holes), so a solved
  // layout may contain more slots than `items.length` — the extra slots are
  // filled by wrapping items (see the render map below). Success is therefore
  // "every item has a slot" (`result.length >= items.length`), not an exact
  // match. We pick the smallest row count that achieves this so the canvas
  // stays tight.
  // Cached previous solve so an items-grows event can APPEND rather than
  // re-solve the whole canvas (which would be O(count × cells) and would
  // visibly reshuffle existing cards). Reset whenever the width changes,
  // because the unit grid shifts and old rects no longer snap cleanly.
  const prevSolveRef = useRef<{ width: number; count: number; height: number; layout: BspRect[] } | null>(null);
  /**
   * Pinned column count from the first successful layout solve. Once the grid
   * geometry is established for a given viewport width, the column count (and
   * thus the card base size) stays frozen — loading more data extends the
   * canvas downward rather than shrinking every card to fit more columns.
   * Reset when the viewport width changes (responsive recalculation).
   */
  const pinnedColsRef = useRef<number | null>(null);

  const { canvasHeight, layout } = useMemo(() => {
    if (!width || !items.length) {
      prevSolveRef.current = null;
      pinnedColsRef.current = null;
      return { canvasHeight: minCanvasHeight, layout: [] as BspRect[] };
    }

    // `generateBspCardLayout` measures the source leaves before its gap inset,
    // so add the gutter back to preserve the promised visible minimum on each axis.
    const sourceMinWidth = resolvedMinWidth + gap;
    const sourceMinHeight = resolvedMinHeight + gap;

    const prev = prevSolveRef.current;
    // When the viewport width changes the grid geometry must be recalculated;
    // clear the pinned cols so the next solve picks the appropriate column
    // count for the new width.
    if (pinnedColsRef.current !== null && prev?.width !== width) {
      pinnedColsRef.current = null;
    }

    // Append path: same width, items only grew. Freeze the existing cards and
    // place just the new ones into the remaining free cells, then extend the
    // canvas downward by enough rows to fit them. This keeps a 1000-card feed
    // cheap: each page load solves only the new page's worth of cards, never
    // the accumulated history.
    if (prev && prev.width === width && items.length > prev.count) {
      // Estimate how tall the canvas must grow to host the new items: average
      // ~1.6 cells per card across the tier mix, so total cells ≈ count × 1.6,
      // and rows ≈ cells / cols. Pass this expanded height so the append solve
      // actually has free cells to drop the new cards into (passing the old
      // height would find a full grid and add nothing).
      const appendGeo = deriveBspGridGeometry({
        canvasWidth: width, canvasHeight: width * 3, targetCount: items.length,
        gap, minWidth: sourceMinWidth, minHeight: sourceMinHeight, minRatio, maxRatio,
        cols: pinnedColsRef.current ?? undefined,
      });
      const estimatedRows = Math.max(2, Math.ceil((items.length * 1.6) / appendGeo.cols));
      const expandedHeight = Math.max(prev.height, Math.round(estimatedRows * appendGeo.unitH));

      const appended = appendPlacements({
        existing: prev.layout,
        canvasWidth: width,
        canvasHeight: expandedHeight,
        newCount: items.length,
        gap,
        minWidth: sourceMinWidth,
        minHeight: sourceMinHeight,
        minRatio,
        maxRatio,
        cols: pinnedColsRef.current ?? undefined,
        seed,
      });
      if (appended.length >= items.length) {
        const maxY = appended.reduce((m, r) => Math.max(m, r.y + r.height), 0);
        const newHeight = Math.max(expandedHeight, Math.ceil(maxY / appendGeo.unitH) * appendGeo.unitH);
        prevSolveRef.current = { width, count: items.length, height: newHeight, layout: appended };
        return { canvasHeight: newHeight, layout: appended };
      }
      // Append could not cover (rare — grid geometry mismatch); fall through to
      // a full recompute so the canvas always reaches the target count.
    }

    // Probe candidate grids by row count. Start from the theoretical minimum
    // (cols × rows ≥ count) and grow until a tier-rich layout is achievable.
    // Pass pinned cols when available so the card base size stays stable
    // across data loads for the same viewport width.
    const geometry = deriveBspGridGeometry({
      canvasWidth: width,
      // Pass a generous height so deriveGrid's row clamp does not pre-trim;
      // we override rows ourselves below.
      canvasHeight: width * 3,
      targetCount: items.length,
      gap,
      minWidth: sourceMinWidth,
      minHeight: sourceMinHeight,
      minRatio,
      maxRatio,
      cols: pinnedColsRef.current ?? undefined,
    });
    const { cols, unitH } = geometry;

    // Pin the column count from the first solve so card sizes never change
    // when items are loaded incrementally at the same viewport width.
    if (pinnedColsRef.current === null && cols > 0) {
      pinnedColsRef.current = cols;
    }

    // The probe budget is count-sensitive. Small collections benefit from
    // trying many seeds × a few row counts to find a varied, hole-free tile
    // mix; large collections (the paginated/infinite case) would make that
    // O(seeds × rows × cells) explode, and variety matters less when the eye
    // can only see a screenful at a time. So: rich search below the threshold,
    // a single deterministic solve above it.
    const richSearchThreshold = 24;
    const rich = items.length <= richSearchThreshold;
    // Average cells a card occupies across the tier mix (~1.6 with the default
    // quotas). Used to estimate the row count that just fits the items so the
    // large-list single solve does not have to probe many heights.
    const avgCellsPerCard = 1.6;
    const estimatedRows = Math.max(2, Math.ceil((items.length * avgCellsPerCard) / cols));
    const minRowsForCount = rich ? Math.max(2, Math.ceil(items.length / cols)) : estimatedRows;

    let bestResult: ReturnType<typeof generateBspCardLayout> = [];
    let chosenHeight = Math.max(minCanvasHeight, Math.round(minRowsForCount * unitH));

    // Try each candidate height (more rows = more room for 2×2 / 1×2 tiers),
    // and within each height try seed variants. Return the first whose solved
    // layout hosts every item (result.length >= items.length) for the smallest
    // viable height. Ties prefer the layout whose slot count is closest to
    // items.length (fewer wrap-fillers).
    //
    // The probe range is generous because `fillToFull` makes the slot count a
    // stepped function of rows (more rows ⇒ strictly more slots, but not
    // linearly), so reaching items.length can need several extra rows beyond
    // the naive ceil(count/cols) minimum. For large lists we cut both axes
    // (1 row probe, 1 seed) since `fillToFull` guarantees a hole-free tile
    // regardless of seed.
    const maxExtraRows = rich ? 4 : 0;
    const seedsPerHeight = rich ? 64 : 1;
    for (let extraRows = 0; extraRows <= maxExtraRows; extraRows++) {
      const rows = minRowsForCount + extraRows;
      const candidateHeight = Math.round(rows * unitH);
      if (candidateHeight < minCanvasHeight && extraRows === 0) continue;

      for (const seedOffset of Array.from({ length: seedsPerHeight }, (_, index) => index)) {
        const result = generateBspCardLayout({
          canvasWidth: width,
          canvasHeight: candidateHeight,
          targetCount: items.length,
          gap,
          minWidth: sourceMinWidth,
          minHeight: sourceMinHeight,
          minRatio,
          maxRatio,
          cols: pinnedColsRef.current ?? undefined,
          seed: seed + seedOffset * 0x9e3779b9,
        });
        const covers = result.length >= items.length;
        const better = covers && (
          bestResult.length < items.length ||
          result.length < bestResult.length ||
          (result.length === bestResult.length && bestResult.length === 0)
        );
        if (better) {
          bestResult = result;
          chosenHeight = candidateHeight;
        }
        if (covers) {
          prevSolveRef.current = { width, count: items.length, height: candidateHeight, layout: result };
          return { canvasHeight: candidateHeight, layout: result };
        }
      }
      if (bestResult.length >= items.length) break;
    }

    prevSolveRef.current = { width, count: items.length, height: chosenHeight, layout: bestResult };
    return { canvasHeight: chosenHeight, layout: bestResult };
  }, [gap, items.length, maxRatio, minCanvasHeight, minRatio, resolvedMinHeight, resolvedMinWidth, seed, width]);

  // Virtualization: only render cards whose placeholder is in (or near) the
  // viewport. The scroll container differs per host (outer page scroll vs an
  // inner overflow region), so observe against the browser viewport (root null)
  // — absolutely-positioned card visibility is fundamentally viewport-relative,
  // which makes this correct regardless of which ancestor scrolls.
  //
  // Card PLACEHOLDER divs are always rendered (they keep the canvas height and
  // absolute geometry), but `renderItem` is only called for visible ones, so
  // off-screen cards cost a few empty divs instead of their full subtree. This
  // is what keeps a 1000-card canvas cheap: React only mounts the handful of
  // cards actually on screen plus an overscan band.
  const slotRefs = useRef<Array<HTMLDivElement | null>>([]);
  const sentinelRef = useRef<HTMLDivElement | null>(null);
  const [visible, setVisible] = useState<Set<number>>(() => new Set());

  useEffect(() => {
    const visibleSet = new Set<number>();
    // root null + a generous rootMargin gives the overscan band: cards are
    // mounted slightly before they scroll into view and stay mounted slightly
    // after, so fast scrolling never shows empty slots.
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          const idx = Number((entry.target as HTMLElement).dataset.bspIndex);
          if (entry.isIntersecting) visibleSet.add(idx);
          else visibleSet.delete(idx);
        }
        // Always keep the first cards mounted so the top of the canvas is never
        // blank on initial paint before the observer fires.
        for (let i = 0; i < Math.min(layout.length, 6); i++) visibleSet.add(i);
        setVisible(new Set(visibleSet));
      },
      { root: null, rootMargin: "400px 0px 400px 0px" },
    );
    for (const el of slotRefs.current) if (el) observer.observe(el);
    return () => observer.disconnect();
  }, [layout]);

  // Load-more sentinel: when it nears the viewport and the host says more is
  // available, fire once. Disconnects while a load is pending are the host's
  // responsibility (it controls `hasMore`).
  useEffect(() => {
    if (!onLoadMore || !hasMore) return;
    const sentinel = sentinelRef.current;
    if (!sentinel) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) onLoadMore();
      },
      { root: null, rootMargin: "600px 0px 0px 0px" },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasMore, onLoadMore, layout.length]);

  return (
    <section
      ref={containerRef}
      aria-label={ariaLabel}
      className={`relative w-full ${className}`}
      style={{ height: canvasHeight }}
    >
      {layout.map((rect, index) => {
        // The solver tiles the whole canvas, so slot count can exceed items.
        // Wrap items so every slot shows real content — this is what makes the
        // no-hole layout read as a full feed rather than a row of empty frames.
        // The wrapped card keeps its own index for keying; the underlying item
        // is `items[index % items.length]`.
        const item = items.length > 0 ? items[index % items.length] : undefined;
        const isVisible = visible.has(index) || index < 6;
        return (
          <div
            key={index}
            data-bsp-index={index}
            ref={(el) => { slotRefs.current[index] = el; }}
            className="absolute"
            style={{ left: rect.x, top: rect.y, width: rect.width, height: rect.height }}
          >
            {isVisible && item !== undefined ? renderItem(item, index) : null}
          </div>
        );
      })}
      {hasMore && (
        <div
          ref={sentinelRef}
          aria-hidden="true"
          style={{ position: "absolute", bottom: 0, left: 0, right: 0, height: 1 }}
        />
      )}
    </section>
  );
}

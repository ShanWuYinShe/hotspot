import { cn } from "./utils";

function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="skeleton"
      className={cn("animate-pulse rounded-md", className)}
      style={{ background: "var(--neu-shadow-light)", ...props.style }}
      {...props}
    />
  );
}

function ListSkeleton({ rows = 4 }: { rows?: number }) {
  return (
    <div className="flex flex-col gap-3 py-2" role="status" aria-busy="true" aria-label="加载中">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 px-1">
          <Skeleton className="flex-shrink-0 !rounded-full" style={{ width: 44, height: 44 }} />
          <div className="flex-1 flex flex-col gap-2">
            <Skeleton style={{ height: 12, width: "55%" }} />
            <Skeleton style={{ height: 10, width: "35%" }} />
          </div>
        </div>
      ))}
    </div>
  );
}

function CardSkeleton({ count = 3 }: { count?: number }) {
  return (
    <div className="flex flex-col gap-4" role="status" aria-busy="true" aria-label="加载中">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="rounded-[18px] overflow-hidden" style={{ background: "var(--neu-bg)", border: "1px solid var(--neu-border)" }}>
          <Skeleton style={{ aspectRatio: "16/9", borderRadius: 0 }} />
          <div style={{ padding: "12px 16px 14px" }}>
            <Skeleton style={{ height: 12, width: "70%", marginBottom: 8 }} />
            <Skeleton style={{ height: 10, width: "40%" }} />
          </div>
        </div>
      ))}
    </div>
  );
}

export { Skeleton, ListSkeleton, CardSkeleton };

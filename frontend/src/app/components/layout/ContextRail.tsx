import React from "react";

type ContextRailProps = {
  children: React.ReactNode;
  /** Attached rails fill the available page height and own their scroll region. */
  attached?: boolean;
  className?: string;
};

/**
 * The single alignment primitive for all auxiliary right-hand content.
 * Width, divider, inset and top edge come from the shared context-rail CSS tokens.
 */
export function ContextRail({ children, attached = false, className = "" }: ContextRailProps) {
  return (
    <aside className={`context-rail ${attached ? "context-rail--attached" : ""} ${className}`}>
      <div className={attached ? "context-rail__content context-rail__content--scroll" : "context-rail__content"}>
        {children}
      </div>
    </aside>
  );
}

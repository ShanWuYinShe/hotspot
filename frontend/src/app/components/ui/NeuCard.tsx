import React from "react";
import { cn } from "../../../lib/utils";

interface NeuCardProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: "flat" | "pressed" | "convex";
}

export const NeuCard = React.forwardRef<HTMLDivElement, NeuCardProps>(
  ({ className, variant = "flat", children, ...props }, ref) => {
    return (
      <div
        ref={ref}
        className={cn(
          "rounded-[32px] p-6 transition-all duration-300",
          variant === "flat" && "neu-flat",
          variant === "pressed" && "neu-pressed",
          variant === "convex" && "neu-convex",
          className
        )}
        {...props}
      >
        {children}
      </div>
    );
  }
);
NeuCard.displayName = "NeuCard";

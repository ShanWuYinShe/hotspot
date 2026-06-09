import React from "react";
import { cn } from "../../../lib/utils";

interface NeuInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  icon?: React.ReactNode;
}

export const NeuInput = React.forwardRef<HTMLInputElement, NeuInputProps>(
  ({ className, icon, ...props }, ref) => {
    return (
      <div className="relative flex items-center w-full">
        {icon && (
          <div className="absolute left-4 text-neu-muted pointer-events-none">
            {icon}
          </div>
        )}
        <input
          ref={ref}
          className={cn(
            // neu-input-base: convex by default, concave on :focus (pure CSS, no JS state needed)
            "w-full neu-input-base rounded-2xl py-3 px-4 placeholder:text-neu-muted transition-all duration-200",
            icon && "pl-12",
            className,
          )}
          {...props}
        />
      </div>
    );
  },
);
NeuInput.displayName = "NeuInput";

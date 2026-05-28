import React from "react";
import { cn } from "../../../lib/utils";

interface NeuAvatarProps extends React.HTMLAttributes<HTMLDivElement> {
  src: string;
  size?: "sm" | "md" | "lg" | "xl" | "hero";
  alt?: string;
  active?: boolean;
}

export const NeuAvatar = React.forwardRef<HTMLDivElement, NeuAvatarProps>(
  ({ className, src, size = "md", alt = "Avatar", active, ...props }, ref) => {
    return (
      <div
        ref={ref}
        className={cn(
          "relative rounded-full flex items-center justify-center neu-flat transition-all duration-300 flex-shrink-0",
          size === "sm" && "w-10 h-10 p-[3px]",
          size === "md" && "w-14 h-14 p-[4px]",
          size === "lg" && "w-20 h-20 p-[5px]",
          size === "xl" && "w-28 h-28 p-[6px]",
          size === "hero" && "w-[120px] h-[120px] p-[6px]",
          active && "neu-pressed",
          className,
        )}
        {...props}
      >
        <div className="w-full h-full rounded-full overflow-hidden relative neu-pressed flex items-center justify-center">
          <img
            src={src}
            alt={alt}
            className="w-[120%] h-[120%] object-cover rounded-full pointer-events-none opacity-90 blur-[1px]" // Slight blur to match the abstract look in the design
          />
        </div>
      </div>
    );
  },
);
NeuAvatar.displayName = "NeuAvatar";

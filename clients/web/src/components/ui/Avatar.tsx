import { cn, getInitials, getAvatarColor } from "../../lib/utils";
import type { PresenceStatus } from "@feather/api-client";

interface AvatarProps {
  src?: string | null;
  name: string;
  id?: string;
  size?: "xs" | "sm" | "md" | "lg";
  status?: PresenceStatus;
  className?: string;
  style?: React.CSSProperties;
  onClick?: () => void;
}

export function Avatar({
  src,
  name,
  id,
  size = "md",
  status,
  className,
  style,
  onClick,
}: AvatarProps) {
  const sizes = {
    xs: "w-5 h-5 text-[10px]",
    sm: "w-6 h-6 text-xs",
    md: "w-8 h-8 text-sm",
    lg: "w-10 h-10 text-base",
  };

  const statusColors = {
    online: "bg-green-500",
    offline: "bg-gray-400",
  };

  const statusSizes = {
    xs: "w-2 h-2 right-[-1px] bottom-[-1px] border-2",
    sm: "w-2 h-2 right-0 bottom-0 border-2",
    md: "w-2.5 h-2.5 right-0 bottom-0 border-2",
    lg: "w-3 h-3 right-0 bottom-0 border-2",
  };

  const content = (
    <>
      {src ? (
        <img
          src={src}
          alt={name}
          className={cn("rounded-full object-cover", sizes[size])}
        />
      ) : (
        <div
          className={cn(
            "rounded-full flex items-center justify-center font-medium text-white",
            id ? getAvatarColor(id) : "bg-primary-500",
            sizes[size],
          )}
        >
          {getInitials(name)}
        </div>
      )}
      {status && (
        <span
          className={cn(
            "absolute rounded-full border-white dark:border-gray-900",
            statusColors[status],
            statusSizes[size],
          )}
        />
      )}
    </>
  );

  if (onClick) {
    return (
      <button
        type="button"
        onClick={onClick}
        className={cn(
          "relative inline-block rounded-full cursor-pointer",
          className,
        )}
        style={style}
      >
        {content}
      </button>
    );
  }

  return (
    <div
      className={cn("relative inline-block rounded-full", className)}
      style={style}
    >
      {content}
    </div>
  );
}

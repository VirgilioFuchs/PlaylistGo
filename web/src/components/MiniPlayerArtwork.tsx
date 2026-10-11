import type { TrackSnapshot } from "../types";
import { getThumbnailForWidth } from "../lib/thumbnail";

type MiniPlayerArtworkProps = {
  track: TrackSnapshot | null;
  count: number;
  variant: "bar" | "compact" | "queue" | "closed" | "summary";
  className?: string;
};

export function MiniPlayerArtwork({ track, count, variant, className = "" }: MiniPlayerArtworkProps) {
  const label = count > 0 ? String(count).padStart(2, "0") : "00";
  const thumbnail = track ? getThumbnailForWidth(track, getArtworkWidth(variant)) : null;
  const imageStyle = thumbnail ? { backgroundImage: `url(${thumbnail.url})` } : undefined;

  return (
    <div
      className={`mini-player-artwork mini-player-artwork--${variant}${thumbnail ? " mini-player-artwork--image" : ""}${className ? ` ${className}` : ""}`}
      aria-hidden="true"
      style={imageStyle}
    >
      {thumbnail ? null : <span>{label}</span>}
    </div>
  );
}

function getArtworkWidth(variant: MiniPlayerArtworkProps["variant"]): number {
  switch (variant) {
    case "bar":
    case "closed":
      return 56;
    case "compact":
      return 48;
    case "queue":
    case "summary":
      return typeof window === "undefined" ? 640 : window.innerWidth;
  }
}

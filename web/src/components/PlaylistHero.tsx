import type { TrackSnapshot } from "../types";
import { MiniPlayerArtwork } from "./MiniPlayerArtwork";

type PlaylistHeroProps = {
  track: TrackSnapshot | null;
  count: number;
  duration: string;
  title: string;
  subtitle: string;
  description?: string;
  compact?: boolean;
  translucent?: boolean;
};

export function PlaylistHero({
  track,
  count,
  duration,
  title,
  subtitle,
  description,
  compact = false,
  translucent = false
}: PlaylistHeroProps) {
  return (
    <section
      className={`playlist-hero${compact ? " playlist-hero--compact" : ""}${track?.thumbnailUrl ? " playlist-hero--image" : ""}${translucent ? " playlist-hero--translucent" : ""}`}
    >
      <MiniPlayerArtwork track={track} count={count} variant={compact ? "compact" : "queue"} className="playlist-hero__artwork" />
      <div className="playlist-hero__copy">
        <p className="shell-kicker">Montando playlist</p>
        <h3>{title}</h3>
        <p>{subtitle}</p>
        {description ? <p className="playlist-hero__description">{description}</p> : null}
      </div>
      <div className="playlist-hero__duration">
        <span>{duration}</span>
      </div>
    </section>
  );
}

import type { TrackSnapshot } from "../types";
import { formatDuration } from "../lib/format";
import { getThumbnailForWidth } from "../lib/thumbnail";
import { IconButton } from "./IconButton";

type TrackCardProps = {
  track: TrackSnapshot;
  selected: boolean;
  locked?: boolean;
  onToggle: (track: TrackSnapshot) => void;
};

export function TrackCard({ track, selected, locked = false, onToggle }: TrackCardProps) {
  const thumbnail = getThumbnailForWidth(track, 320);

  return (
    <article className={`track-card${selected ? " track-card--selected" : ""}`}>
      {thumbnail ? (
        <img
          className="track-card__thumb"
          src={thumbnail.url}
          alt={track.title}
          width={thumbnail.width ?? 320}
          height={thumbnail.height ?? 180}
          loading="lazy"
          decoding="async"
        />
      ) : <div className="track-card__thumb" aria-hidden="true" />}
      <div className="track-card__body">
        <div className="track-card__meta">
          <p className="track-card__title">{track.title}</p>
          <p className="track-card__channel">{track.channel}</p>
        </div>
        <div className="track-card__footer">
          <span className="track-card__duration">{formatDuration(track.durationSeconds)}</span>
          <IconButton
            type="button"
            selected={selected}
            aria-label={selected ? `Remover ${track.title} da seleção` : `Adicionar ${track.title} à seleção`}
            onClick={() => onToggle(track)}
            disabled={locked}
          >
            {selected ? "−" : "+"}
          </IconButton>
        </div>
      </div>
    </article>
  );
}

import type { TrackSnapshot } from "../types";
import { formatDuration } from "../lib/format";
import { TrackRow } from "./TrackRow";

type QueueTrackListProps = {
  tracks: TrackSnapshot[];
  locked: boolean;
  onRemove: (videoId: string) => void;
  ariaLabel: string;
  previewCount?: number;
};

export function QueueTrackList({ tracks, locked, onRemove, ariaLabel, previewCount }: QueueTrackListProps) {
  const visibleTracks = typeof previewCount === "number" ? tracks.slice(0, previewCount) : tracks;

  return (
    <section className="mini-player__queue-list" aria-label={ariaLabel}>
      {visibleTracks.length === 0 ? (
        <p className="mini-player__empty">
          {previewCount ? "Selecione faixas na busca para montar fila." : "Nenhuma faixa selecionada ainda."}
        </p>
      ) : (
        <div className="mini-player__queue-items">
          {visibleTracks.map((track) => (
            <TrackRow
              key={track.videoId}
              title={track.title}
              subtitle={`${track.channel} · ${formatDuration(track.durationSeconds)}`}
              actionLabel={`Remover ${track.title} da fila`}
              onAction={() => onRemove(track.videoId)}
              disabled={locked}
            />
          ))}
          {typeof previewCount === "number" && tracks.length > previewCount ? (
            <p className="mini-player__hint">+{tracks.length - previewCount} faixas aguardando vista completa.</p>
          ) : null}
        </div>
      )}
    </section>
  );
}

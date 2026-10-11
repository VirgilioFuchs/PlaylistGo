import type { TrackSnapshot } from "../types";
import { SurfaceSection } from "./SurfaceSection";
import { TrackCard } from "./TrackCard";

type SearchResultsProps = {
  tracks: TrackSnapshot[];
  selectedIds: Set<string>;
  loading?: boolean;
  locked?: boolean;
  emptyMessage?: string;
  title?: string;
  description?: string;
  onToggle: (track: TrackSnapshot) => void;
};

export function SearchResults({
  tracks,
  selectedIds,
  loading = false,
  locked = false,
  emptyMessage = "Pesquise uma faixa para começar.",
  title = "Seleção e revisão",
  description = "Use os cards para montar sua playlist.",
  onToggle
}: SearchResultsProps) {
  if (loading) {
    return (
      <section className="results-state" aria-live="polite">
        <p>Carregando resultados...</p>
      </section>
    );
  }

  if (tracks.length === 0) {
    return (
      <section className="results-state results-state--empty" aria-live="polite">
        <p>{emptyMessage}</p>
      </section>
    );
  }

  return (
    <SurfaceSection className="results-section" eyebrow="Resultados" title={title} description={description}>
      <div className="results-grid" aria-label="Resultados da busca">
        {tracks.map((track) => (
          <TrackCard
            key={track.videoId}
            track={track}
            selected={selectedIds.has(track.videoId)}
            locked={locked}
            onToggle={onToggle}
          />
        ))}
      </div>
    </SurfaceSection>
  );
}

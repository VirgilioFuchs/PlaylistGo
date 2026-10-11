import type { PointerEvent, WheelEvent } from "react";
import type { TrackSnapshot } from "../types";
import { formatDuration, formatTrackCount } from "../lib/format";
import { MiniPlayerArtwork } from "./MiniPlayerArtwork";
import styles from "./ExpandedMiniPlayer.module.css";

type PlayerSummaryPaneProps = {
  artworkTrack: TrackSnapshot | null;
  title: string;
  trackCount: number;
  durationSeconds: number;
  source: "queue" | "playlist";
  ownerName?: string;
  onRequestClose: () => void;
  onPanePointerDown: (event: PointerEvent<HTMLElement>) => void;
  onPaneWheel: (event: WheelEvent<HTMLElement>) => void;
  onQueueHandlePointerDown: (event: PointerEvent<HTMLElement>) => void;
  onQueueHandleClick: () => void;
};

export function PlayerSummaryPane({
  artworkTrack,
  title,
  trackCount,
  durationSeconds,
  source,
  ownerName,
  onRequestClose,
  onPanePointerDown,
  onPaneWheel,
  onQueueHandlePointerDown,
  onQueueHandleClick
}: PlayerSummaryPaneProps) {
  const subtitle = source === "queue" ? "Fila temporária" : "Playlist";
  const displayOwnerName = ownerName?.trim() || "você";

  return (
    <section
      className={styles.summaryPane}
      aria-label="Resumo da playlist"
      onWheel={onPaneWheel}
      onPointerDown={(event) => {
        if (!(event.target instanceof Element) || !event.target.closest("button")) {
          onPanePointerDown(event);
        }
      }}
    >
      <header className={styles.summaryHeader}>
        <button
          type="button"
          className={styles.iconButton}
          onClick={onRequestClose}
          aria-label="Recolher mini-player"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
            <path d="m6 9 6 6 6-6" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" />
          </svg>
        </button>
        <p className={styles.summaryContext}>{`Playlist de ${displayOwnerName}`}</p>
      </header>
      <MiniPlayerArtwork
        track={artworkTrack}
        count={trackCount}
        variant="summary"
        className={styles.summaryBackdrop}
      />
      <div className={styles.summaryScrim} aria-hidden="true" />
      <div className={styles.summaryBody}>
        <div className={styles.summaryCopy}>
          <h2 className={styles.summaryTitle} title={title}>{title}</h2>
          <p className={styles.summarySubtitle}>{subtitle}</p>
          <p className={styles.summaryMeta}>{`${formatTrackCount(trackCount)} · ${formatDuration(durationSeconds)}`}</p>
        </div>
      </div>
      <footer className={styles.summaryFooter}>
        <button
          type="button"
          className={styles.queueHandle}
          onPointerDown={onQueueHandlePointerDown}
          onClick={onQueueHandleClick}
          aria-label="Abrir fila de faixas"
        >
          <span className={styles.grip} aria-hidden="true" />
          <span>Fila de faixas</span>
        </button>
      </footer>
    </section>
  );
}

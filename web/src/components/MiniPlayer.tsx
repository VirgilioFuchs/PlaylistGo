import { useRef } from "react";
import { motion, useAnimationControls } from "framer-motion";
import type { PanInfo } from "framer-motion";
import type { WheelEvent as ReactWheelEvent } from "react";
import type { TrackSnapshot } from "../types";
import { formatDuration, formatTrackCount } from "../lib/format";
import { ExpandedMiniPlayer } from "./ExpandedMiniPlayer";
import { MiniPlayerArtwork } from "./MiniPlayerArtwork";
import styles from "./MiniPlayer.module.css";

export type MiniPlayerLifecyclePhase = "idle" | "preparing" | "processing" | "finalizing" | "completed" | "failed";
export type MiniPlayerLifecycle = {
  phase: MiniPlayerLifecyclePhase;
  confirmedTrackCount: number;
  totalTrackCount: number;
};

type MiniPlayerProps = {
  artworkTrack: TrackSnapshot | null;
  tracks?: TrackSnapshot[];
  title: string;
  ownerName?: string;
  trackCount: number;
  durationSeconds: number;
  source: "queue" | "playlist";
  isExpanded?: boolean;
  busy?: boolean;
  lifecycle: MiniPlayerLifecycle;
  isExiting?: boolean;
  onRequestExpand: () => void;
  onRequestCollapse?: () => void;
  onExitComplete?: () => void;
};

export function MiniPlayer({
  artworkTrack,
  tracks = [],
  title,
  ownerName,
  trackCount,
  durationSeconds,
  source,
  isExpanded = false,
  busy = false,
  lifecycle,
  isExiting = false,
  onRequestExpand,
  onRequestCollapse = () => undefined,
  onExitComplete
}: MiniPlayerProps) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const triggerControls = useAnimationControls();
  const formattedCount = formatTrackCount(trackCount);
  const formattedDuration = formatDuration(durationSeconds);
  const status = getLifecycleStatus(lifecycle, title, formattedCount);
  const accessibleName = status
    ? `${status.title}. ${status.detail}. ${formattedDuration}.`
    : `${title}. ${formattedCount}. ${formattedDuration}.`;
  const rootClassName = [styles.root, isExiting ? styles.exiting : ""].filter(Boolean).join(" ");

  function handleClosedDragEnd(_: MouseEvent | TouchEvent | globalThis.PointerEvent, info: PanInfo) {
    const triggerHeight = triggerRef.current?.getBoundingClientRect().height ?? 1;
    const crossedTravelThreshold = -info.offset.y > triggerHeight * 0.55;
    const crossedVelocityThreshold = -info.velocity.y > 500;
    if (crossedTravelThreshold || crossedVelocityThreshold) {
      triggerControls.set({ y: 0 });
      onRequestExpand();
      return;
    }

    void triggerControls.start({ y: 0, transition: { type: "spring", duration: 0.26, bounce: 0 } });
  }

  function handleClosedWheel(event: ReactWheelEvent<HTMLButtonElement>) {
    if (event.deltaY >= 0 || busy || isExiting) {
      return;
    }

    event.preventDefault();
    onRequestExpand();
  }

  return (
    <>
      <section
        className={rootClassName}
        data-mini-player-closed="true"
        data-source={source}
        data-phase={lifecycle.phase}
        aria-label="Mini-player fechado"
        onAnimationEnd={
          isExiting
            ? (event) => {
                if (event.currentTarget === event.target) {
                  onExitComplete?.();
                }
              }
            : undefined
        }
      >
        <motion.button
          ref={triggerRef}
          type="button"
          className={styles.trigger}
          animate={triggerControls}
          drag="y"
          dragConstraints={{ bottom: 0, top: -window.innerHeight }}
          dragElastic={0.18}
          dragMomentum={false}
          onDragEnd={handleClosedDragEnd}
          onWheel={handleClosedWheel}
          onClick={onRequestExpand}
          disabled={busy || isExiting}
          aria-busy={busy || isExiting || undefined}
          aria-expanded={isExpanded}
          aria-label={accessibleName}
        >
          <MiniPlayerArtwork
            track={artworkTrack}
            count={trackCount}
            variant="closed"
            className={styles.artwork}
          />
          <span className={styles.copy}>
            <strong title={status?.title ?? title}>{status?.title ?? title}</strong>
            <span>{status?.detail ?? formattedCount}</span>
          </span>
          <span className={styles.duration} aria-hidden="true">{formattedDuration}</span>
        </motion.button>
        {status ? <span className={styles.liveStatus} aria-live="polite">{status.announcement}</span> : null}
      </section>
      <ExpandedMiniPlayer
        artworkTrack={artworkTrack}
        durationSeconds={durationSeconds}
        open={isExpanded}
        returnFocusRef={triggerRef}
        source={source}
        title={title}
        ownerName={ownerName}
        trackCount={trackCount}
        tracks={tracks}
        onRequestClose={onRequestCollapse}
      />
    </>
  );
}

function getLifecycleStatus(
  lifecycle: MiniPlayerLifecycle,
  title: string,
  formattedCount: string
): { title: string; detail: string; announcement: string } | null {
  const progress = `${lifecycle.confirmedTrackCount} de ${lifecycle.totalTrackCount} faixas confirmadas`;

  switch (lifecycle.phase) {
    case "preparing":
      return {
        title: "Preparando playlist",
        detail: "Criando destino",
        announcement: "Preparando playlist."
      };
    case "processing":
      return {
        title: "Adicionando faixas",
        detail: progress,
        announcement: "Adicionando faixas confirmadas à playlist."
      };
    case "finalizing":
      return {
        title: "Finalizando playlist",
        detail: progress,
        announcement: "Finalizando playlist."
      };
    case "completed":
      return {
        title: "Playlist concluída",
        detail: formattedCount,
        announcement: "Playlist concluída."
      };
    case "failed":
      return {
        title: "Não foi possível finalizar",
        detail: "A fila foi preservada",
        announcement: "Não foi possível finalizar a playlist. A fila foi preservada."
      };
    case "idle":
      return null;
    default:
      return { title, detail: formattedCount, announcement: title };
  }
}

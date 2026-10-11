import { Fragment, useRef, useState } from "react";
import type { PointerEvent } from "react";
import { motion, useAnimationControls, useDragControls } from "framer-motion";
import type { TrackSnapshot } from "../types";
import { formatDuration, formatTrackCount } from "../lib/format";
import { MiniPlayerArtwork } from "./MiniPlayerArtwork";
import styles from "./ExpandedMiniPlayer.module.css";

type QueuePreviewPaneProps = {
  artworkTrack: TrackSnapshot | null;
  ownerName?: string;
  title: string;
  source: "queue" | "playlist";
  trackCount: number;
  tracks: TrackSnapshot[];
  onRequestClose: () => void;
  onDraftTracksChange: (tracks: TrackSnapshot[]) => void;
  onPanePointerDown: (event: PointerEvent<HTMLElement>) => void;
  onQueueHandlePointerDown: (event: PointerEvent<HTMLElement>) => void;
  onQueueHandleClick: () => void;
};

type QueuePreviewTrackRowProps = {
  track: TrackSnapshot;
  index: number;
  onDragStart: () => void;
  onDragMove: (offsetY: number, rowHeight: number) => void;
  onDragEnd: (offsetY: number, rowHeight: number) => void;
  onKeyboardMove: (direction: -1 | 1) => void;
};

function QueuePreviewTrackRow({
  track,
  index,
  onDragStart,
  onDragMove,
  onDragEnd,
  onKeyboardMove
}: QueuePreviewTrackRowProps) {
  const rowRef = useRef<HTMLLIElement>(null);
  const rowControls = useAnimationControls();
  const dragControls = useDragControls();

  return (
    <motion.li
      ref={rowRef}
      className={styles.trackRow}
      animate={rowControls}
      layout="position"
      drag="y"
      dragControls={dragControls}
      dragListener={false}
      dragElastic={0}
      dragMomentum={false}
      onDragStart={onDragStart}
      onDrag={(_, info) => onDragMove(info.offset.y, rowRef.current?.getBoundingClientRect().height ?? 0)}
      onDragEnd={(_, info) => {
        onDragEnd(info.offset.y, rowRef.current?.getBoundingClientRect().height ?? 0);
        void rowControls.start({ y: 0, transition: { duration: 0 } });
      }}
    >
      <span className={styles.trackPosition} aria-hidden="true">{index + 1}</span>
      <span className={styles.queueTrackCopy}>
        <span className={styles.trackTitle} title={track.title}>{track.title}</span>
        <span className={styles.trackDuration}>{formatDuration(track.durationSeconds)}</span>
      </span>
      <button
        type="button"
        className={styles.reorderHandle}
        aria-label={`Reordenar ${track.title}`}
        onPointerDown={(event) => {
          dragControls.start(event, { snapToCursor: false });
        }}
        onKeyDown={(event) => {
          if (event.key === "ArrowUp" || event.key === "ArrowDown") {
            event.preventDefault();
            onKeyboardMove(event.key === "ArrowUp" ? -1 : 1);
          }
        }}
      >
        <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
          <path d="M5 7h14M5 12h14M5 17h14" fill="none" stroke="currentColor" strokeLinecap="round" strokeWidth="2" />
        </svg>
      </button>
    </motion.li>
  );
}

export function QueuePreviewPane({
  artworkTrack,
  ownerName,
  title,
  source,
  trackCount,
  tracks,
  onRequestClose,
  onDraftTracksChange,
  onPanePointerDown,
  onQueueHandlePointerDown,
  onQueueHandleClick
}: QueuePreviewPaneProps) {
  const [draggedTrackId, setDraggedTrackId] = useState<string | null>(null);
  const [dragTargetIndex, setDragTargetIndex] = useState<number | null>(null);
  const dragTargetIndexRef = useRef<number | null>(null);
  const queueTracks = tracks;

  function getDragTargetIndex(videoId: string, offsetY: number, rowHeight: number): number | null {
    const sourceIndex = queueTracks.findIndex((track) => track.videoId === videoId);
    if (sourceIndex < 0 || rowHeight <= 0) {
      return sourceIndex < 0 ? null : sourceIndex;
    }

    return Math.max(
      0,
      Math.min(queueTracks.length - 1, sourceIndex + Math.round(offsetY / rowHeight))
    );
  }

  function updateDraftOrder(videoId: string, targetIndex: number | null) {
    const sourceIndex = tracks.findIndex((track) => track.videoId === videoId);
    if (sourceIndex < 0 || targetIndex === null || sourceIndex === targetIndex) {
      return;
    }

    const nextTracks = [...tracks];
    const [movedTrack] = nextTracks.splice(sourceIndex, 1);
    nextTracks.splice(targetIndex, 0, movedTrack);
    onDraftTracksChange(nextTracks);
  }

  function handleTrackDragStart(videoId: string) {
    const sourceIndex = queueTracks.findIndex((track) => track.videoId === videoId);
    dragTargetIndexRef.current = sourceIndex >= 0 ? sourceIndex : null;
    setDraggedTrackId(videoId);
    setDragTargetIndex(dragTargetIndexRef.current);
  }

  function handleTrackDragMove(videoId: string, offsetY: number, rowHeight: number) {
    const targetIndex = getDragTargetIndex(videoId, offsetY, rowHeight);
    dragTargetIndexRef.current = targetIndex;
    setDragTargetIndex((current) => current === targetIndex ? current : targetIndex);
  }

  function handleTrackDragEnd(videoId: string, offsetY: number, rowHeight: number) {
    const targetIndex = getDragTargetIndex(videoId, offsetY, rowHeight) ?? dragTargetIndexRef.current;
    updateDraftOrder(videoId, targetIndex);
    dragTargetIndexRef.current = null;
    setDraggedTrackId(null);
    setDragTargetIndex(null);
  }

  function handleKeyboardMove(videoId: string, direction: -1 | 1) {
    const sourceIndex = queueTracks.findIndex((track) => track.videoId === videoId);
    if (sourceIndex < 0) {
      return;
    }

    const targetIndex = Math.max(0, Math.min(queueTracks.length - 1, sourceIndex + direction));
    updateDraftOrder(videoId, targetIndex);
  }

  const draggedSourceIndex = draggedTrackId === null
    ? -1
    : queueTracks.findIndex((track) => track.videoId === draggedTrackId);
  const dropPlaceholderIndex = draggedSourceIndex >= 0
    && dragTargetIndex !== null
    && dragTargetIndex !== draggedSourceIndex
    ? dragTargetIndex > draggedSourceIndex ? dragTargetIndex + 1 : dragTargetIndex
    : null;

  const subtitle = source === "queue" ? "Fila" : "Playlist";
  const displayOwnerName = ownerName?.trim() || "você";
  const durationSeconds = queueTracks.reduce((total, track) => total + track.durationSeconds, 0);
  const emptyMessage = source === "queue"
    ? "Selecione faixas na busca para montar fila."
    : "Esta playlist não tem faixas.";

  return (
    <section
      className={styles.queuePane}
      aria-label="Fila de faixas"
      onPointerDown={(event) => {
        if (!(event.target instanceof Element) || !event.target.closest("button, [data-scroll-owner]")) {
          onPanePointerDown(event);
        }
      }}
    >
      <section className={styles.queueContext} aria-label="Resumo da playlist">
        <MiniPlayerArtwork
          track={artworkTrack}
          count={trackCount}
          variant="summary"
          className={styles.queueContextBackdrop}
        />
        <div className={styles.queueContextScrim} aria-hidden="true" />
        <header className={styles.queueContextHeader}>
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
        <div className={styles.queueContextBody}>
          <h2 className={styles.queueContextTitle} title={title}>{title}</h2>
          <p className={styles.queueContextSubtitle}>{subtitle}</p>
          <p className={styles.queueContextMeta}>{`${formatTrackCount(trackCount)} · ${formatDuration(durationSeconds)}`}</p>
        </div>
      </section>
      <header className={styles.queueHeader}>
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
        <div className={styles.queueTitleBlock}>
          <MiniPlayerArtwork
            track={artworkTrack}
            count={trackCount}
            variant="compact"
            className={styles.compactArtwork}
          />
          <h2 title={title}>{title}</h2>
        </div>
      </header>
      <footer className={styles.queueFooter}>
        <button
          type="button"
          className={styles.queueHandle}
          onPointerDown={onQueueHandlePointerDown}
          onClick={onQueueHandleClick}
          aria-label={"Abrir gest\u00e3o da fila"}
        >
          <span className={styles.grip} aria-hidden="true" />
          <span>Fila de faixas</span>
        </button>
      </footer>
      <div className={styles.queueViewport} data-scroll-owner>
        {queueTracks.length === 0 ? <p className={styles.emptyQueue}>{emptyMessage}</p> : (
          <ol className={styles.trackList} aria-label="Faixas da fila">
            {queueTracks.map((track, index) => (
              <Fragment key={track.videoId}>
                {dropPlaceholderIndex === index ? (
                  <li className={styles.dropPlaceholder} aria-hidden="true" />
                ) : null}
                <QueuePreviewTrackRow
                  track={track}
                  index={index}
                  onDragStart={() => handleTrackDragStart(track.videoId)}
                  onDragMove={(offsetY, rowHeight) => handleTrackDragMove(track.videoId, offsetY, rowHeight)}
                  onDragEnd={(offsetY, rowHeight) => handleTrackDragEnd(track.videoId, offsetY, rowHeight)}
                  onKeyboardMove={(direction) => handleKeyboardMove(track.videoId, direction)}
                />
              </Fragment>
            ))}
            {dropPlaceholderIndex === queueTracks.length ? (
              <li className={styles.dropPlaceholder} aria-hidden="true" />
            ) : null}
          </ol>
        )}
      </div>
    </section>
  );
}

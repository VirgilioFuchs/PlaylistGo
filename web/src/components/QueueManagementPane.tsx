import { Fragment, useRef, useState } from "react";
import { motion, useAnimationControls } from "framer-motion";
import type { PanInfo } from "framer-motion";
import type { TrackSnapshot } from "../types";
import { formatDuration } from "../lib/format";
import styles from "./ExpandedMiniPlayer.module.css";

type QueueManagementPaneProps = {
  source: "queue" | "playlist";
  tracks: TrackSnapshot[];
  isEditing: boolean;
  onRequestQueuePreview: () => void;
  onRequestClose: () => void;
  onStartEditing: () => void;
  onDraftTracksChange: (tracks: TrackSnapshot[]) => void;
  onCompleteEditing: () => void;
  onCommitTemporaryOrder: (orderedIds: string[]) => void;
  onCommitPersistedOrder: (orderedIds: string[]) => Promise<void>;
  onDeleteTemporary: (videoId: string) => void;
  onDeletePersisted: (videoId: string) => Promise<void>;
};

type RetryMutation =
  | { kind: "order" }
  | { kind: "delete"; videoId: string }
  | { kind: "delete-batch"; videoIds: string[] }
  | null;

const REORDER_FAILURE_MESSAGE = "Não foi possível salvar a nova ordem. Tente novamente.";
const DELETE_FAILURE_MESSAGE = "Não foi possível excluir a faixa. Tente novamente.";

const PANE_TRAVEL_RATIO = 0.55;
const FLING_VELOCITY = 500;

type QueueManagementTrackRowProps = {
  track: TrackSnapshot;
  index: number;
  isEditing: boolean;
  isBusy: boolean;
  isConfirmingDelete: boolean;
  isSelectedForDelete: boolean;
  onDragStart: () => void;
  onDragMove: (offsetY: number, rowHeight: number) => void;
  onDragEnd: (offsetY: number, rowHeight: number) => void;
  onToggleDeleteSelection: (checked: boolean) => void;
  onConfirmDelete: () => void;
  onCancelDelete: () => void;
  onRequestDelete: () => void;
};

function QueueManagementTrackRow({
  track,
  index,
  isEditing,
  isBusy,
  isConfirmingDelete,
  isSelectedForDelete,
  onDragStart,
  onDragMove,
  onDragEnd,
  onToggleDeleteSelection,
  onConfirmDelete,
  onCancelDelete,
  onRequestDelete
}: QueueManagementTrackRowProps) {
  const rowRef = useRef<HTMLLIElement>(null);
  const rowControls = useAnimationControls();

  return (
    <motion.li
      ref={rowRef}
      className={[
        styles.managementRow,
        isEditing ? styles.managementRowEditing : "",
        isConfirmingDelete ? styles.managementRowConfirming : ""
      ].filter(Boolean).join(" ")}
      animate={rowControls}
      layout="position"
      drag={isEditing ? "y" : false}
      dragElastic={0}
      dragMomentum={false}
      onDragStart={onDragStart}
      onDrag={(_, info) => onDragMove(info.offset.y, rowRef.current?.getBoundingClientRect().height ?? 0)}
      onDragEnd={(_, info) => {
        onDragEnd(info.offset.y, rowRef.current?.getBoundingClientRect().height ?? 0);
        void rowControls.start({ y: 0, transition: { duration: 0 } });
      }}
    >
      <span className={styles.trackSelection}>
        <span className={styles.trackPosition} aria-hidden={!isEditing}>{index + 1}</span>
        {!isEditing ? (
          <label className={styles.deleteSelection}>
            <input
              type="checkbox"
              checked={isSelectedForDelete}
              onChange={(event) => onToggleDeleteSelection(event.target.checked)}
              disabled={isBusy}
              aria-label={`Selecionar ${track.title} para exclusão`}
            />
            <span className={styles.deleteCheckboxIndicator} aria-hidden="true" />
          </label>
        ) : null}
      </span>
      <span className={styles.managementTrackCopy}>
        <span className={styles.trackTitle} title={track.title}>{track.title}</span>
        <span className={styles.trackDuration}>{formatDuration(track.durationSeconds)}</span>
      </span>
      {isEditing ? (
        <button
          type="button"
          className={styles.reorderHandle}
          aria-label={`Reordenar ${track.title}`}
        >
          <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
            <path d="M5 7h14M5 12h14M5 17h14" fill="none" stroke="currentColor" strokeLinecap="round" strokeWidth="2" />
          </svg>
        </button>
      ) : isConfirmingDelete ? (
        <span className={styles.inlineConfirmation}>
          <button
            type="button"
            className={styles.confirmDeleteButton}
            onClick={onConfirmDelete}
            disabled={isBusy}
          >
            Confirmar excluir
          </button>
          <button
            type="button"
            className={styles.cancelDeleteButton}
            onClick={onCancelDelete}
            disabled={isBusy}
          >
            Cancelar
          </button>
        </span>
      ) : (
        <button
          type="button"
          className={styles.deleteButton}
          onClick={onRequestDelete}
          disabled={isBusy}
          aria-label={`Excluir ${track.title} da fila`}
        >
          Excluir
        </button>
      )}
    </motion.li>
  );
}

export function QueueManagementPane({
  source,
  tracks,
  isEditing,
  onRequestQueuePreview,
  onRequestClose,
  onStartEditing,
  onDraftTracksChange,
  onCompleteEditing,
  onCommitTemporaryOrder,
  onCommitPersistedOrder,
  onDeleteTemporary,
  onDeletePersisted
}: QueueManagementPaneProps) {
  const [confirmingDeleteId, setConfirmingDeleteId] = useState<string | null>(null);
  const [pendingMutation, setPendingMutation] = useState<"order" | "delete" | null>(null);
  const [mutationError, setMutationError] = useState<string | null>(null);
  const [retryMutation, setRetryMutation] = useState<RetryMutation>(null);
  const [selectedDeleteIds, setSelectedDeleteIds] = useState<Set<string>>(() => new Set());
  const [draggedTrackId, setDraggedTrackId] = useState<string | null>(null);
  const [dragTargetIndex, setDragTargetIndex] = useState<number | null>(null);
  const dragTargetIndexRef = useRef<number | null>(null);
  const drawerControls = useAnimationControls();

  function returnToQueuePreview(event: MouseEvent | TouchEvent | globalThis.PointerEvent, info: PanInfo) {
    const availableTravel = (event.currentTarget as HTMLElement | null)?.parentElement?.getBoundingClientRect().height
      ?? window.innerHeight;
    const exceedsDistance = info.offset.y > availableTravel * PANE_TRAVEL_RATIO;
    const exceedsVelocity = info.velocity.y > FLING_VELOCITY;

    if (exceedsDistance || exceedsVelocity) {
      onRequestQueuePreview();
      return;
    }

    void drawerControls.start({ y: 0, transition: { type: "spring", duration: 0.26, bounce: 0 } });
  }

  function beginEditing() {
    setMutationError(null);
    setRetryMutation(null);
    setSelectedDeleteIds(new Set());
    setConfirmingDeleteId(null);
    setDraggedTrackId(null);
    setDragTargetIndex(null);
    dragTargetIndexRef.current = null;
    onStartEditing();
  }

  async function completeEditing() {
    const orderedIds = tracks.map((track) => track.videoId);
    setMutationError(null);
    setRetryMutation(null);
    setPendingMutation("order");

    try {
      if (source === "queue") {
        onCommitTemporaryOrder(orderedIds);
      } else {
        await onCommitPersistedOrder(orderedIds);
      }
      onCompleteEditing();
    } catch {
      setMutationError(REORDER_FAILURE_MESSAGE);
      setRetryMutation({ kind: "order" });
    } finally {
      setPendingMutation(null);
    }
  }

  function getDragTargetIndex(
    videoId: string,
    offsetY: number,
    rowHeight: number
  ): number | null {
    const sourceIndex = tracks.findIndex((track) => track.videoId === videoId);
    if (sourceIndex < 0 || rowHeight <= 0) {
      return sourceIndex < 0 ? null : sourceIndex;
    }

    const requestedIndex = sourceIndex + Math.round(offsetY / rowHeight);
    return Math.max(0, Math.min(tracks.length - 1, requestedIndex));
  }

  function handleTrackDragStart(videoId: string) {
    const sourceIndex = tracks.findIndex((track) => track.videoId === videoId);
    const targetIndex = sourceIndex >= 0 ? sourceIndex : null;
    dragTargetIndexRef.current = targetIndex;
    setDraggedTrackId(videoId);
    setDragTargetIndex(targetIndex);
  }

  function handleTrackDragMove(
    videoId: string,
    offsetY: number,
    rowHeight: number
  ) {
    const targetIndex = getDragTargetIndex(videoId, offsetY, rowHeight);
    dragTargetIndexRef.current = targetIndex;
    setDragTargetIndex((current) => current === targetIndex ? current : targetIndex);
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

  function handleTrackDragEnd(
    videoId: string,
    offsetY: number,
    rowHeight: number
  ) {
    const targetIndex = getDragTargetIndex(videoId, offsetY, rowHeight) ?? dragTargetIndexRef.current;
    updateDraftOrder(videoId, targetIndex);
    dragTargetIndexRef.current = null;
    setDraggedTrackId(null);
    setDragTargetIndex(null);
  }

  function requestDelete(videoId: string) {
    setMutationError(null);
    setRetryMutation(null);
    setConfirmingDeleteId(videoId);
  }

  function cancelDelete() {
    setMutationError(null);
    setRetryMutation(null);
    setConfirmingDeleteId(null);
  }

  async function confirmDelete(videoId: string) {
    setMutationError(null);
    setRetryMutation(null);
    setPendingMutation("delete");

    try {
      if (source === "queue") {
        onDeleteTemporary(videoId);
      } else {
        await onDeletePersisted(videoId);
      }
      setConfirmingDeleteId(null);
      setSelectedDeleteIds((current) => {
        const next = new Set(current);
        next.delete(videoId);
        return next;
      });
    } catch {
      setConfirmingDeleteId(null);
      setMutationError(DELETE_FAILURE_MESSAGE);
      setRetryMutation({ kind: "delete", videoId });
    } finally {
      setPendingMutation(null);
    }
  }

  async function confirmSelectedDeletes(videoIds?: string[]): Promise<void> {
    if (isBusy) {
      return;
    }

    const requestedIds = videoIds ?? Array.from(selectedDeleteIds);
    const selectedIds = tracks
      .filter((track) => requestedIds.includes(track.videoId))
      .map((track) => track.videoId);

    if (selectedIds.length === 0) {
      return;
    }

    setMutationError(null);
    setRetryMutation(null);
    setPendingMutation("delete");
    const completedIds: string[] = [];

    try {
      for (const selectedId of selectedIds) {
        if (source === "queue") {
          onDeleteTemporary(selectedId);
        } else {
          await onDeletePersisted(selectedId);
        }
        completedIds.push(selectedId);
      }
      setSelectedDeleteIds(new Set());
    } catch {
      const remainingIds = selectedIds.filter((selectedId) => !completedIds.includes(selectedId));
      setSelectedDeleteIds(new Set(remainingIds));
      setMutationError(DELETE_FAILURE_MESSAGE);
      setRetryMutation({ kind: "delete-batch", videoIds: remainingIds });
    } finally {
      setPendingMutation(null);
    }
  }

  function retryFailedMutation() {
    if (!retryMutation || isBusy) {
      return;
    }

    if (retryMutation.kind === "order") {
      void completeEditing();
      return;
    }

    if (retryMutation.kind === "delete-batch") {
      void confirmSelectedDeletes(retryMutation.videoIds);
      return;
    }

    void confirmDelete(retryMutation.videoId);
  }

  const emptyMessage = source === "queue"
    ? "Selecione faixas na busca para montar fila."
    : "Esta playlist não tem faixas.";
  const isBusy = pendingMutation !== null;
  const isDeleteMode = !isEditing;
  const draggedSourceIndex = draggedTrackId === null
    ? -1
    : tracks.findIndex((track) => track.videoId === draggedTrackId);
  const dropPlaceholderIndex = draggedSourceIndex >= 0
    && dragTargetIndex !== null
    && dragTargetIndex !== draggedSourceIndex
    ? dragTargetIndex > draggedSourceIndex ? dragTargetIndex + 1 : dragTargetIndex
    : null;

  return (
    <motion.section
      className={styles.managementDrawer}
      aria-label="Gestão da fila"
      animate={drawerControls}
      drag="y"
      dragConstraints={{ bottom: window.innerHeight, top: 0 }}
      dragElastic={0.18}
      dragMomentum={false}
      dragPropagation={false}
      onDragEnd={returnToQueuePreview}
    >
      <header className={styles.managementHeader}>
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
        <button
          type="button"
          className={styles.managementHandle}
          onClick={(event) => event.preventDefault()}
          onKeyDown={(event) => {
            if (event.key === "Enter" || event.key === " ") {
              event.preventDefault();
              onRequestQueuePreview();
            }
          }}
          disabled={isBusy}
          aria-label="Voltar à prévia da fila"
        >
          <span className={styles.grip} aria-hidden="true" />
        </button>
        <span aria-hidden="true" />
      </header>
      <div
        className={styles.managementToolbar}
        onPointerDown={(event) => {
          if (event.target instanceof Element && event.target.closest("button")) {
            event.stopPropagation();
          }
        }}
      >
        <h2>Sua fila</h2>
        {isEditing ? (
          <button
            type="button"
            className={styles.deleteModeButton}
            onClick={() => void completeEditing()}
            disabled={isBusy}
            aria-label="Ativar modo de exclusão"
          >
            Excluir
          </button>
        ) : (
          <button
            type="button"
            className={styles.editButton}
            onClick={beginEditing}
            disabled={isBusy}
            aria-label="Editar fila"
          >
            Editar
          </button>
        )}
      </div>
      {mutationError ? (
        <p className={styles.managementStatus} role="status">
          {mutationError}{" "}
          {retryMutation ? (
            <button
              type="button"
              className={styles.editButton}
              onClick={retryFailedMutation}
              disabled={isBusy}
            >
              Tentar novamente
            </button>
          ) : null}
        </p>
      ) : null}
      <div
        className={styles.managementViewport}
        data-scroll-owner
        onPointerDown={(event) => event.stopPropagation()}
      >
        {tracks.length === 0 ? <p className={styles.emptyQueue}>{emptyMessage}</p> : (
          <ol className={styles.managementList} aria-label="Gerenciar faixas da fila">
            {tracks.map((track, index) => {
              const isConfirmingDelete = confirmingDeleteId === track.videoId;
              const isSelectedForDelete = selectedDeleteIds.has(track.videoId);
              return (
                <Fragment key={track.videoId}>
                  {dropPlaceholderIndex === index ? (
                    <li className={styles.dropPlaceholder} aria-hidden="true" />
                  ) : null}
                  <QueueManagementTrackRow
                    track={track}
                    index={index}
                    isEditing={isEditing}
                    isBusy={isBusy}
                    isConfirmingDelete={isConfirmingDelete}
                    isSelectedForDelete={isSelectedForDelete}
                    onDragStart={() => handleTrackDragStart(track.videoId)}
                    onDragMove={(offsetY, rowHeight) => handleTrackDragMove(track.videoId, offsetY, rowHeight)}
                    onDragEnd={(offsetY, rowHeight) => handleTrackDragEnd(track.videoId, offsetY, rowHeight)}
                    onToggleDeleteSelection={(checked) => {
                      setSelectedDeleteIds((current) => {
                        const next = new Set(current);
                        if (checked) {
                          next.add(track.videoId);
                        } else {
                          next.delete(track.videoId);
                        }
                        return next;
                      });
                    }}
                    onConfirmDelete={() => void confirmDelete(track.videoId)}
                    onCancelDelete={cancelDelete}
                    onRequestDelete={() => requestDelete(track.videoId)}
                  />
                </Fragment>
              );
            })}
            {dropPlaceholderIndex === tracks.length ? (
              <li className={styles.dropPlaceholder} aria-hidden="true" />
            ) : null}
          </ol>
        )}
      </div>
      {isDeleteMode ? (
        <footer className={styles.deleteFooter}>
          <button
            type="button"
            className={styles.continueDeleteButton}
            onClick={() => void confirmSelectedDeletes()}
            disabled={isBusy || confirmingDeleteId !== null || selectedDeleteIds.size === 0}
            aria-label="Continuar e confirmar exclusão"
          >
            Continuar
          </button>
        </footer>
      ) : null}
    </motion.section>
  );
}

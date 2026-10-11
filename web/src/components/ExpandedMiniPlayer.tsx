import { createContext, useContext, useEffect, useRef, useState } from "react";
import type { KeyboardEvent, PointerEvent as ReactPointerEvent, PropsWithChildren, RefObject, WheelEvent as ReactWheelEvent } from "react";
import { motion, useAnimationControls, useDragControls, useReducedMotion } from "framer-motion";
import type { PanInfo } from "framer-motion";
import { reorderTrackSelection } from "../state/cart";
import type { TrackSnapshot } from "../types";
import { PlayerSummaryPane } from "./PlayerSummaryPane";
import { QueueManagementPane } from "./QueueManagementPane";
import { QueuePreviewPane } from "./QueuePreviewPane";
import styles from "./ExpandedMiniPlayer.module.css";

export type QueueMutationCallbacks = {
  onCommitTemporaryOrder?: (orderedIds: string[]) => void;
  onCommitPersistedOrder?: (orderedIds: string[]) => Promise<void>;
  onDeleteTemporary?: (videoId: string) => void;
  onDeletePersisted?: (videoId: string) => Promise<void>;
};

export type ExpandedMiniPlayerOwner = QueueMutationCallbacks & {
  tracks: TrackSnapshot[];
};

const ExpandedMiniPlayerOwnerContext = createContext<ExpandedMiniPlayerOwner | null>(null);

export function ExpandedMiniPlayerOwnerProvider({
  children,
  value
}: PropsWithChildren<{ value: ExpandedMiniPlayerOwner | null }>) {
  return (
    <ExpandedMiniPlayerOwnerContext.Provider value={value}>
      {children}
    </ExpandedMiniPlayerOwnerContext.Provider>
  );
}

type ExpandedMiniPlayerProps = QueueMutationCallbacks & {
  open: boolean;
  artworkTrack: TrackSnapshot | null;
  title: string;
  ownerName?: string;
  trackCount: number;
  durationSeconds: number;
  source: "queue" | "playlist";
  tracks: TrackSnapshot[];
  returnFocusRef: RefObject<HTMLButtonElement | null>;
  onRequestClose: () => void;
};

type Pane = "summary" | "queue" | "management";
const PANE_TRAVEL_RATIO = 0.55;
const FLING_VELOCITY = 500;
const STANDARD_EASE = [0.2, 0, 0, 1] as const;

function getFocusableElements(container: HTMLElement): HTMLElement[] {
  return Array.from(
    container.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
    )
  );
}

export function ExpandedMiniPlayer({
  open,
  artworkTrack,
  title,
  ownerName,
  trackCount,
  durationSeconds,
  source,
  tracks,
  returnFocusRef,
  onRequestClose,
  onCommitTemporaryOrder,
  onCommitPersistedOrder,
  onDeleteTemporary,
  onDeletePersisted
}: ExpandedMiniPlayerProps) {
  const owner = useContext(ExpandedMiniPlayerOwnerContext);
  const currentTracks = owner?.tracks ?? tracks;
  const commitTemporaryOrderCallback = onCommitTemporaryOrder ?? owner?.onCommitTemporaryOrder;
  const commitPersistedOrderCallback = onCommitPersistedOrder ?? owner?.onCommitPersistedOrder;
  const deleteTemporaryCallback = onDeleteTemporary ?? owner?.onDeleteTemporary;
  const deletePersistedCallback = onDeletePersisted ?? owner?.onDeletePersisted;
  const panelRef = useRef<HTMLElement>(null);
  const shouldRestoreFocusRef = useRef(false);
  const wasOpenRef = useRef(false);
  const [pane, setPane] = useState<Pane>("summary");
  const [isEditingQueue, setIsEditingQueue] = useState(true);
  const [draftTracks, setDraftTracks] = useState<TrackSnapshot[]>(currentTracks);
  const controls = useAnimationControls();
  const dragControls = useDragControls();
  const reduceMotion = useReducedMotion();

  useEffect(() => {
    if (!open) {
      wasOpenRef.current = false;
      if (shouldRestoreFocusRef.current) {
        returnFocusRef.current?.focus();
        shouldRestoreFocusRef.current = false;
      }
      return;
    }

    if (!wasOpenRef.current) {
      setPane("summary");
      setIsEditingQueue(true);
      setDraftTracks(currentTracks);
    }
    wasOpenRef.current = true;
    controls.set({ opacity: 1, y: 0 });

    const documentOverflow = {
      body: document.body.style.overflow,
      documentElement: document.documentElement.style.overflow
    };
    document.body.style.overflow = "hidden";
    document.documentElement.style.overflow = "hidden";

    const background = Array.from(
      document.querySelectorAll<HTMLElement>(
        ".shell-tab-panel, [data-mini-player-closed='true'], nav[aria-label='Navegação principal']"
      )
    );
    const previousState = background.map((element) => ({
      element,
      ariaHidden: element.getAttribute("aria-hidden"),
      inert: element.inert,
      pointerEvents: element.style.pointerEvents
    }));

    background.forEach((element) => {
      element.setAttribute("aria-hidden", "true");
      element.inert = true;
      element.style.pointerEvents = "none";
    });

    const focusFrame = window.requestAnimationFrame(() => {
      getFocusableElements(panelRef.current ?? document.body)[0]?.focus();
    });

    return () => {
      window.cancelAnimationFrame(focusFrame);
      document.body.style.overflow = documentOverflow.body;
      document.documentElement.style.overflow = documentOverflow.documentElement;
      previousState.forEach(({ element, ariaHidden, inert, pointerEvents }) => {
        if (ariaHidden === null) {
          element.removeAttribute("aria-hidden");
        } else {
          element.setAttribute("aria-hidden", ariaHidden);
        }
        element.inert = inert;
        element.style.pointerEvents = pointerEvents;
      });
    };
  }, [controls, currentTracks, open, returnFocusRef]);

  if (!open) {
    return null;
  }

  const viewportHeight = panelRef.current?.getBoundingClientRect().height ?? window.innerHeight;
  const transition = { duration: 0.26, ease: STANDARD_EASE };
  const previewTracks = pane === "queue" || isEditingQueue ? draftTracks : currentTracks;

  function startPaneDrag(event: ReactPointerEvent<HTMLElement>) {
    dragControls.start(event, { snapToCursor: false });
  }

  function handleSummaryWheel(event: ReactWheelEvent<HTMLElement>) {
    if (event.deltaY === 0) {
      return;
    }

    event.preventDefault();
    if (event.deltaY > 0) {
      closePlayer();
      return;
    }

    navigateTo("queue", -1);
  }

  function closePlayer() {
    shouldRestoreFocusRef.current = true;
    if (reduceMotion) {
      onRequestClose();
      return;
    }

    void controls.start({ y: viewportHeight, opacity: 0, transition: { duration: 0.32, ease: STANDARD_EASE } })
      .then(onRequestClose);
  }

  function resetDrag() {
    void controls.start({
      y: 0,
      opacity: 1,
      transition: { type: "spring", duration: 0.26, bounce: 0 }
    });
  }

  function navigateTo(nextPane: Pane, direction: 1 | -1) {
    if (nextPane === "queue" && !isEditingQueue) {
      setDraftTracks(currentTracks);
      setIsEditingQueue(true);
    }

    if (reduceMotion) {
      setPane(nextPane);
      controls.set({ opacity: 1, y: 0 });
      return;
    }

    void controls.start({ y: direction * viewportHeight, opacity: 1, transition })
      .then(() => {
        setPane(nextPane);
        controls.set({ y: -direction * viewportHeight, opacity: 1 });
        return controls.start({ y: 0, opacity: 1, transition });
      });
  }

  function startQueueEditing() {
    setDraftTracks(currentTracks);
    setIsEditingQueue(true);
  }

  function completeQueueEditing() {
    setIsEditingQueue(false);
  }

  function commitTemporaryOrder(orderedIds: string[]) {
    const orderedTracks = reorderTrackSelection(currentTracks, orderedIds);
    if (!orderedTracks || !commitTemporaryOrderCallback) {
      throw new Error("Temporary queue order could not be committed.");
    }

    commitTemporaryOrderCallback(orderedTracks.map((track) => track.videoId));
  }

  async function commitPersistedOrder(orderedIds: string[]): Promise<void> {
    if (!commitPersistedOrderCallback) {
      throw new Error("Persisted playlist order callback is unavailable.");
    }

    await commitPersistedOrderCallback(orderedIds);
  }

  function deleteTemporary(videoId: string) {
    if (!deleteTemporaryCallback) {
      throw new Error("Temporary queue delete callback is unavailable.");
    }

    deleteTemporaryCallback(videoId);
  }

  async function deletePersisted(videoId: string): Promise<void> {
    if (!deletePersistedCallback) {
      throw new Error("Persisted playlist delete callback is unavailable.");
    }

    await deletePersistedCallback(videoId);
  }

  function handleDragEnd(_: MouseEvent | TouchEvent | globalThis.PointerEvent, info: PanInfo) {
    const exceededDistance = Math.abs(info.offset.y) > viewportHeight * PANE_TRAVEL_RATIO;
    const exceededVelocity = Math.abs(info.velocity.y) > FLING_VELOCITY;
    const direction = exceededDistance ? Math.sign(info.offset.y) : Math.sign(info.velocity.y);

    if ((!exceededDistance && !exceededVelocity) || direction === 0) {
      resetDrag();
      return;
    }

    if (pane === "summary") {
      if (direction > 0) {
        closePlayer();
      } else {
        navigateTo("queue", -1);
      }
      return;
    }

    if (pane === "queue" && direction > 0) {
      navigateTo("summary", 1);
      return;
    }

    if (pane === "queue" && direction < 0) {
      navigateTo("management", -1);
      return;
    }

    if (pane === "management" && direction > 0) {
      navigateTo("queue", 1);
      return;
    }

    resetDrag();
  }

  function handleKeyDown(event: KeyboardEvent<HTMLElement>) {
    if (event.key === "Escape") {
      event.preventDefault();
      closePlayer();
      return;
    }

    if (event.key !== "Tab") return;

    const focusable = getFocusableElements(panelRef.current ?? document.body);
    if (focusable.length === 0) {
      event.preventDefault();
      return;
    }

    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  return (
    <div className={styles.root} role="dialog" aria-modal="true" aria-label="Mini-player expandido">
      <motion.section
        ref={panelRef}
        animate={controls}
        className={styles.dragSurface}
        drag="y"
        dragConstraints={{ bottom: viewportHeight, top: -viewportHeight }}
        dragControls={dragControls}
        dragElastic={0.18}
        dragListener={false}
        dragMomentum={false}
        onDragEnd={handleDragEnd}
        onKeyDown={handleKeyDown}
        tabIndex={-1}
      >
        {pane === "summary" ? (
          <PlayerSummaryPane
            artworkTrack={artworkTrack}
            durationSeconds={durationSeconds}
            source={source}
            title={title}
            ownerName={ownerName}
            trackCount={trackCount}
            onPanePointerDown={startPaneDrag}
            onPaneWheel={handleSummaryWheel}
            onQueueHandlePointerDown={startPaneDrag}
            onQueueHandleClick={() => navigateTo("queue", -1)}
            onRequestClose={closePlayer}
          />
        ) : pane === "queue" ? (
          <QueuePreviewPane
            artworkTrack={artworkTrack}
            ownerName={ownerName}
            source={source}
            title={title}
            trackCount={trackCount}
            tracks={previewTracks}
            onPanePointerDown={startPaneDrag}
            onQueueHandlePointerDown={startPaneDrag}
            onQueueHandleClick={() => navigateTo("management", -1)}
            onRequestClose={closePlayer}
            onDraftTracksChange={setDraftTracks}
          />
        ) : (
          <div className={styles.managementStage}>
            <QueueManagementPane
              source={source}
              tracks={previewTracks}
              isEditing={isEditingQueue}
              onRequestQueuePreview={() => navigateTo("queue", 1)}
              onRequestClose={closePlayer}
              onStartEditing={startQueueEditing}
              onDraftTracksChange={setDraftTracks}
              onCompleteEditing={completeQueueEditing}
              onCommitTemporaryOrder={commitTemporaryOrder}
              onCommitPersistedOrder={commitPersistedOrder}
              onDeleteTemporary={deleteTemporary}
              onDeletePersisted={deletePersisted}
            />
          </div>
        )}
      </motion.section>
    </div>
  );
}

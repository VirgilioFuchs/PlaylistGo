import type { CSSProperties, ReactNode } from "react";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  addTrackToPlaylist,
  createPlaylist,
  listPlaylists,
  removeTrackFromPlaylist,
  reorderPlaylistTracks,
  searchTracks
} from "./api/client";
import { ExpandedMiniPlayerOwnerProvider } from "./components/ExpandedMiniPlayer";
import { BottomNav } from "./components/BottomNav";
import { FinalizeCart } from "./components/FinalizeCart";
import { HomeHub } from "./components/HomeHub";
import { MiniPlayer } from "./components/MiniPlayer";
import type { MiniPlayerLifecycle } from "./components/MiniPlayer";
import { PlaylistPanel } from "./components/PlaylistPanel";
import { SearchTab } from "./components/SearchTab";
import type { SearchTabMessage, SearchTabState } from "./components/SearchTab";
import { StatusBanner } from "./components/StatusBanner";
import {
  loadCartSelection,
  normalizeCartTracks,
  removeTrackFromSelection,
  reorderTrackSelection,
  saveCartSelection,
  toggleTrackSelection
} from "./state/cart";
import type { ApiError, Playlist, SearchCacheStatus, SearchQuota, TrackSnapshot } from "./types";

type AppTab = "home" | "search" | "playlists";
type FinalizeTarget = { kind: "new"; name: string } | { kind: "existing"; playlistId: string };
type MiniPlayerSource = { kind: "queue" } | { kind: "playlist"; playlist: Playlist };
type CartState = {
  tracks: TrackSnapshot[];
  source: MiniPlayerSource | null;
};
type MiniPlayerPresentation = {
  source: MiniPlayerSource["kind"];
  artworkTrack: TrackSnapshot | null;
  title: string;
  trackCount: number;
  durationSeconds: number;
};
type FinalizationRecovery = {
  target: FinalizeTarget;
  destination: Playlist | null;
  confirmedVideoIds: string[];
  requiresReconciliation: boolean;
  blockedVideoId?: string;
};
type TabContentProps = {
  activeTab: AppTab;
  home: ReactNode;
  search: ReactNode;
  playlists: ReactNode;
};

const SEARCH_HISTORY_STORAGE_KEY = "musicgo.search-history.v1";
const SEARCH_HISTORY_LIMIT = 6;
const FIRST_PLAYLIST_SUCCESS_STORAGE_KEY = "musicgo.first-playlist-success.v1";
const idleMiniPlayerLifecycle: MiniPlayerLifecycle = {
  phase: "idle",
  confirmedTrackCount: 0,
  totalTrackCount: 0
};
let firstPlaylistSuccessRecordedInMemory = false;

const initialSearchMessage: SearchTabMessage = {
  title: "Pronto",
  message: "Use a busca para encontrar faixas e montar sua playlist.",
  variant: "info"
};

function getErrorCode(error: unknown): string | undefined {
  if (error && typeof error === "object" && "code" in error) {
    const candidate = error as { code?: unknown };
    return typeof candidate.code === "string" ? candidate.code : undefined;
  }
  return undefined;
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }

  if (error && typeof error === "object" && "message" in error) {
    const candidate = error as { message?: unknown };
    if (typeof candidate.message === "string" && candidate.message.trim()) {
      return candidate.message;
    }
  }

  return "Something went wrong.";
}

function isApiError(error: unknown): error is ApiError {
  return Boolean(error && typeof error === "object" && "status" in error && "code" in error);
}

function loadSearchHistory(): string[] {
  if (typeof window === "undefined") {
    return [];
  }

  try {
    const raw = window.localStorage.getItem(SEARCH_HISTORY_STORAGE_KEY);
    if (!raw) {
      return [];
    }

    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) {
      return [];
    }

    return parsed.filter((item): item is string => typeof item === "string" && item.trim().length > 0).slice(0, SEARCH_HISTORY_LIMIT);
  } catch {
    return [];
  }
}

function saveSearchHistory(entries: string[]): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(SEARCH_HISTORY_STORAGE_KEY, JSON.stringify(entries.slice(0, SEARCH_HISTORY_LIMIT)));
  } catch {
    // Ignore storage failures. Search history remains in memory.
  }
}

function mergeSearchHistory(entries: string[], query: string): string[] {
  const normalized = query.trim();
  if (!normalized) {
    return entries;
  }

  return [normalized, ...entries.filter((entry) => entry.toLowerCase() !== normalized.toLowerCase())].slice(0, SEARCH_HISTORY_LIMIT);
}

function ShellTabContent({ activeTab, home, search, playlists }: TabContentProps) {
  const panels: Array<{ id: AppTab; label: string; content: ReactNode }> = [
    { id: "home", label: "Home", content: home },
    { id: "search", label: "Busca", content: search },
    { id: "playlists", label: "Biblioteca de playlists", content: playlists }
  ];
  return panels.map(({ id, label, content }) => {
    const isActive = id === activeTab;
    return (
      <section
        key={id}
        className="shell-tab-content"
        aria-label={label}
        hidden={!isActive}
        inert={!isActive || undefined}
      >
        {content}
      </section>
    );
  });
}

function isSameFinalizeTarget(left: FinalizeTarget, right: FinalizeTarget): boolean {
  if (left.kind !== right.kind) {
    return false;
  }

  if (left.kind === "existing" && right.kind === "existing") {
    return left.playlistId === right.playlistId;
  }

  return left.kind === "new" && right.kind === "new" && left.name.trim() === right.name.trim();
}

function createInitialCartState(): CartState {
  const tracks = normalizeCartTracks(loadCartSelection());
  return {
    tracks,
    source: tracks.length > 0 ? { kind: "queue" } : null
  };
}

function createQueueMiniPlayerPresentation(tracks: TrackSnapshot[]): MiniPlayerPresentation | null {
  const leadTrack = tracks[0] ?? null;
  if (!leadTrack) {
    return null;
  }

  return {
    source: "queue",
    artworkTrack: leadTrack,
    title: leadTrack.title,
    trackCount: tracks.length,
    durationSeconds: tracks.reduce((total, track) => total + Math.max(0, track.durationSeconds), 0)
  };
}

function prefersReducedMotion(): boolean {
  return typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

function recordFirstPlaylistSuccess(): boolean {
  if (firstPlaylistSuccessRecordedInMemory) {
    return false;
  }

  if (typeof window !== "undefined") {
    try {
      if (window.localStorage.getItem(FIRST_PLAYLIST_SUCCESS_STORAGE_KEY) === "1") {
        firstPlaylistSuccessRecordedInMemory = true;
        return false;
      }
      window.localStorage.setItem(FIRST_PLAYLIST_SUCCESS_STORAGE_KEY, "1");
    } catch {
      // Storage may be unavailable. The in-memory fallback still keeps this session idempotent.
    }
  }

  firstPlaylistSuccessRecordedInMemory = true;
  return true;
}

export default function App() {
  const [activeTab, setActiveTab] = useState<AppTab>("home");
  const [query, setQuery] = useState("");
  const [searchState, setSearchState] = useState<SearchTabState>("idle");
  const [searchResults, setSearchResults] = useState<TrackSnapshot[]>([]);
  const [searchQuota, setSearchQuota] = useState<SearchQuota | null>(null);
  const [searchCache, setSearchCache] = useState<SearchCacheStatus | null>(null);
  const [searchMessage, setSearchMessage] = useState<SearchTabMessage>(initialSearchMessage);
  const [lastQuery, setLastQuery] = useState("");
  const [searchHistory, setSearchHistory] = useState<string[]>(() => loadSearchHistory());
  const [quotaAcknowledged, setQuotaAcknowledged] = useState(false);
  const [quotaDetailsOpen, setQuotaDetailsOpen] = useState(false);

  const [playlists, setPlaylists] = useState<Playlist[]>([]);
  const [playlistsLoading, setPlaylistsLoading] = useState(true);
  const [playlistBusy, setPlaylistBusy] = useState(false);
  const [playlistError, setPlaylistError] = useState<string | null>(null);
  const playlistMutationVersionRef = useRef(0);
  const playlistRefreshRequestRef = useRef(0);

  const [cartState, setCartState] = useState<CartState>(createInitialCartState);
  const selectedTracks = cartState.tracks;
  const miniPlayerSource = cartState.source;
  const [finalizeOpen, setFinalizeOpen] = useState(false);
  const [finalizeBusy, setFinalizeBusy] = useState(false);
  const [finalizationRecovery, setFinalizationRecovery] = useState<FinalizationRecovery | null>(null);
  const [finalizationLifecycle, setFinalizationLifecycle] = useState<MiniPlayerLifecycle>(idleMiniPlayerLifecycle);
  const [exitingMiniPlayerPresentation, setExitingMiniPlayerPresentation] = useState<MiniPlayerPresentation | null>(null);
  const [successState, setSuccessState] = useState<{ playlist: Playlist } | null>(null);
  const [expandedSource, setExpandedSource] = useState<MiniPlayerSource | null>(null);

  const selectedIds = useMemo(() => new Set(selectedTracks.map((track) => track.videoId)), [selectedTracks]);
  const hasSelection = selectedTracks.length > 0;
  const miniPlayerPresentation = useMemo<MiniPlayerPresentation | null>(() => {
    if (!miniPlayerSource) {
      return null;
    }

    if (miniPlayerSource.kind === "playlist") {
      const { playlist } = miniPlayerSource;
      return {
        source: "playlist",
        artworkTrack: playlist.tracks?.[0] ?? null,
        title: playlist.name,
        trackCount: playlist.trackCount,
        durationSeconds: playlist.durationSeconds
      };
    }

    return createQueueMiniPlayerPresentation(selectedTracks);
  }, [miniPlayerSource, selectedTracks]);
  const renderedMiniPlayerPresentation = miniPlayerPresentation ?? exitingMiniPlayerPresentation;
  const hasActiveMiniPlayer = renderedMiniPlayerPresentation !== null;
  const expandedTracks = useMemo<TrackSnapshot[]>(() => {
    if (!expandedSource) {
      return [];
    }

    return expandedSource.kind === "playlist" ? expandedSource.playlist.tracks ?? [] : selectedTracks;
  }, [expandedSource, selectedTracks]);
  const shellStyle = {
    "--content-safe-bottom": hasActiveMiniPlayer
      ? "calc(var(--bottom-nav-height) + var(--player-bar-height) + env(safe-area-inset-bottom))"
      : "calc(var(--bottom-nav-height) + env(safe-area-inset-bottom))"
  } as CSSProperties;

  function handleTabChange(tab: AppTab) {
    setActiveTab(tab);
  }

  function handleOpenPlaylist(playlist: Playlist) {
    setCartState((current) => ({
      ...current,
      source: { kind: "playlist", playlist }
    }));
    setExitingMiniPlayerPresentation(null);
    setFinalizationLifecycle(idleMiniPlayerLifecycle);
    setExpandedSource(null);
  }

  function handleResumeQueue() {
    if (selectedTracks.length === 0) {
      return;
    }

    setCartState((current) => ({ ...current, source: { kind: "queue" } }));
    setExitingMiniPlayerPresentation(null);
    setFinalizationLifecycle(idleMiniPlayerLifecycle);
    setActiveTab("search");
  }

  function handleMiniPlayerExpansionIntent() {
    if (miniPlayerSource) {
      setExpandedSource(miniPlayerSource);
    }
  }

  function handleMiniPlayerCollapse() {
    setCartState((current) => {
      if (current.source || !expandedSource) {
        return current;
      }

      if (expandedSource.kind === "queue" && current.tracks.length === 0) {
        return current;
      }

      return { ...current, source: expandedSource };
    });
    setExpandedSource(null);
  }

  useEffect(() => {
    saveCartSelection(selectedTracks);
  }, [selectedTracks]);

  useEffect(() => {
    saveSearchHistory(searchHistory);
  }, [searchHistory]);

  useEffect(() => {
    void refreshPlaylists();
  }, []);

  async function refreshPlaylists() {
    const requestId = playlistRefreshRequestRef.current + 1;
    playlistRefreshRequestRef.current = requestId;
    const mutationVersion = playlistMutationVersionRef.current;
    setPlaylistsLoading(true);
    setPlaylistError(null);
    try {
      const response = await listPlaylists("recent");
      if (requestId === playlistRefreshRequestRef.current && mutationVersion === playlistMutationVersionRef.current) {
        setPlaylists(response.data);
      }
    } catch (error) {
      if (requestId === playlistRefreshRequestRef.current) {
        setPlaylistError(getErrorMessage(error));
      }
    } finally {
      if (requestId === playlistRefreshRequestRef.current) {
        setPlaylistsLoading(false);
      }
    }
  }

  function upsertPlaylist(playlist: Playlist) {
    playlistMutationVersionRef.current += 1;
    setPlaylists((current) => [playlist, ...current.filter((item) => item.id !== playlist.id)]);
  }

  function syncCanonicalPlaylist(playlist: Playlist) {
    upsertPlaylist(playlist);
    setCartState((current) => ({
      ...current,
      source:
        current.source?.kind === "playlist" && current.source.playlist.id === playlist.id
          ? { kind: "playlist", playlist }
          : current.source
    }));
    setExpandedSource((current) =>
      current?.kind === "playlist" && current.playlist.id === playlist.id
        ? { kind: "playlist", playlist }
        : current
    );
  }

  function handleCommitTemporaryOrder(orderedIds: string[]) {
    const orderedTracks = reorderTrackSelection(selectedTracks, orderedIds);
    if (!orderedTracks) {
      throw new Error("Temporary queue order is invalid.");
    }

    setCartState((current) => ({
      ...current,
      tracks: orderedTracks
    }));
  }

  async function handleCommitPersistedOrder(orderedIds: string[]): Promise<void> {
    if (expandedSource?.kind !== "playlist") {
      throw new Error("Persisted playlist source is unavailable.");
    }

    const updated = await reorderPlaylistTracks(
      expandedSource.playlist.id,
      orderedIds,
      expandedSource.playlist.revision
    );
    if (!Array.isArray(updated.tracks)) {
      throw new Error("Canonical playlist response is missing tracks.");
    }

    syncCanonicalPlaylist(updated);
  }

  function handleDeleteTemporaryFromExpanded(videoId: string) {
    const isRemovingLastQueueTrack =
      expandedSource?.kind === "queue" &&
      selectedTracks.length === 1 &&
      selectedTracks[0]?.videoId === videoId;

    handleRemoveSelected(videoId);
    if (isRemovingLastQueueTrack) {
      setExpandedSource(null);
    }
  }

  async function handleDeletePersistedFromExpanded(videoId: string): Promise<void> {
    if (expandedSource?.kind !== "playlist") {
      throw new Error("Persisted playlist source is unavailable.");
    }

    setPlaylistBusy(true);
    setPlaylistError(null);
    try {
      const updated = await removeTrackFromPlaylist(expandedSource.playlist.id, videoId);
      syncCanonicalPlaylist(updated);
    } catch (error) {
      setPlaylistError(getErrorMessage(error));
      throw error;
    } finally {
      setPlaylistBusy(false);
    }
  }

  async function handleSearch(rawQuery: string) {
    const normalized = rawQuery.trim();
    if (!normalized) {
      setSearchState("validation_error");
      setSearchMessage({
        title: "Consulta obrigatoria",
        message: "Digite uma faixa, artista ou video antes de pesquisar.",
        variant: "warning"
      });
      setSearchResults([]);
      setActiveTab("search");
      return;
    }

    setQuery(normalized);
    setLastQuery(normalized);
    setSearchHistory((current) => mergeSearchHistory(current, normalized));
    setActiveTab("search");
    setSearchState("loading");
    setSearchMessage({
      title: "Pesquisando",
      message: `Buscando resultados para "${normalized}".`,
      variant: "info"
    });

    try {
      const response = await searchTracks(normalized);
      setSearchQuota(response.quota);
      setSearchCache(response.cache);
      setSearchResults(response.data);

      if (response.data.length === 0) {
        setSearchState("empty");
        setSearchMessage({
          title: "Sem resultados",
          message: "Nenhuma faixa correspondeu a essa busca.",
          variant: "info"
        });
        return;
      }

      setSearchState("success");
      setSearchMessage({
        title: response.cache.hit ? "Cache local" : "Busca concluida",
        message: response.cache.hit ? "Os resultados vieram do cache local." : "Os resultados vieram do backend.",
        variant: "success"
      });
    } catch (error) {
      const code = isApiError(error) ? error.code : getErrorCode(error);
      const message = getErrorMessage(error);
      setSearchResults([]);
      setSearchCache(null);

      if (code === "quota_limited") {
        setSearchState("quota_limited");
        setQuotaAcknowledged(false);
        setQuotaDetailsOpen(false);
        setSearchQuota(isApiError(error) ? error.quota ?? null : null);
        setSearchMessage({
          title: "Limite de busca atingido",
          message,
          variant: "warning"
        });
        return;
      }

      if (code === "validation_error") {
        setSearchState("validation_error");
        setSearchMessage({
          title: "Consulta obrigatoria",
          message,
          variant: "warning"
        });
        return;
      }

      setSearchState("temporary_error");
      setSearchMessage({
        title: "Falha na busca",
        message,
        variant: "error"
      });
    }
  }

  function handleHistorySearch(entry: string) {
    void handleSearch(entry);
  }

  function handleClearSearchHistory() {
    setSearchHistory([]);
  }

  function handleRetry() {
    void handleSearch(lastQuery);
  }

  function handleToggleTrack(track: TrackSnapshot) {
    if (selectedIds.has(track.videoId)) {
      handleRemoveSelected(track.videoId);
      return;
    }

    setExitingMiniPlayerPresentation(null);
    setFinalizationLifecycle(idleMiniPlayerLifecycle);
    setCartState((current) => {
      const tracks = toggleTrackSelection(current.tracks, track);
      return {
        tracks,
        source: { kind: "queue" }
      };
    });
  }

  function handleRemoveSelected(videoId: string) {
    resolveBlockedFinalizationTrack(videoId);
    const isRemovingLastQueueTrack =
      miniPlayerSource?.kind === "queue" &&
      selectedTracks.length === 1 &&
      selectedTracks[0]?.videoId === videoId;

    if (isRemovingLastQueueTrack && !prefersReducedMotion()) {
      setExitingMiniPlayerPresentation(createQueueMiniPlayerPresentation(selectedTracks));
    } else if (isRemovingLastQueueTrack) {
      setExitingMiniPlayerPresentation(null);
    }
    setFinalizationLifecycle(idleMiniPlayerLifecycle);
    setCartState((current) => {
      const tracks = removeTrackFromSelection(current.tracks, videoId);
      return {
        tracks,
        source: current.source?.kind === "queue" && tracks.length === 0 ? null : current.source
      };
    });
  }

  function resolveBlockedFinalizationTrack(videoId: string) {
    setFinalizationRecovery((current) =>
      current?.blockedVideoId === videoId
        ? { ...current, requiresReconciliation: false, blockedVideoId: undefined }
        : current
    );
  }

  async function handleCreatePlaylist(name: string) {
    const trimmed = name.trim();
    if (!trimmed) {
      setPlaylistError("Playlist name is required.");
      return;
    }

    setPlaylistBusy(true);
    setPlaylistError(null);
    try {
      const created = await createPlaylist(trimmed);
      upsertPlaylist(created);
    } catch (error) {
      setPlaylistError(getErrorMessage(error));
    } finally {
      setPlaylistBusy(false);
    }
  }

  async function handleRemoveTrackFromPlaylist(playlistId: string, videoId: string) {
    setPlaylistBusy(true);
    setPlaylistError(null);
    try {
      const updated = await removeTrackFromPlaylist(playlistId, videoId);
      syncCanonicalPlaylist(updated);
    } catch (error) {
      setPlaylistError(getErrorMessage(error));
    } finally {
      setPlaylistBusy(false);
    }
  }

  async function handleFinalize(target: FinalizeTarget) {
    if (selectedTracks.length === 0 || finalizeBusy) {
      return;
    }

    const matchingRecovery =
      finalizationRecovery && isSameFinalizeTarget(finalizationRecovery.target, target)
        ? finalizationRecovery
        : null;

    if (
      finalizationRecovery &&
      !matchingRecovery &&
      (finalizationRecovery.destination || finalizationRecovery.confirmedVideoIds.length > 0)
    ) {
      setPlaylistError("Há uma finalização parcial em andamento. Retome ou reconcilie o destino anterior antes de escolher outra playlist.");
      return;
    }

    if (matchingRecovery?.requiresReconciliation) {
      setPlaylistError("A faixa em conflito pode já estar na playlist. Confira-a e remova-a da fila para retomar a finalização das demais faixas.");
      return;
    }

    const queueSnapshot = [...selectedTracks];
    const finalizedVideoIds = new Set(queueSnapshot.map((track) => track.videoId));
    let recovery: FinalizationRecovery = matchingRecovery ?? {
      target,
      destination: null,
      confirmedVideoIds: [],
      requiresReconciliation: false
    };

    const totalTrackCount = queueSnapshot.length;
    setExitingMiniPlayerPresentation(null);
    setFinalizationLifecycle({
      phase: !recovery.destination && target.kind === "new" ? "preparing" : "processing",
      confirmedTrackCount: recovery.confirmedVideoIds.length,
      totalTrackCount
    });
    setFinalizeBusy(true);
    setPlaylistError(null);
    try {
      let destination = recovery.destination;
      if (!destination && target.kind === "new") {
        destination = await createPlaylist(target.name);
        upsertPlaylist(destination);
        recovery = { ...recovery, destination };
        setFinalizationRecovery(recovery);
        setFinalizationLifecycle({
          phase: "processing",
          confirmedTrackCount: recovery.confirmedVideoIds.length,
          totalTrackCount
        });
      }

      const targetPlaylistId = destination?.id ?? (target.kind === "existing" ? target.playlistId : "");
      if (!targetPlaylistId) {
        throw new Error("Não foi possível confirmar a playlist de destino.");
      }
      const confirmedVideoIds = new Set(recovery.confirmedVideoIds);
      const pendingTracks = queueSnapshot.filter((track) => !confirmedVideoIds.has(track.videoId));
      for (const [index, track] of pendingTracks.entries()) {
        const isLastPendingTrack = index === pendingTracks.length - 1;
        if (isLastPendingTrack) {
          setFinalizationLifecycle({
            phase: "finalizing",
            confirmedTrackCount: confirmedVideoIds.size,
            totalTrackCount
          });
        }

        try {
          const updated = await addTrackToPlaylist(targetPlaylistId, track);
          destination = updated;
          confirmedVideoIds.add(track.videoId);
          recovery = {
            ...recovery,
            destination: updated,
            confirmedVideoIds: [...confirmedVideoIds]
          };
          upsertPlaylist(updated);
          setFinalizationRecovery(recovery);
          setFinalizationLifecycle({
            phase: isLastPendingTrack ? "completed" : "processing",
            confirmedTrackCount: confirmedVideoIds.size,
            totalTrackCount
          });
        } catch (error) {
          if (getErrorCode(error) === "conflict") {
            recovery = {
              ...recovery,
              confirmedVideoIds: [...confirmedVideoIds],
              requiresReconciliation: true,
              blockedVideoId: track.videoId
            };
            setFinalizationRecovery(recovery);
          }
          throw error;
        }
      }

      if (!destination) {
        throw new Error("Não foi possível confirmar a playlist de destino.");
      }

      if (pendingTracks.length === 0) {
        setFinalizationLifecycle({
          phase: "completed",
          confirmedTrackCount: confirmedVideoIds.size,
          totalTrackCount
        });
      }

      setCartState((current) => ({
        tracks: current.tracks.filter((track) => !finalizedVideoIds.has(track.videoId)),
        source: { kind: "playlist", playlist: destination }
      }));
      setFinalizationRecovery(null);
      setFinalizeOpen(false);
      setExpandedSource({ kind: "playlist", playlist: destination });
      if (target.kind === "new" && recordFirstPlaylistSuccess()) {
        setSuccessState({ playlist: destination });
      }
    } catch (error) {
      setFinalizationLifecycle({
        phase: "failed",
        confirmedTrackCount: recovery.confirmedVideoIds.length,
        totalTrackCount
      });
      if (getErrorCode(error) === "conflict") {
        setPlaylistError("Há uma faixa duplicada ou uma confirmação incerta. A fila foi preservada; confira a faixa em conflito e remova-a da fila antes de concluir.");
      } else {
        setPlaylistError(getErrorMessage(error));
      }
    } finally {
      setFinalizeBusy(false);
    }
  }

  const searchContent = (
    <SearchTab
      query={query}
      lastQuery={lastQuery}
      state={searchState}
      message={searchMessage}
      tracks={searchResults}
      selectedIds={selectedIds}
      history={searchHistory}
      locked={finalizeBusy}
      quota={searchQuota}
      cache={searchCache}
      quotaAcknowledged={quotaAcknowledged}
      quotaDetailsOpen={quotaDetailsOpen}
      onQueryChange={setQuery}
      onSearch={handleSearch}
      onRetry={handleRetry}
      onHistorySearch={handleHistorySearch}
      onClearHistory={handleClearSearchHistory}
      onToggleTrack={handleToggleTrack}
      onAcknowledgeQuota={() => setQuotaAcknowledged(true)}
      onToggleQuotaDetails={() => setQuotaDetailsOpen((current) => !current)}
    />
  );

  const playlistsContent = (
    <section className="shell-main" aria-label="Biblioteca de playlists">
      <section className="shell-band shell-band--muted">
        <div className="shell-band__copy">
          <p className="shell-band__eyebrow">Playlists</p>
          <h2>Biblioteca em grid, com clima de colecao tocavel e leitura rapida.</h2>
          <p className="shell-band__lede">Playlists ficam em cards com capa, contexto e preview, nao em bloco administrativo.</p>
        </div>
      </section>

      <PlaylistPanel
        playlists={playlists}
        loading={playlistsLoading}
        errorMessage={playlistError}
        busy={playlistBusy}
        onCreate={handleCreatePlaylist}
        onRemoveTrack={handleRemoveTrackFromPlaylist}
        onOpenPlaylist={handleOpenPlaylist}
        onRefresh={refreshPlaylists}
      />
    </section>
  );

  const homeContent = (
    <HomeHub
      hasSearchResults={searchResults.length > 0}
      lastQuery={lastQuery}
      playlistCount={playlists.length}
      selectedCount={selectedTracks.length}
      onOpenSearch={() => handleTabChange("search")}
      onOpenPlaylists={() => handleTabChange("playlists")}
      onResumeQueue={handleResumeQueue}
    />
  );

  return (
    <div
      className="app-shell"
      data-has-selection={hasActiveMiniPlayer ? "true" : "false"}
      data-expanded-mini-player={expandedSource ? "true" : "false"}
      style={shellStyle}
    >
      <main className="shell-tab-panel shell-stage" aria-label="Main content">
        <ShellTabContent activeTab={activeTab} home={homeContent} search={searchContent} playlists={playlistsContent} />
      </main>

      {renderedMiniPlayerPresentation ? (
        <ExpandedMiniPlayerOwnerProvider
          value={expandedSource ? {
            tracks: expandedTracks,
            onCommitTemporaryOrder: handleCommitTemporaryOrder,
            onCommitPersistedOrder: handleCommitPersistedOrder,
            onDeleteTemporary: handleDeleteTemporaryFromExpanded,
            onDeletePersisted: handleDeletePersistedFromExpanded
          } : null}
        >
          <MiniPlayer
            {...renderedMiniPlayerPresentation}
            tracks={expandedTracks}
            isExpanded={Boolean(expandedSource)}
            busy={finalizeBusy}
            lifecycle={exitingMiniPlayerPresentation && !miniPlayerPresentation ? idleMiniPlayerLifecycle : finalizationLifecycle}
            isExiting={Boolean(exitingMiniPlayerPresentation && !miniPlayerPresentation)}
            onRequestExpand={handleMiniPlayerExpansionIntent}
            onRequestCollapse={handleMiniPlayerCollapse}
            onExitComplete={() => setExitingMiniPlayerPresentation(null)}
          />
        </ExpandedMiniPlayerOwnerProvider>
      ) : null}

      <BottomNav activeTab={activeTab} onChange={handleTabChange} />

      {successState ? (
        <StatusBanner
          title="Playlist concluída"
          message="Sua playlist está pronta."
          variant="success"
          actionLabel="Abrir playlist"
          onAction={() => {
            setExpandedSource({ kind: "playlist", playlist: successState.playlist });
            setSuccessState(null);
          }}
        />
      ) : null}

      <FinalizeCart
        open={finalizeOpen}
        locked={finalizeBusy}
        selectedTracks={selectedTracks}
        playlists={playlists}
        onClose={() => setFinalizeOpen(false)}
        onConfirm={handleFinalize}
      />
    </div>
  );
}

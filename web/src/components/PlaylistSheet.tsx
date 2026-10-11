import type { Playlist } from "../types";
import { Sheet } from "./Sheet";
import { PlaylistPanel } from "./PlaylistPanel";

export type PlaylistSheetMode = "closed" | "compact" | "expanded";

type PlaylistSheetProps = {
  mode: PlaylistSheetMode;
  locked?: boolean;
  playlists: Playlist[];
  loading?: boolean;
  errorMessage?: string | null;
  busy?: boolean;
  onCreate: (name: string) => void;
  onRemoveTrack: (playlistId: string, videoId: string) => void;
  onRefresh: () => void;
  onClose: () => void;
};

export function PlaylistSheet({
  mode,
  locked = false,
  playlists,
  loading = false,
  errorMessage = null,
  busy = false,
  onCreate,
  onRemoveTrack,
  onRefresh,
  onClose
}: PlaylistSheetProps) {
  return (
    <Sheet title="Biblioteca de playlists" open={mode !== "closed"} locked={locked} onClose={onClose}>
      <PlaylistPanel
        playlists={playlists}
        loading={loading}
        errorMessage={errorMessage}
        busy={busy}
        onCreate={onCreate}
        onRemoveTrack={onRemoveTrack}
        onRefresh={onRefresh}
      />
    </Sheet>
  );
}

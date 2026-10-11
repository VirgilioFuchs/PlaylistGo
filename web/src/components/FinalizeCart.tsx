import { useEffect, useMemo, useState } from "react";
import type { Playlist, TrackSnapshot } from "../types";
import { Button } from "./Button";
import { Input } from "./Input";
import { Sheet } from "./Sheet";

type FinalizeTarget =
  | { kind: "new"; name: string }
  | { kind: "existing"; playlistId: string };

type FinalizeCartProps = {
  open: boolean;
  locked?: boolean;
  selectedTracks: TrackSnapshot[];
  playlists: Playlist[];
  onClose: () => void;
  onConfirm: (target: FinalizeTarget) => Promise<void> | void;
};

export function FinalizeCart({
  open,
  locked = false,
  selectedTracks,
  playlists,
  onClose,
  onConfirm
}: FinalizeCartProps) {
  const [targetKind, setTargetKind] = useState<"new" | "existing">("new");
  const [playlistName, setPlaylistName] = useState("");
  const firstPlaylistId = useMemo(() => playlists[0]?.id ?? "", [playlists]);
  const [playlistId, setPlaylistId] = useState(firstPlaylistId);

  useEffect(() => {
    if (!playlistId && firstPlaylistId) {
      setPlaylistId(firstPlaylistId);
    }
  }, [firstPlaylistId, playlistId]);

  if (!open) {
    return null;
  }

  return (
    <Sheet title="Finalizar playlist" open={open} locked={locked} onClose={onClose}>
      <form
        className="finalize-cart"
        onSubmit={(event) => {
          event.preventDefault();
          if (targetKind === "new") {
            void onConfirm({ kind: "new", name: playlistName });
            return;
          }
          void onConfirm({ kind: "existing", playlistId });
        }}
      >
        <section className="finalize-cart__summary">
          <p className="section-kicker">Finalizar</p>
          <h2>Escolha para onde as faixas selecionadas devem ir.</h2>
          <p>{selectedTracks.length} faixas selecionadas.</p>
        </section>

        <div className="finalize-cart__choices" role="radiogroup" aria-label="Playlist de destino">
          <label className={`finalize-cart__choice${targetKind === "new" ? " is-active" : ""}`}>
            <input
              type="radio"
              name="finalize-target"
              value="new"
              checked={targetKind === "new"}
              onChange={() => setTargetKind("new")}
            />
            <span>Criar uma playlist nova</span>
          </label>
          <label className={`finalize-cart__choice${targetKind === "existing" ? " is-active" : ""}`}>
            <input
              type="radio"
              name="finalize-target"
              value="existing"
              checked={targetKind === "existing"}
              onChange={() => {
                setTargetKind("existing");
                setPlaylistId((current) => current || firstPlaylistId);
              }}
            />
            <span>Adicionar a uma playlist existente</span>
          </label>
        </div>

        {targetKind === "new" ? (
          <Input
            label="Nome da playlist"
            type="text"
            value={playlistName}
            onChange={(event) => setPlaylistName(event.target.value)}
            placeholder="Nome da playlist"
            autoComplete="off"
            disabled={locked}
          />
        ) : (
          <label className="finalize-cart__field">
            <span>Playlist existente</span>
            <select
              value={playlistId}
              onChange={(event) => setPlaylistId(event.target.value)}
              disabled={locked || playlists.length === 0}
            >
              {playlists.length === 0 ? <option value="">Nenhuma playlist disponivel</option> : null}
              {playlists.map((playlist) => (
                <option key={playlist.id} value={playlist.id}>
                  {playlist.name}
                </option>
              ))}
            </select>
          </label>
        )}

        <footer className="finalize-cart__footer">
          <Button type="button" variant="secondary" onClick={onClose} disabled={locked}>
            Cancelar
          </Button>
          <Button
            type="submit"
            loading={locked}
            loadingLabel="Finalizando..."
            disabled={locked || selectedTracks.length === 0 || (targetKind === "existing" && playlists.length === 0)}
          >
            Finalizar
          </Button>
        </footer>
      </form>
    </Sheet>
  );
}

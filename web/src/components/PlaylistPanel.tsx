import { useMemo, useRef } from "react";
import type { Playlist } from "../types";
import { formatCountLabel, formatDate } from "../lib/format";
import { Button } from "./Button";
import { PlaylistForm } from "./PlaylistForm";
import { SurfaceSection } from "./SurfaceSection";

type PlaylistPanelProps = {
  playlists: Playlist[];
  loading?: boolean;
  errorMessage?: string | null;
  busy?: boolean;
  onCreate: (name: string) => void;
  onRemoveTrack: (playlistId: string, videoId: string) => void;
  onOpenPlaylist?: (playlist: Playlist) => void;
  onRefresh: () => void;
};

function buildPlaylistTone(index: number): string {
  const tones = ["sea", "gold", "ember", "berry"];
  return tones[index % tones.length];
}

export function PlaylistPanel({
  playlists,
  loading = false,
  errorMessage = null,
  busy = false,
  onCreate,
  onOpenPlaylist,
  onRefresh
}: PlaylistPanelProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const playlistSummary = useMemo(() => {
    const trackCount = playlists.reduce((total, playlist) => total + playlist.trackCount, 0);
    return {
      playlistCount: playlists.length,
      trackCount
    };
  }, [playlists]);

  function focusCreateInput() {
    inputRef.current?.focus();
  }

  return (
    <SurfaceSection
      className="playlist-panel"
      eyebrow="Playlists"
      title="Biblioteca pronta para continuar sessao, nao para administrar tabela."
      description="Cada playlist vira card de colecao com capa, contexto e clima de player."
      actions={
        <Button type="button" variant="secondary" onClick={onRefresh} disabled={loading}>
          Atualizar
        </Button>
      }
    >
      <div className="playlist-library__hero">
        <div className="playlist-library__hero-copy">
          <p className="shell-kicker">Deck da biblioteca</p>
          <h3>Playlists precisam parecer colecao viva, nao lista utilitaria.</h3>
          <p>Crie rapido, volte para busca depois e mantenha a biblioteca pronta para proxima acao.</p>
          <div className="playlist-library__hero-stats" aria-label="Resumo da biblioteca">
            <article className="playlist-library__hero-pill">
              <span>Playlists</span>
              <strong>{playlistSummary.playlistCount}</strong>
            </article>
            <article className="playlist-library__hero-pill">
              <span>Faixas salvas</span>
              <strong>{playlistSummary.trackCount}</strong>
            </article>
          </div>
        </div>
        <PlaylistForm onCreate={onCreate} busy={busy} inputRef={inputRef} />
      </div>

      {errorMessage ? <p className="playlist-panel__message playlist-panel__message--error">{errorMessage}</p> : null}
      {loading ? <p className="playlist-panel__message">Carregando playlists...</p> : null}

      {!loading && playlists.length === 0 ? (
        <section className="playlist-library__empty" aria-label="Nenhuma playlist">
          <div className="playlist-library__cover playlist-library__cover--empty" aria-hidden="true">
            <span>First Drop</span>
          </div>
          <div className="playlist-library__empty-copy">
            <p className="shell-kicker">Biblioteca vazia</p>
            <h3>Crie primeira playlist para transformar busca em estante jogavel.</h3>
            <p>Comece com nome simples, depois mande a fila para dentro sem perder navegacao nem contexto.</p>
            <div className="playlist-library__empty-actions">
              <Button type="button" onClick={focusCreateInput} disabled={busy}>
                Criar primeira playlist
              </Button>
              <p>CTA abre o ponto de criacao logo acima.</p>
            </div>
          </div>
        </section>
      ) : null}

      <div className="playlist-library__grid">
        {playlists.map((playlist, index) => (
          <article className="playlist-library__card" key={playlist.id}>
            <div className={`playlist-library__cover playlist-library__cover--${buildPlaylistTone(index)}`} aria-hidden="true">
              {playlist.trackCount === 0 ? <span className="playlist-library__cover-empty-copy">Sem faixa</span> : null}
            </div>

            <div className="playlist-library__body">
              <div className="playlist-library__footer">
                <div className="playlist-library__title-block">
                  <h3>{playlist.name}</h3>
                  <span className="playlist-library__count">{formatCountLabel(playlist.trackCount, "faixa", "faixas")}</span>
                </div>
                <span className="playlist-library__stamp">{formatDate(playlist.createdAt)}</span>
              </div>
              {onOpenPlaylist ? (
                <Button type="button" variant="secondary" onClick={() => onOpenPlaylist(playlist)} disabled={busy}>
                  Abrir playlist
                </Button>
              ) : null}
            </div>
          </article>
        ))}
      </div>
    </SurfaceSection>
  );
}

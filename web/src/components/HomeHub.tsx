type HomeHubProps = {
  hasSearchResults: boolean;
  lastQuery: string;
  playlistCount: number;
  selectedCount: number;
  onOpenSearch: () => void;
  onOpenPlaylists: () => void;
  onResumeQueue: () => void;
};

export function HomeHub({
  hasSearchResults,
  lastQuery,
  playlistCount,
  selectedCount,
  onOpenSearch,
  onOpenPlaylists,
  onResumeQueue
}: HomeHubProps) {
  const hasRecentQuery = lastQuery.length > 0;

  return (
    <section className="home-hub" aria-labelledby="home-hub-title">
      <div className="home-hub__hero shell-band shell-band--accent">
        <div className="home-hub__hero-copy">
          <p className="shell-band__eyebrow">Home</p>
          <h2 id="home-hub-title">Continue de onde você parou.</h2>
          <p className="shell-band__lede">
            Busca, fila e biblioteca ficam acessíveis sem trocar de contexto nem perder o que já estava em andamento.
          </p>
        </div>
        <div className="home-hub__actions">
          <Button type="button" onClick={onOpenSearch}>
            {hasRecentQuery ? "Retomar busca" : "Começar busca"}
          </Button>
        </div>
        <div className="home-hub__quick-actions" aria-label="Ações rápidas">
          <Button type="button" variant="secondary" onClick={onResumeQueue} disabled={selectedCount === 0}>
            Rever fila
          </Button>
          <Button type="button" variant="secondary" onClick={onOpenPlaylists}>
            Abrir biblioteca
          </Button>
        </div>
      </div>

      <div className="home-hub__grid">
        <article className="home-hub__card home-hub__card--primary">
          <p className="shell-kicker">Busca recente</p>
          <h3>{hasRecentQuery ? `"${lastQuery}"` : "Nenhuma busca recente"}</h3>
          <p>{hasSearchResults ? "Resultados prontos para voltar ao fluxo de descoberta." : "A primeira busca vai abrir o fluxo principal do app."}</p>
          <Button type="button" variant="secondary" onClick={onOpenSearch}>
            {hasRecentQuery ? "Abrir Search" : "Ir para Search"}
          </Button>
        </article>

        <article className="home-hub__card">
          <p className="shell-kicker">Fila atual</p>
          <h3>{selectedCount} faixas selecionadas</h3>
          <p>A fila precisa continuar visível e acessível sem dominar a shell inteira.</p>
          <Button
            type="button"
            variant="secondary"
            onClick={onResumeQueue}
            disabled={selectedCount === 0}
          >
            Revisar fila
          </Button>
        </article>

        <article className="home-hub__card">
          <p className="shell-kicker">Biblioteca</p>
          <h3>{playlistCount} playlists salvas</h3>
          <p>A biblioteca fica isolada para manter criação, revisão e manutenção no mesmo lugar.</p>
          <Button type="button" variant="secondary" onClick={onOpenPlaylists}>
            Ver biblioteca
          </Button>
        </article>
      </div>
    </section>
  );
}
import { Button } from "./Button";

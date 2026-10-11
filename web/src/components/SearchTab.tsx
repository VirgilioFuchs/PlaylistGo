import { Button } from "./Button";
import { SearchForm } from "./SearchForm";
import { SearchResults } from "./SearchResults";
import { StatusBanner } from "./StatusBanner";
import type { SearchCacheStatus, SearchQuota, TrackSnapshot } from "../types";

export type SearchTabState =
  | "idle"
  | "loading"
  | "success"
  | "empty"
  | "validation_error"
  | "quota_limited"
  | "temporary_error";

export type SearchTabMessage = {
  title: string;
  message: string;
  variant: "info" | "success" | "warning" | "error";
};

type SearchTabProps = {
  query: string;
  lastQuery: string;
  state: SearchTabState;
  message: SearchTabMessage;
  tracks: TrackSnapshot[];
  selectedIds: Set<string>;
  history: string[];
  locked?: boolean;
  quota: SearchQuota | null;
  cache: SearchCacheStatus | null;
  quotaAcknowledged: boolean;
  quotaDetailsOpen: boolean;
  onQueryChange: (value: string) => void;
  onSearch: (query: string) => void;
  onRetry: () => void;
  onHistorySearch: (entry: string) => void;
  onClearHistory: () => void;
  onToggleTrack: (track: TrackSnapshot) => void;
  onAcknowledgeQuota: () => void;
  onToggleQuotaDetails: () => void;
};

export function SearchTab({
  query,
  lastQuery,
  state,
  message,
  tracks,
  selectedIds,
  history,
  locked = false,
  quota,
  cache,
  quotaAcknowledged,
  quotaDetailsOpen,
  onQueryChange,
  onSearch,
  onRetry,
  onHistorySearch,
  onClearHistory,
  onToggleTrack,
  onAcknowledgeQuota,
  onToggleQuotaDetails
}: SearchTabProps) {
  const showIntro = state === "idle" && tracks.length === 0;
  const showBanner = state === "validation_error" || state === "temporary_error" || state === "quota_limited";
  const showResults = !showIntro;

  return (
    <section className="shell-main" aria-label="Pesquisa e conteúdo">
      <section className="search-hub shell-band shell-band--accent" aria-labelledby="search-title">
        <div className="shell-band__copy">
          <p className="shell-band__eyebrow">Search</p>
          <h2 id="search-title">Descoberta, histórico e resultados no mesmo fluxo.</h2>
          <p className="shell-band__lede">
            A busca precisa ser retomável: consulte de novo, refine a fila e continue sem perder contexto.
          </p>
        </div>
        <SearchForm value={query} loading={state === "loading"} onChange={onQueryChange} onSubmit={onSearch} />
      </section>

      <section className="search-history" aria-label="Histórico de busca">
        <div className="search-history__header">
          <div>
            <p className="shell-kicker">Recentes</p>
            <h3>Reabra consultas sem redigitar.</h3>
          </div>
          {history.length > 0 ? (
            <div className="search-history__actions">
              <span>{history.length} salvas</span>
              <Button type="button" variant="secondary" className="search-history__clear" onClick={onClearHistory}>
                Limpar
              </Button>
            </div>
          ) : null}
        </div>

        {history.length > 0 ? (
          <div className="search-history__chips">
            {history.map((entry) => (
              <Button
                key={entry}
                type="button"
                variant="secondary"
                className="search-history__chip"
                selected={entry === lastQuery}
                onClick={() => onHistorySearch(entry)}
              >
                {entry}
              </Button>
            ))}
          </div>
        ) : (
          <p className="search-history__empty">As próximas buscas aparecem aqui para você retomar rápido.</p>
        )}
      </section>

      {showIntro ? (
        <section className="shell-intro search-intro" aria-label="Introdução">
          <div className="shell-intro__card">
            <p className="shell-kicker">Comece aqui</p>
            <h2>Pesquise uma faixa para abrir o fluxo de descoberta.</h2>
            <p>Depois da primeira busca, este tab passa a concentrar histórico, resultados e retomada.</p>
          </div>
        </section>
      ) : null}

      {showBanner ? (
        <StatusBanner
          title={message.title}
          message={message.message}
          variant={message.variant}
          actionLabel={state === "temporary_error" ? "Tentar de novo" : undefined}
          onAction={state === "temporary_error" ? onRetry : undefined}
          secondaryActionLabel={state === "quota_limited" && quotaAcknowledged ? "Ver detalhe" : undefined}
          onSecondaryAction={state === "quota_limited" && quotaAcknowledged ? onToggleQuotaDetails : undefined}
        >
          {state === "quota_limited" && !quotaAcknowledged ? (
            <Button type="button" variant="secondary" onClick={onAcknowledgeQuota}>
              Entendi
            </Button>
          ) : null}
          {state === "quota_limited" && quotaAcknowledged ? (
            <p className="status-banner__detail">
              {quotaDetailsOpen
                ? "O limite é controlado pelo orçamento do backend. Resultados em cache continuam disponíveis."
                : "Limite reconhecido. Você pode seguir navegando ou abrir a explicação."}
            </p>
          ) : null}
        </StatusBanner>
      ) : null}

      {state === "success" && quota ? (
        <p className="search-meta">
          {cache?.hit ? "Resultados em cache" : "Resultados novos"} | {quota.remaining} buscas restantes hoje
        </p>
      ) : null}

      {showResults ? (
        <SearchResults
          tracks={tracks}
          selectedIds={selectedIds}
          loading={state === "loading"}
          locked={locked}
          title={lastQuery ? `Resultados para "${lastQuery}"` : "Resultados de busca"}
          description="Cards mantêm descoberta legível enquanto a fila continua fixa no shell."
          emptyMessage={state === "empty" ? "Nenhuma faixa correspondeu a essa busca." : "Pesquise uma faixa para começar."}
          onToggle={onToggleTrack}
        />
      ) : null}
    </section>
  );
}

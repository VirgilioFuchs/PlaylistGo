import type {
  ApiErrorCode,
  ApiErrorPayload,
  ApiErrorResponse,
  Playlist,
  PlaylistListResponse,
  PlaylistSort,
  SearchQuota,
  SearchResponse,
  TrackSnapshot
} from "../types";

type ViteEnv = {
  VITE_API_BASE_URL?: string;
};

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
};

const DEFAULT_API_BASE_URL = "";
const JSON_CONTENT_TYPE = "application/json";

export class ApiClientError extends Error {
  readonly status: number;
  readonly code: ApiErrorCode;
  readonly error: ApiErrorPayload;
  readonly quota?: SearchQuota;

  constructor(status: number, error: ApiErrorPayload, quota?: SearchQuota) {
    super(error.message);
    this.name = "ApiClientError";
    this.status = status;
    this.code = error.code;
    this.error = error;
    this.quota = quota;
  }
}

export function searchTracks(query: string): Promise<SearchResponse> {
  const params = new URLSearchParams({ q: query });
  return request<SearchResponse>(`/api/search?${params.toString()}`);
}

export function listPlaylists(sort?: PlaylistSort): Promise<PlaylistListResponse> {
  const params = sort ? `?${new URLSearchParams({ sort }).toString()}` : "";
  return request<PlaylistListResponse>(`/api/playlists${params}`);
}

export function createPlaylist(name: string): Promise<Playlist> {
  return request<Playlist>("/api/playlists", {
    method: "POST",
    body: { name }
  });
}

export function addTrackToPlaylist(
  playlistId: string,
  track: TrackSnapshot
): Promise<Playlist> {
  return request<Playlist>(`/api/playlists/${encodePathSegment(playlistId)}/tracks`, {
    method: "POST",
    body: track
  });
}

export function removeTrackFromPlaylist(
  playlistId: string,
  videoId: string
): Promise<Playlist> {
  return request<Playlist>(
    `/api/playlists/${encodePathSegment(playlistId)}/tracks/${encodePathSegment(videoId)}`,
    { method: "DELETE" }
  );
}

export function reorderPlaylistTracks(
  playlistId: string,
  trackIds: string[],
  expectedRevision: number
): Promise<Playlist> {
  return request<Playlist>(
    `/api/playlists/${encodePathSegment(playlistId)}/tracks/order`,
    {
      method: "PUT",
      body: { trackIds, expectedRevision }
    }
  );
}

export function getApiBaseUrl(): string {
  const env = (import.meta as ImportMeta & { env?: ViteEnv }).env;
  return trimTrailingSlash(env?.VITE_API_BASE_URL || DEFAULT_API_BASE_URL);
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const response = await fetch(`${getApiBaseUrl()}${path}`, {
    method: options.method || "GET",
    headers: options.body === undefined ? undefined : { "Content-Type": JSON_CONTENT_TYPE },
    body: options.body === undefined ? undefined : JSON.stringify(options.body)
  });

  const payload = await parseJSON(response);
  if (!response.ok) {
    throw createApiError(response.status, payload);
  }

  return payload as T;
}

async function parseJSON(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) {
    return null;
  }

  try {
    return JSON.parse(text) as unknown;
  } catch {
    return null;
  }
}

function createApiError(status: number, payload: unknown): ApiClientError {
  if (isApiErrorResponse(payload)) {
    return new ApiClientError(status, payload.error, payload.quota);
  }

  return new ApiClientError(status, {
    code: "upstream_unavailable",
    message: "Servico indisponivel no momento."
  });
}

function isApiErrorResponse(payload: unknown): payload is ApiErrorResponse {
  if (!payload || typeof payload !== "object" || !("error" in payload)) {
    return false;
  }

  const error = (payload as { error?: unknown }).error;
  if (!error || typeof error !== "object") {
    return false;
  }

  const code = (error as { code?: unknown }).code;
  const message = (error as { message?: unknown }).message;
  return isApiErrorCode(code) && typeof message === "string";
}

function isApiErrorCode(value: unknown): value is ApiErrorCode {
  return (
    value === "validation_error" ||
    value === "quota_limited" ||
    value === "conflict" ||
    value === "not_found" ||
    value === "upstream_unavailable" ||
    value === "method_not_allowed" ||
    value === "internal_error"
  );
}

function encodePathSegment(value: string): string {
  return encodeURIComponent(value);
}

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, "");
}

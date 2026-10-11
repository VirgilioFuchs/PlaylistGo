export type Thumbnail = {
  url: string;
  width?: number;
  height?: number;
};

export type ThumbnailVariants = {
  default?: Thumbnail;
  medium?: Thumbnail;
  high?: Thumbnail;
  standard?: Thumbnail;
  maxres?: Thumbnail;
};

export type TrackSnapshot = {
  videoId: string;
  title: string;
  channel: string;
  durationSeconds: number;
  thumbnailUrl: string;
  thumbnails?: ThumbnailVariants;
  addedAt?: string;
};

export type SearchQuota = {
  dailyLimit: number;
  remaining: number;
  usedPercent: number;
  alertLevel: string;
};

export type SearchCacheStatus = {
  hit: boolean;
};

export type SearchResponse = {
  data: TrackSnapshot[];
  quota: SearchQuota;
  cache: SearchCacheStatus;
};

export type Playlist = {
  id: string;
  name: string;
  trackCount: number;
  durationSeconds: number;
  tracks?: TrackSnapshot[];
  revision: number;
  createdAt: string;
  updatedAt: string;
};

export type PlaylistSort = "recent" | "name" | "updated";

export type PlaylistListResponse = {
  data: Playlist[];
};

export type ApiErrorCode =
  | "validation_error"
  | "quota_limited"
  | "conflict"
  | "not_found"
  | "upstream_unavailable"
  | "method_not_allowed"
  | "internal_error";

export type ApiErrorPayload = {
  code: ApiErrorCode;
  message: string;
};

export type ApiErrorResponse = {
  error: ApiErrorPayload;
  quota?: SearchQuota;
};

export type ApiError = {
  status: number;
  code: ApiErrorCode;
  message: string;
  quota?: SearchQuota;
};
